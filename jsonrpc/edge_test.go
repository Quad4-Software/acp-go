// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestParseErrorResponse(t *testing.T) {
	ta, tb := pipePair()
	a := NewConn(ta)
	runConn(t, a)

	// Write malformed JSON straight onto b's transport and expect a
	// JSON-RPC parse error response addressed to a null id.
	if err := tb.Write(context.Background(), json.RawMessage("{not json")); err != nil {
		t.Fatal(err)
	}
	resp, err := tb.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		ID    json.RawMessage `json:"id"`
		Error *Error          `json:"error"`
	}
	if err := json.Unmarshal(resp, &m); err != nil {
		t.Fatalf("bad response: %s", resp)
	}
	if m.Error == nil || m.Error.Code != ErrCodeParseError {
		t.Fatalf("expected parse error, got %s", resp)
	}
	if string(m.ID) != "null" {
		t.Fatalf("expected null id, got %s", m.ID)
	}
}

func TestExtMethodFallback(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.SetFallback(func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]bool{"handled": true}, nil
	})
	runConn(t, a)
	runConn(t, b)

	var out struct {
		Handled bool `json:"handled"`
	}
	if err := a.Call(context.Background(), "_custom/method", nil, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Handled {
		t.Fatal("fallback did not run")
	}
}

func TestExtNotificationFallback(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	got := make(chan struct{}, 1)
	b.SetNotificationFallback(func(context.Context, json.RawMessage) { got <- struct{}{} })
	runConn(t, a)
	runConn(t, b)

	if err := a.Notify(context.Background(), "_custom/note", map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("fallback notification not received")
	}
}

func TestShutdownFailsPendingCall(t *testing.T) {
	ta, tb := pipePair()
	a, b := NewConn(ta), NewConn(tb)
	b.Handle("hang", func(ctx context.Context, _ json.RawMessage) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	runConn(t, a)
	runConn(t, b)

	done := make(chan error, 1)
	go func() { done <- a.Call(context.Background(), "hang", nil, nil) }()
	time.Sleep(50 * time.Millisecond)
	_ = a.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from pending call")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pending call not released")
	}
}

func TestConnDone(t *testing.T) {
	ta, _ := pipePair()
	a := NewConn(ta)
	select {
	case <-a.Done():
		t.Fatal("closed too early")
	default:
	}
	_ = a.Close()
	select {
	case <-a.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed")
	}
}

func TestCancelRequestOnCanceledContext(t *testing.T) {
	ta, _ := pipePair()
	a := NewConn(ta)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.CancelRequest(ctx, IntID(1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRequestID(t *testing.T) {
	if !(RequestID{}).IsNull() {
		t.Fatal("zero RequestID should be null")
	}
	if IntID(1).IsNull() || StringID("x").IsNull() {
		t.Fatal("non-null IDs report null")
	}
	if IntID(7).String() == StringID("7").String() {
		t.Fatal("numeric and string ids collide")
	}

	for _, in := range []RequestID{IntID(-3), StringID("abc"), {}} {
		b, err := in.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var out RequestID
		if err := out.UnmarshalJSON(b); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if out != in {
			t.Fatalf("round trip %v -> %s -> %v", in, b, out)
		}
	}

	var id RequestID
	if err := id.UnmarshalJSON([]byte("1.5")); err == nil {
		t.Fatal("fractional id should fail")
	}
	if err := id.UnmarshalJSON([]byte("{bad")); err == nil {
		t.Fatal("malformed id should fail")
	}
	if err := id.UnmarshalJSON([]byte("\"s1\"")); err != nil || id != StringID("s1") {
		t.Fatalf("string id: %v %v", id, err)
	}
}
