// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/Quad4-Software/acp-go/transport"
)

// captureClient records session updates in arrival order.
type captureClient struct {
	UnimplementedClient
	mu      sync.Mutex
	updates []SessionUpdate
}

func (c *captureClient) SessionUpdate(_ context.Context, n *SessionNotification) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates = append(c.updates, n.Update)
}

func (c *captureClient) waitForUpdates(t *testing.T, want int) []SessionUpdate {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		if len(c.updates) >= want {
			out := append([]SessionUpdate(nil), c.updates...)
			c.mu.Unlock()
			return out
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	t.Fatalf("timed out waiting for %d updates, got %d", want, len(c.updates))
	return nil
}

// TestE2EEchoAgent drives the real echo-agent example binary through a
// full ACP session over a subprocess transport.
func TestE2EEchoAgent(t *testing.T) {
	bin := buildEchoAgent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tr, err := transport.NewCommandTransport(exec.CommandContext(ctx, bin), io.Discard) // #nosec G204 -- test-built binary
	if err != nil {
		t.Fatalf("spawn agent: %v", err)
	}
	client := &captureClient{}
	side := NewClientSide(tr, client)
	go func() { _ = side.Run(ctx) }()
	defer func() { _ = side.Close() }()

	init, err := side.Initialize(ctx, &InitializeRequest{
		ProtocolVersion: ProtocolVersion1,
		ClientInfo:      &Implementation{Name: "e2e", Version: "0"},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.ProtocolVersion != ProtocolVersion1 {
		t.Fatalf("protocol version %d, want %d", init.ProtocolVersion, ProtocolVersion1)
	}
	if init.AgentInfo == nil || init.AgentInfo.Name != "echo-agent" {
		t.Fatalf("agent info: %+v", init.AgentInfo)
	}

	sess, err := side.NewSession(ctx, &NewSessionRequest{CWD: t.TempDir()})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if sess.SessionID == "" {
		t.Fatal("empty session id")
	}
	if sess.Modes == nil || sess.Modes.CurrentModeID != "echo" {
		t.Fatalf("modes: %+v", sess.Modes)
	}

	resp, err := side.Prompt(ctx, &PromptRequest{
		SessionID: sess.SessionID,
		Prompt:    []ContentBlock{NewTextBlock("hello e2e")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if resp.StopReason != StopEndTurn {
		t.Fatalf("stop reason %q, want %q", resp.StopReason, StopEndTurn)
	}

	updates := client.waitForUpdates(t, 3)
	var sawToolCall, sawChunk, sawComplete bool
	var echoed string
	for _, u := range updates {
		switch {
		case u.ToolCall != nil:
			sawToolCall = true
			if u.ToolCall.ToolCallID != "echo-1" || u.ToolCall.Status != ToolCallInProgress {
				t.Fatalf("bad tool call: %+v", u.ToolCall)
			}
		case u.AgentMessageChunk != nil:
			sawChunk = true
			if u.AgentMessageChunk.Content.Text != nil {
				echoed = u.AgentMessageChunk.Content.Text.Text
			}
		case u.ToolCallUpdate != nil:
			if u.ToolCallUpdate.Status == ToolCallCompleted {
				sawComplete = true
			}
		}
	}
	if !sawToolCall || !sawChunk || !sawComplete {
		t.Fatalf("missing updates: tool=%v chunk=%v complete=%v", sawToolCall, sawChunk, sawComplete)
	}
	if echoed != "echo: hello e2e" {
		t.Fatalf("echoed %q", echoed)
	}

	// Second turn on the same session.
	resp, err = side.Prompt(ctx, &PromptRequest{
		SessionID: sess.SessionID,
		Prompt:    []ContentBlock{NewTextBlock("again")},
	})
	if err != nil || resp.StopReason != StopEndTurn {
		t.Fatalf("second prompt: %v %+v", err, resp)
	}

	// Clean shutdown: stdin close lets the agent exit on its own.
	if err := side.Close(); err != nil {
		t.Logf("close: %v", err)
	}
}

// TestE2EUnknownMethod verifies the real agent returns method not
// found for extension methods it does not implement.
func TestE2EUnknownMethod(t *testing.T) {
	bin := buildEchoAgent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tr, err := transport.NewCommandTransport(exec.CommandContext(ctx, bin), io.Discard) // #nosec G204 -- test-built binary
	if err != nil {
		t.Fatalf("spawn agent: %v", err)
	}
	side := NewClientSide(tr, UnimplementedClient{})
	go func() { _ = side.Run(ctx) }()
	defer func() { _ = side.Close() }()

	var out any
	err = side.Conn().Call(ctx, "_vendor/nope", nil, &out)
	if !IsErrorCode(err, ErrCodeMethodNotFound) {
		t.Fatalf("expected method not found, got %v", err)
	}
}
