// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
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

func TestInvalidParamsError(t *testing.T) {
	ta, tb := pipePair()
	agent := &testAgent{promptStarted: make(chan struct{})}
	as := NewAgentSide(ta, agent)
	cs := NewClientSide(tb, UnimplementedClient{})
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })

	// Send a malformed prompt payload directly.
	err := cs.Conn().Call(context.Background(), MethodSessionPrompt,
		json.RawMessage(`{"sessionId": 123}`), nil)
	if !IsErrorCode(err, ErrCodeInvalidParams) {
		t.Fatalf("got %v, want invalid params", err)
	}
}

func TestCommandTransport(t *testing.T) {
	tr, err := NewCommandTransport(exec.Command("cat"), nil)
	if err != nil {
		t.Skipf("cat unavailable: %v", err)
	}
	msg := json.RawMessage(`{"jsonrpc":"2.0","method":"ping","id":1}`)
	if err := tr.Write(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	got, err := tr.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(msg) {
		t.Fatalf("got %s, want %s", got, msg)
	}
	if tr.Process() == nil {
		t.Fatal("no process")
	}
	if err := tr.Close(); err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		// cat exits 0 after stdin close; a nil error is expected on
		// most systems but accept signal errors on odd platforms.
		t.Logf("close: %v", err)
	}
}

func TestCommandTransportRejectsWiredCmd(t *testing.T) {
	cmd := exec.Command("cat")
	cmd.Stdout = &noopWriter{}
	if _, err := NewCommandTransport(cmd, nil); err == nil {
		t.Fatal("expected error for pre-wired cmd")
	}
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestNilAgentAndClient(t *testing.T) {
	ta, tb := pipePair()
	as := NewAgentSide(ta, nil)
	cs := NewClientSide(tb, nil)
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })

	_, err := cs.Initialize(context.Background(), &InitializeRequest{ProtocolVersion: 1})
	var e *Error
	if !errors.As(err, &e) || e.Code != ErrCodeInternalError {
		t.Fatalf("got %v, want not-implemented internal error", err)
	}
}

func TestHelpers(t *testing.T) {
	if !(RequestID{}).IsNull() {
		t.Fatal("zero RequestID should be null")
	}
	if IntID(1).IsNull() || StringID("x").IsNull() {
		t.Fatal("non-null IDs report null")
	}
	if IntID(7).String() == StringID("7").String() {
		t.Fatal("numeric and string ids collide")
	}

	e := NewError(ErrCodeInvalidParams, "bad").WithData(map[string]int{"x": 1})
	if e.Data == nil {
		t.Fatal("WithData lost")
	}
	if got := ErrorCodeOf(e); got != ErrCodeInvalidParams {
		t.Fatalf("ErrorCodeOf: %v", got)
	}
	if ErrorCodeOf(errors.New("x")) != ErrCodeInternalError {
		t.Fatal("non-RPC error should map to internal")
	}
	if ErrorCodeOf(nil) != 0 {
		t.Fatal("nil should map to 0")
	}
	if ResourceNotFound("f").Code != ErrCodeResourceNotFound {
		t.Fatal("wrong code")
	}
	if InvalidParams("p").Code != ErrCodeInvalidParams {
		t.Fatal("wrong code")
	}
	var reqID RequestID
	if err := reqID.UnmarshalJSON([]byte("1.5")); err == nil {
		t.Fatal("fractional id should fail")
	}
}
