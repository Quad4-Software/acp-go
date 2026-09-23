// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// testAgent is a minimal agent used by the end-to-end tests.
type testAgent struct {
	UnimplementedAgent
	side *AgentSide

	promptStarted chan struct{}
	promptBlock   chan struct{}
}

func (a *testAgent) Initialize(_ context.Context, _ *InitializeRequest) (*InitializeResponse, error) {
	return &InitializeResponse{
		ProtocolVersion: ProtocolVersion1,
		AgentCapabilities: AgentCapabilities{
			PromptCapabilities: PromptCapabilities{Image: true},
		},
		AgentInfo: &Implementation{Name: "test-agent", Version: "0.1.0"},
	}, nil
}

func (a *testAgent) NewSession(_ context.Context, _ *NewSessionRequest) (*NewSessionResponse, error) {
	return &NewSessionResponse{
		SessionID: "sess-1",
		Modes: &SessionModeState{
			CurrentModeID: "code",
			AvailableModes: []SessionMode{
				{ID: "code", Name: "Code"},
				{ID: "ask", Name: "Ask"},
			},
		},
	}, nil
}

func (a *testAgent) Prompt(ctx context.Context, req *PromptRequest) (*PromptResponse, error) {
	close(a.promptStarted)
	if a.promptBlock != nil {
		select {
		case <-ctx.Done():
			return &PromptResponse{StopReason: StopCancelled}, nil
		case <-a.promptBlock:
		}
	}
	if err := a.side.SessionUpdate(ctx, req.SessionID,
		NewAgentMessageChunkUpdate(NewTextBlock("working"))); err != nil {
		return nil, err
	}
	content, err := a.side.ReadTextFile(ctx, &ReadTextFileRequest{
		SessionID: req.SessionID,
		Path:      "/tmp/main.go",
	})
	if err != nil {
		return nil, err
	}
	if err := a.side.SessionUpdate(ctx, req.SessionID,
		NewAgentMessageChunkUpdate(NewTextBlock(content.Content))); err != nil {
		return nil, err
	}
	return &PromptResponse{StopReason: StopEndTurn}, nil
}

func (a *testAgent) Cancel(_ context.Context, _ *CancelNotification) {
	if a.promptBlock != nil {
		close(a.promptBlock)
	}
}

// testClient is a minimal client used by the end-to-end tests.
type testClient struct {
	UnimplementedClient

	mu      sync.Mutex
	updates []SessionNotification
}

func (c *testClient) ReadTextFile(_ context.Context, req *ReadTextFileRequest) (*ReadTextFileResponse, error) {
	if req.Path != "/tmp/main.go" {
		return nil, ResourceNotFound(req.Path)
	}
	return &ReadTextFileResponse{Content: "package main"}, nil
}

func (c *testClient) SessionUpdate(_ context.Context, n *SessionNotification) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates = append(c.updates, *n)
}

func newTestPair(t *testing.T) (*ClientSide, *testAgent, *testClient) {
	t.Helper()
	ta, tb := pipePair()
	agent := &testAgent{promptStarted: make(chan struct{})}
	client := &testClient{}
	as := NewAgentSide(ta, agent)
	agent.side = as
	cs := NewClientSide(tb, client)
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() {
		_ = as.Close()
		_ = cs.Close()
	})
	return cs, agent, client
}

func TestEndToEnd(t *testing.T) {
	cs, _, client := newTestPair(t)
	ctx := context.Background()

	init, err := cs.Initialize(ctx, &InitializeRequest{
		ProtocolVersion: ProtocolVersion1,
		ClientCapabilities: ClientCapabilities{
			FS: FileSystemCapabilities{ReadTextFile: true},
		},
		ClientInfo: &Implementation{Name: "test-client", Version: "0.1.0"},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.ProtocolVersion != ProtocolVersion1 || !init.AgentCapabilities.PromptCapabilities.Image {
		t.Fatalf("bad initialize response: %+v", init)
	}

	sess, err := cs.NewSession(ctx, &NewSessionRequest{
		CWD:        "/tmp",
		MCPServers: []MCPServer{},
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if sess.SessionID != "sess-1" || sess.Modes == nil || len(sess.Modes.AvailableModes) != 2 {
		t.Fatalf("bad session response: %+v", sess)
	}

	prompt, err := cs.Prompt(ctx, &PromptRequest{
		SessionID: sess.SessionID,
		Prompt:    []ContentBlock{NewTextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if prompt.StopReason != StopEndTurn {
		t.Fatalf("bad stop reason: %v", prompt.StopReason)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		client.mu.Lock()
		n := len(client.updates)
		client.mu.Unlock()
		if n >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("got %d updates, want 2", n)
		}
		time.Sleep(5 * time.Millisecond)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.updates[0].Update.AgentMessageChunk == nil ||
		client.updates[0].Update.AgentMessageChunk.Content.Text == nil ||
		client.updates[0].Update.AgentMessageChunk.Content.Text.Text != "working" {
		t.Fatalf("bad first update: %+v", client.updates[0])
	}
	if client.updates[1].Update.AgentMessageChunk.Content.Text.Text != "package main" {
		t.Fatalf("agent did not receive file content: %+v", client.updates[1])
	}
}

func TestSessionCancel(t *testing.T) {
	cs, agent, _ := newTestPair(t)
	agent.promptBlock = make(chan struct{})
	ctx := context.Background()

	if _, err := cs.NewSession(ctx, &NewSessionRequest{CWD: "/tmp"}); err != nil {
		t.Fatalf("new session: %v", err)
	}

	done := make(chan *PromptResponse, 1)
	go func() {
		resp, err := cs.Prompt(ctx, &PromptRequest{SessionID: "sess-1"})
		if err != nil {
			t.Errorf("prompt: %v", err)
		}
		done <- resp
	}()

	<-agent.promptStarted
	if err := cs.Cancel(ctx, "sess-1"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	select {
	case resp := <-done:
		if resp == nil || resp.StopReason != StopCancelled {
			t.Fatalf("bad response after cancel: %+v", resp)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("prompt did not return after cancel")
	}
}

func TestAgentErrorSurfaces(t *testing.T) {
	ta, tb := pipePair()
	as := NewAgentSide(ta, UnimplementedAgent{})
	cs := NewClientSide(tb, UnimplementedClient{})
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })

	err := cs.Cancel(context.Background(), "s")
	if err != nil {
		t.Fatalf("cancel notification: %v", err)
	}
	_, err = cs.NewSession(context.Background(), &NewSessionRequest{CWD: "/"})
	var e *Error
	if !errors.As(err, &e) || e.Code != ErrCodeInternalError {
		t.Fatalf("got %v, want internal error", err)
	}
}
