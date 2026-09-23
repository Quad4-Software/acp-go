// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/Quad4-Software/acp-go/transport"
)

// ErrClosed is returned by calls made after the connection is closed.
var ErrClosed = errors.New("jsonrpc: connection closed")

// Handler processes an incoming request and returns its result.
// Returning a nil result produces a JSON-RPC result of null.
// Return an *Error to control the response code and data.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// NotificationHandler processes an incoming notification. Handlers run
// sequentially in the order notifications arrive. A handler may make
// outbound calls on the same connection.
type NotificationHandler func(ctx context.Context, params json.RawMessage)

// Conn is a bidirectional JSON-RPC connection over a Transport. It
// dispatches inbound requests and notifications to registered handlers
// and correlates outbound calls with their responses.
type Conn struct {
	t transport.Transport

	mu            sync.Mutex
	pending       map[string]chan rpcOutcome
	inflight      map[string]context.CancelFunc
	reqHandlers   map[string]Handler
	notifHandlers map[string]NotificationHandler
	fallback      Handler
	notifFallback NotificationHandler
	notifyCh      chan rawMessage

	idCtr     atomic.Int64
	closed    chan struct{}
	closeOnce sync.Once
	runOnce   sync.Once
}

type rpcOutcome struct {
	result json.RawMessage
	err    *Error
}

// NewConn returns a connection on transport t.
func NewConn(t transport.Transport) *Conn {
	return &Conn{
		t:             t,
		pending:       make(map[string]chan rpcOutcome),
		inflight:      make(map[string]context.CancelFunc),
		reqHandlers:   make(map[string]Handler),
		notifHandlers: make(map[string]NotificationHandler),
		notifyCh:      make(chan rawMessage, 64),
		closed:        make(chan struct{}),
	}
}

// Handle registers h for requests to method. A nil h unregisters the
// method. Register before starting Run.
func (c *Conn) Handle(method string, h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h == nil {
		delete(c.reqHandlers, method)
	} else {
		c.reqHandlers[method] = h
	}
}

// HandleNotification registers h for notifications to method. A nil h
// unregisters the method. Register before starting Run.
func (c *Conn) HandleNotification(method string, h NotificationHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h == nil {
		delete(c.notifHandlers, method)
	} else {
		c.notifHandlers[method] = h
	}
}

// SetFallback registers a handler for requests to unregistered methods,
// including extension methods prefixed with an underscore. Without a
// fallback such requests fail with ErrCodeMethodNotFound.
func (c *Conn) SetFallback(h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallback = h
}

// SetNotificationFallback registers a handler for notifications to
// unregistered methods.
func (c *Conn) SetNotificationFallback(h NotificationHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifFallback = h
}

// Call sends a request and waits for its response. The result is
// unmarshaled into out, which may be nil or a *json.RawMessage to keep
// the raw result. If ctx is canceled while waiting, a $/cancel_request
// notification is sent to the peer and ctx.Err is returned.
func (c *Conn) Call(ctx context.Context, method string, params, out any) error {
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("jsonrpc: marshal params: %w", err)
		}
		rawParams = b
	}

	id := IntID(c.idCtr.Add(1))
	key := id.String()
	ch := make(chan rpcOutcome, 1)
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return ErrClosed
	default:
	}
	c.pending[key] = ch
	c.mu.Unlock()

	req := rpcRequest{JSONRPC: JSONRPCVersion, ID: id, Method: method, Params: rawParams}
	if err := c.write(req); err != nil {
		c.dropPending(key)
		return err
	}

	select {
	case oc := <-ch:
		if oc.err != nil {
			return oc.err
		}
		if out == nil || len(oc.result) == 0 {
			return nil
		}
		if raw, ok := out.(*json.RawMessage); ok {
			*raw = oc.result
			return nil
		}
		if err := json.Unmarshal(oc.result, out); err != nil {
			return fmt.Errorf("jsonrpc: decode %s result: %w", method, err)
		}
		return nil
	case <-ctx.Done():
		c.dropPending(key)
		_ = c.sendCancelRequest(id)
		return ctx.Err()
	case <-c.closed:
		return ErrClosed
	}
}

// Notify sends a notification. Notifications never produce a response.
func (c *Conn) Notify(ctx context.Context, method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("jsonrpc: marshal params: %w", err)
		}
		rawParams = b
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.write(rpcNotification{JSONRPC: JSONRPCVersion, Method: method, Params: rawParams})
}

// CancelRequest sends a $/cancel_request notification asking the peer to
// abort processing the request with the given ID. Callers normally do
// not need this: canceling the context of an in-flight Call sends the
// notification automatically.
func (c *Conn) CancelRequest(ctx context.Context, id RequestID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.sendCancelRequest(id)
}

func (c *Conn) sendCancelRequest(id RequestID) error {
	params, err := json.Marshal(CancelRequestNotification{RequestID: id})
	if err != nil {
		return err
	}
	return c.write(rpcNotification{JSONRPC: JSONRPCVersion, Method: MethodCancelRequest, Params: params})
}

