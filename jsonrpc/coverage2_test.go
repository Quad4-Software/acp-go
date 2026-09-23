// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorAPI(t *testing.T) {
	e := NewError(ErrCodeInvalidParams, "bad")
	if e.Error() == "" {
		t.Fatal("empty Error()")
	}
	// Unmarshalable data leaves the error unchanged.
	if got := e.WithData(make(chan int)); got.Data != nil {
		t.Fatal("WithData should no-op on marshal failure")
	}
	got := e.WithData(map[string]int{"x": 1})
	if got.Data == nil {
		t.Fatal("WithData lost")
	}
	if MethodNotFound("m").Code != ErrCodeMethodNotFound {
		t.Fatal("MethodNotFound code")
	}
	if InvalidParams("p").Code != ErrCodeInvalidParams {
		t.Fatal("InvalidParams code")
	}
	if ErrorCodeOf(NewError(ErrCodeParseError, "x")) != ErrCodeParseError {
		t.Fatal("ErrorCodeOf")
	}
	if !IsErrorCode(e, ErrCodeInvalidParams) || IsErrorCode(errors.New("x"), ErrCodeParseError) {
		t.Fatal("IsErrorCode")
	}
}

func TestCallMarshalErrors(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	runConn(t, a)
	runConn(t, b)

	if err := a.Call(context.Background(), "m", make(chan int), nil); err == nil {
		t.Fatal("unmarshalable params accepted")
	}
	if err := a.Notify(context.Background(), "m", make(chan int)); err == nil {
		t.Fatal("unmarshalable notify params accepted")
	}
}

func TestCallsAfterClose(t *testing.T) {
	ta, _ := pipePair()
	a := NewConn(ta)
	_ = a.Close()
	if err := a.Call(context.Background(), "m", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
	if err := a.Notify(context.Background(), "m", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
}

func TestHandleUnregister(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("m", func(context.Context, json.RawMessage) (any, error) { return 1, nil })
	runConn(t, a)
	runConn(t, b)
	b.Handle("m", nil)

	err := a.Call(context.Background(), "m", nil, nil)
	if !IsErrorCode(err, ErrCodeMethodNotFound) {
		t.Fatalf("got %v, want method not found", err)
	}
}

func TestInvalidRequestIDResponse(t *testing.T) {
	ta, tb := pipePair()
	a := NewConn(ta)
	runConn(t, a)

	if err := tb.Write(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":true,"method":"m"}`)); err != nil {
		t.Fatal(err)
	}
	resp, err := tb.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Error *Error `json:"error"`
	}
	if err := json.Unmarshal(resp, &m); err != nil || m.Error == nil {
		t.Fatalf("expected invalid request error: %s", resp)
	}
	if m.Error.Code != ErrCodeInvalidRequest {
		t.Fatalf("code %v", m.Error.Code)
	}
}

func TestUnknownResponseIDIgnored(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	runConn(t, a)
	runConn(t, b)
	// Responses for requests we never sent are dropped without harm.
	if err := tb.Write(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":999,"result":null}`)); err != nil {
		t.Fatal(err)
	}
	if err := tb.Write(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":"nope","result":{}}`)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := a.Call(context.Background(), "x", nil, nil); !IsErrorCode(err, ErrCodeMethodNotFound) {
		t.Fatalf("conn still works: %v", err)
	}
}

func TestHandlerResultMarshalError(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("bad", func(context.Context, json.RawMessage) (any, error) {
		return make(chan int), nil
	})
	runConn(t, a)
	runConn(t, b)
	err := a.Call(context.Background(), "bad", nil, nil)
	if !IsErrorCode(err, ErrCodeInternalError) {
		t.Fatalf("got %v, want internal error", err)
	}
}

func TestCancelRequestDelivers(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	got := make(chan struct{}, 1)
	b.Handle("slow", func(ctx context.Context, _ json.RawMessage) (any, error) {
		<-ctx.Done()
		close(got)
		return nil, NewError(ErrCodeRequestCancelled, "cancelled")
	})
	runConn(t, a)
	runConn(t, b)

	// Issue a raw request with a known id, then cancel it by id.
	id := IntID(42)
	if err := ta.Write(context.Background(), mustMarshal(t, rpcRequest{
		JSONRPC: JSONRPCVersion, ID: id, Method: "slow",
	})); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := a.CancelRequest(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("handler ctx not canceled")
	}
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRawMessageResult(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("m", func(context.Context, json.RawMessage) (any, error) {
		return json.RawMessage(`{"nested":[1,2]}`), nil
	})
	runConn(t, a)
	runConn(t, b)

	var raw json.RawMessage
	if err := a.Call(context.Background(), "m", nil, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"nested":[1,2]}` {
		t.Fatalf("raw result: %s", raw)
	}
}

func TestNotificationNoHandlerNoPanic(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	runConn(t, a)
	runConn(t, b)
	if err := a.Notify(context.Background(), "nobody", map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	b.HandleNotification("nobody", nil)
	time.Sleep(50 * time.Millisecond)
}
