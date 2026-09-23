// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/Quad4-Software/acp-go/transport"
)

// chaosTransport wraps a transport and randomly drops, corrupts, or
// fails writes. It models a hostile or flaky channel between peers.
type chaosTransport struct {
	inner interface {
		Read(context.Context) (json.RawMessage, error)
		Write(context.Context, json.RawMessage) error
		Close() error
	}
	mu                  sync.Mutex
	rng                 *rand.Rand
	drop, fail, corrupt float64
}

func (c *chaosTransport) Read(ctx context.Context) (json.RawMessage, error) {
	return c.inner.Read(ctx)
}

func (c *chaosTransport) Write(ctx context.Context, msg json.RawMessage) error {
	c.mu.Lock()
	r := c.rng.Float64()
	var corruptIdx int
	if len(msg) > 4 {
		corruptIdx = 1 + c.rng.IntN(len(msg)-2)
	}
	c.mu.Unlock()
	switch {
	case r < c.drop:
		return nil
	case r < c.drop+c.fail:
		return errors.New("chaos: write failed")
	case r < c.drop+c.fail+c.corrupt:
		b := append([]byte(nil), msg...)
		if len(b) > 4 {
			b[corruptIdx] ^= 0xFF
		}
		return c.inner.Write(ctx, b)
	default:
		return c.inner.Write(ctx, msg)
	}
}

func (c *chaosTransport) Close() error { return c.inner.Close() }

// TestChaosConnection hammers a connection over a lossy channel. Calls
// may fail, but the conn must never panic or deadlock, and shutdown
// must complete.
func TestChaosConnection(t *testing.T) {
	ta, tb := pipePair()
	// Deterministic seeds keep the chaos reproducible.
	chaosA := &chaosTransport{inner: ta, rng: rand.New(rand.NewPCG(7, 1)), drop: 0.1, fail: 0.05, corrupt: 0.15} // #nosec G404 -- fault injection, not crypto
	chaosB := &chaosTransport{inner: tb, rng: rand.New(rand.NewPCG(9, 3)), drop: 0.1, fail: 0.05, corrupt: 0.15} // #nosec G404 -- fault injection, not crypto

	srv := NewConn(chaosA)
	srv.Handle("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]bool{"pong": true}, nil
	})
	cli := NewConn(chaosB)

	ctx := context.Background()
	go func() { _ = srv.Run(ctx) }()
	go func() { _ = cli.Run(ctx) }()
	defer func() { _ = srv.Close() }()
	defer func() { _ = cli.Close() }()

	var ok, failed int
	for i := 0; i < 100; i++ {
		cctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		var out map[string]bool
		err := cli.Call(cctx, "ping", nil, &out)
		cancel()
		if err == nil && out["pong"] {
			ok++
		} else {
			failed++
		}
	}
	if ok == 0 {
		t.Fatal("no call ever succeeded over chaos channel")
	}
	if failed == 0 {
		t.Log("chaos rates produced zero failures; check rng seeds")
	}
	t.Logf("chaos results: %d ok, %d failed", ok, failed)
}

// failReadTransport fails every read immediately.
type failReadTransport struct{}

func (failReadTransport) Read(context.Context) (json.RawMessage, error) {
	return nil, errors.New("chaos: read failed")
}

func (failReadTransport) Write(context.Context, json.RawMessage) error { return nil }
func (failReadTransport) Close() error                                 { return nil }

// TestChaosReadFailure verifies a transport whose reads fail tears the
// conn down and fails pending calls instead of hanging.
func TestChaosReadFailure(t *testing.T) {
	c := NewConn(failReadTransport{})
	if err := c.Run(context.Background()); err == nil {
		t.Fatal("expected Run to return the read error")
	}

	done := make(chan error, 1)
	go func() { done <- c.Call(context.Background(), "ping", nil, nil) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("call on dead conn returned nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("call hung on dead conn")
	}
}

// TestChaosAbruptEOF closes the pipe mid-call. The pending call must
// fail, and the conn must shut down without leaking.
func TestChaosAbruptEOF(t *testing.T) {
	aR, aW := io.Pipe()
	ta := transport.NewLine(aR, io.Discard, nil)
	c := NewConn(ta)
	go func() { _ = c.Run(context.Background()) }()
	t.Cleanup(func() { _ = c.Close() })

	done := make(chan error, 1)
	go func() { done <- c.Call(context.Background(), "ping", nil, nil) }()

	// Give the call a moment to register, then kill the channel.
	time.Sleep(50 * time.Millisecond)
	_ = aR.Close()
	_ = aW.Close()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("pending call returned nil after EOF")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending call hung after EOF")
	}
}

// TestChaosPartialLine writes a truncated message then EOF. The peer
// sees EOF and shuts down rather than waiting forever.
func TestChaosPartialLine(t *testing.T) {
	pr, pw := io.Pipe()
	tr := transport.NewLine(pr, pw, nil)
	c := NewConn(tr)
	done := make(chan error, 1)
	go func() { done <- c.Run(context.Background()) }()

	if _, err := pw.Write([]byte(`{"jsonrpc":"2.0","id`)); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v after partial line EOF", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run hung after partial line EOF")
	}
	_ = pr.Close()
}