// Run reads and dispatches messages until the transport fails, ctx is
// canceled, or Close is called. It returns nil on clean peer EOF.
// Exactly one goroutine may call Run.
func (c *Conn) Run(ctx context.Context) error {
	var ret error
	c.runOnce.Do(func() {
		go c.notifyWorker()
		ret = c.readLoop(ctx)
		c.shutdown()
	})
	return ret
}

func (c *Conn) readLoop(ctx context.Context) error {
	for {
		raw, err := c.t.Read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil ||
				errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
		var m rawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			c.respond(json.RawMessage("null"), nil,
				NewError(ErrCodeParseError, "invalid JSON-RPC message"))
			continue
		}
		switch {
		case m.isRequest():
			c.dispatchRequest(m)
		case m.isNotification():
			c.dispatchNotification(m)
		case m.isResponse():
			c.dispatchResponse(m)
		}
	}
}

// dispatchRequest runs the handler in a goroutine and sends the
// response when it returns.
func (c *Conn) dispatchRequest(m rawMessage) {
	var id RequestID
	if err := id.UnmarshalJSON(*m.ID); err != nil {
		c.respond(*m.ID, nil, NewError(ErrCodeInvalidRequest, "invalid request id"))
		return
	}
	key := id.String()

	c.mu.Lock()
	h := c.reqHandlers[m.Method]
	fb := c.fallback
	c.mu.Unlock()
	if h == nil {
		h = fb
	}
	if h == nil {
		c.respond(*m.ID, nil, MethodNotFound(m.Method))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		cancel()
		return
	default:
	}
	c.inflight[key] = cancel
	c.mu.Unlock()

	go func() {
		defer func() {
			c.mu.Lock()
			delete(c.inflight, key)
			c.mu.Unlock()
			cancel()
		}()
		result, err := h(ctx, m.Params)
		if err != nil {
			var e *Error
			if !errors.As(err, &e) {
				e = NewError(ErrCodeInternalError, err.Error())
			}
			c.respond(*m.ID, nil, e)
			return
		}
		var raw json.RawMessage
		if result != nil {
			if rm, ok := result.(json.RawMessage); ok {
				raw = rm
			} else {
				b, merr := json.Marshal(result)
				if merr != nil {
					c.respond(*m.ID, nil, NewErrorf(ErrCodeInternalError, "marshal result: %v", merr))
					return
				}
				raw = b
			}
		}
		c.respond(*m.ID, raw, nil)
	}()
}

// dispatchNotification queues the notification for ordered handling.
// The $/cancel_request notification is handled internally.
func (c *Conn) dispatchNotification(m rawMessage) {
	if m.Method == MethodCancelRequest {
		var n CancelRequestNotification
		if json.Unmarshal(m.Params, &n) == nil {
			c.mu.Lock()
			if cancel, ok := c.inflight[n.RequestID.String()]; ok {
				cancel()
			}
			c.mu.Unlock()
		}
		return
	}
	select {
	case c.notifyCh <- m:
	case <-c.closed:
	}
}

func (c *Conn) notifyWorker() {
	for {
		select {
		case m := <-c.notifyCh:
			c.mu.Lock()
			h := c.notifHandlers[m.Method]
			fb := c.notifFallback
			c.mu.Unlock()
			if h == nil {
				h = fb
			}
			if h != nil {
				h(context.Background(), m.Params)
			}
		case <-c.closed:
			return
		}
	}
}

func (c *Conn) dispatchResponse(m rawMessage) {
	var id RequestID
	if err := id.UnmarshalJSON(*m.ID); err != nil {
		return
	}
	key := id.String()
	c.mu.Lock()
	ch, ok := c.pending[key]
	if ok {
		delete(c.pending, key)
	}
	c.mu.Unlock()
	if ok {
		ch <- rpcOutcome{result: m.Result, err: m.Err}
	}
}

func (c *Conn) respond(id json.RawMessage, result json.RawMessage, e *Error) {
	_ = c.write(rpcResponse{ID: id, Result: result, Err: e})
}

func (c *Conn) write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case <-c.closed:
		return ErrClosed
	default:
	}
	return c.t.Write(context.Background(), b)
}

func (c *Conn) dropPending(key string) {
	c.mu.Lock()
	delete(c.pending, key)
	c.mu.Unlock()
}

func (c *Conn) shutdown() {
	c.mu.Lock()
	for key, ch := range c.pending {
		ch <- rpcOutcome{err: NewError(ErrCodeInternalError, "connection closed")}
		delete(c.pending, key)
	}
	for key, cancel := range c.inflight {
		cancel()
		delete(c.inflight, key)
	}
	c.mu.Unlock()
	_ = c.Close()
}

// Close shuts down the connection and its transport. Pending calls fail
// and in-flight handler contexts are canceled.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.t.Close()
	})
	return err
}

// Done returns a channel closed when the connection is closed.
func (c *Conn) Done() <-chan struct{} { return c.closed }
