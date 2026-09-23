// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// pipePair returns two transports connected back to back.
func pipePair() (*LineTransport, *LineTransport) {
	aR, aW := io.Pipe()
	bR, bW := io.Pipe()
	return NewLineTransport(aR, bW, nil), NewLineTransport(bR, aW, nil)
}

func runConn(t *testing.T, c *Conn) {
	t.Helper()
	go func() { _ = c.Run(context.Background()) }()
	t.Cleanup(func() { _ = c.Close() })
}

func TestCallAndResponse(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)

	b.Handle("ping", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{ N int }
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return map[string]int{"pong": p.N + 1}, nil
	})
	runConn(t, a)
	runConn(t, b)

	var out struct {
		Pong int `json:"pong"`
	}
	err := a.Call(context.Background(), "ping", map[string]int{"n": 41}, &out)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if out.Pong != 42 {
		t.Fatalf("got %d, want 42", out.Pong)
	}
}

func TestMethodNotFound(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	runConn(t, a)
	runConn(t, b)

	err := a.Call(context.Background(), "missing", nil, nil)
	if !IsErrorCode(err, ErrCodeMethodNotFound) {
		t.Fatalf("got %v, want method not found", err)
	}
}

func TestErrorPropagation(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("auth", func(context.Context, json.RawMessage) (any, error) {
		return nil, AuthRequired()
	})
	runConn(t, a)
	runConn(t, b)

	err := a.Call(context.Background(), "auth", nil, nil)
	var e *Error
	if !errors.As(err, &e) || e.Code != ErrCodeAuthRequired {
		t.Fatalf("got %v, want auth required", err)
	}
}

func TestCancelRequestPropagates(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)

	started := make(chan struct{})
	b.Handle("slow", func(ctx context.Context, _ json.RawMessage) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, NewError(ErrCodeRequestCancelled, "cancelled")
	})
	runConn(t, a)
	runConn(t, b)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Call(ctx, "slow", nil, nil) }()

	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("call did not return after cancel")
	}
}

func TestNotification(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)

	got := make(chan string, 1)
	b.HandleNotification("note", func(_ context.Context, params json.RawMessage) {
		var p struct{ Text string }
		_ = json.Unmarshal(params, &p)
		got <- p.Text
	})
	runConn(t, a)
	runConn(t, b)

	if err := a.Notify(context.Background(), "note", map[string]string{"text": "hi"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	select {
	case s := <-got:
		if s != "hi" {
			t.Fatalf("got %q, want hi", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("notification not received")
	}
}

func TestNotificationsAreOrdered(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)

	var count atomic.Int64
	seen := make(chan int64, 100)
	b.HandleNotification("seq", func(_ context.Context, params json.RawMessage) {
		var p struct{ N int64 }
		_ = json.Unmarshal(params, &p)
		count.Add(1)
		seen <- p.N
	})
	runConn(t, a)
	runConn(t, b)

	const n = 50
	for i := int64(1); i <= n; i++ {
		if err := a.Notify(context.Background(), "seq", map[string]int64{"n": i}); err != nil {
			t.Fatalf("notify %d: %v", i, err)
		}
	}
	for i := int64(1); i <= n; i++ {
		select {
		case got := <-seen:
			if got != i {
				t.Fatalf("order broken: got %d, want %d", got, i)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("missing notification %d", i)
		}
	}
}

func TestConcurrentCalls(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("echo", func(_ context.Context, params json.RawMessage) (any, error) {
		return params, nil
	})
	runConn(t, a)
	runConn(t, b)

	const n = 32
	errs := make(chan error, n)
	for i := range n {
		go func() {
			var out struct{ I int }
			errs <- a.Call(context.Background(), "echo", map[string]int{"i": i}, &out)
		}()
	}
	for range n {
		if err := <-errs; err != nil {
			t.Fatalf("call: %v", err)
		}
	}
}

func TestCallAfterClose(t *testing.T) {
	ta, _ := pipePair()
	a := NewConn(ta)
	_ = a.Close()
	if err := a.Call(context.Background(), "x", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
}

func TestRequestIDRoundTrip(t *testing.T) {
	for _, id := range []RequestID{IntID(7), StringID("abc"), {}} {
		b, err := json.Marshal(id)
		if err != nil {
			t.Fatal(err)
		}
		var got RequestID
		if err := got.UnmarshalJSON(b); err != nil {
			t.Fatal(err)
		}
		if got.String() != id.String() {
			t.Fatalf("round trip: %v != %v", got, id)
		}
	}
}
