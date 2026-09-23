// SPDX-License-Identifier: 0BSD

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

type closeFlag struct{ closed *bool }

func (c closeFlag) Close() error { *c.closed = true; return nil }

func TestStdio(t *testing.T) {
	if Stdio() == nil {
		t.Fatal("Stdio returned nil")
	}
}

func TestLineCloseRunsCloser(t *testing.T) {
	closed := false
	tr := NewLine(strings.NewReader(""), io.Discard, closeFlag{&closed})
	if err := tr.Close(); err != nil || !closed {
		t.Fatalf("close: %v closed=%v", err, closed)
	}
}

func TestLineCRLF(t *testing.T) {
	tr := NewLine(strings.NewReader("{\"a\":1}\r\n"), io.Discard, nil)
	m, err := tr.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(m) != `{"a":1}` {
		t.Fatalf("CRLF not trimmed: %q", m)
	}
}

func TestLineOversized(t *testing.T) {
	big := strings.Repeat("x", MaxMessageSize+1) + "\n"
	tr := NewLine(strings.NewReader(big), io.Discard, nil)
	if _, err := tr.Read(context.Background()); err == nil {
		t.Fatal("oversized message accepted")
	}
}

func TestCommandHelper(t *testing.T) {
	tr, err := Command(context.Background(), "cat")
	if err != nil {
		t.Skipf("cat unavailable: %v", err)
	}
	msg := json.RawMessage(`{"ok":true}`)
	if err := tr.Write(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	got, err := tr.Read(context.Background())
	if err != nil || string(got) != string(msg) {
		t.Fatalf("echo: %v %s", err, got)
	}
	_ = tr.Close()
}

func TestCommandStartError(t *testing.T) {
	_, err := NewCommandTransport(exec.Command("/nonexistent-acp-binary-xyz"), nil)
	if err == nil {
		t.Fatal("expected start error")
	}
}

func TestCommandKillOnClose(t *testing.T) {
	tr, err := NewCommandTransport(exec.Command("sleep", "60"), nil)
	if err != nil {
		t.Skipf("sleep unavailable: %v", err)
	}
	if err := tr.Close(); err == nil {
		t.Fatal("expected kill error from sleep")
	}
}

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestCommandStderrForward(t *testing.T) {
	var buf lockedBuf
	tr, err := NewCommandTransport(
		exec.Command("sh", "-c", "echo errline >&2; cat"), &buf)
	if err != nil {
		t.Skipf("sh unavailable: %v", err)
	}
	msg := json.RawMessage(`{"x":1}`)
	if err := tr.Write(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.Read(context.Background()); err != nil {
		t.Fatal(err)
	}
	_ = tr.Close()
	if !strings.Contains(buf.String(), "errline") {
		t.Fatalf("stderr not forwarded: %q", buf.String())
	}
}

func TestCommandReadAfterClose(t *testing.T) {
	tr, err := NewCommandTransport(exec.Command("cat"), nil)
	if err != nil {
		t.Skipf("cat unavailable: %v", err)
	}
	_ = tr.Close()
	if _, err := tr.Read(context.Background()); err == nil {
		t.Fatal("expected error reading after close")
	}
}

func TestLineReadEOF(t *testing.T) {
	tr := NewLine(strings.NewReader(""), io.Discard, nil)
	if _, err := tr.Read(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("got %v, want EOF", err)
	}
}
