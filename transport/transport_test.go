// SPDX-License-Identifier: 0BSD

package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestLineRoundTrip(t *testing.T) {
	var sb strings.Builder
	tr := NewLine(strings.NewReader("{\"a\":1}\n{\"b\":2}\n"), &sb, nil)
	m1, err := tr.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m2, err := tr.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(m1) != `{"a":1}` || string(m2) != `{"b":2}` {
		t.Fatalf("got %s %s", m1, m2)
	}
	if _, err := tr.Read(context.Background()); !errors.Is(err, context.Canceled) && err == nil {
		// io.EOF from exhausted reader
		t.Fatalf("expected EOF, got %v", err)
	}
	if err := tr.Write(context.Background(), json.RawMessage(`{"c":3}`)); err != nil {
		t.Fatal(err)
	}
	if sb.String() != "{\"c\":3}\n" {
		t.Fatalf("wrote %q", sb.String())
	}
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLineReadCanceledContext(t *testing.T) {
	pr, pw := io.Pipe()
	defer func() { _ = pw.Close() }()
	tr := NewLine(pr, pw, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tr.Read(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestLineWriteCanceledContext(t *testing.T) {
	tr := NewLine(strings.NewReader(""), &strings.Builder{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tr.Write(ctx, json.RawMessage(`{}`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
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
	if err := tr.Close(); err != nil {
		t.Logf("close: %v", err)
	}
}

func TestCommandTransportWait(t *testing.T) {
	tr, err := Command(context.Background(), "true")
	if err != nil {
		t.Skipf("true unavailable: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- tr.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not return")
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
