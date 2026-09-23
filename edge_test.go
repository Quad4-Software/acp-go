// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

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

func TestErrorHelpers(t *testing.T) {
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
	if AuthRequired().Code != ErrCodeAuthRequired {
		t.Fatal("wrong code")
	}
	if InvalidParams("p").Code != ErrCodeInvalidParams {
		t.Fatal("wrong code")
	}
	if MethodNotFound("m").Code != ErrCodeMethodNotFound {
		t.Fatal("wrong code")
	}
	if !IsErrorCode(e, ErrCodeInvalidParams) || IsErrorCode(e, ErrCodeParseError) {
		t.Fatal("IsErrorCode mismatch")
	}
	if NewErrorf(ErrCodeInternalError, "x %d", 2).Message != "x 2" {
		t.Fatal("NewErrorf broken")
	}
}
