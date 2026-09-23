// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"testing"
	"time"
)

// fullClient implements every client method for the wrapper tests.
type fullClient struct {
	UnimplementedClient
	updates chan SessionNotification
}

func (f *fullClient) RequestPermission(_ context.Context, _ *RequestPermissionRequest) (*RequestPermissionResponse, error) {
	return &RequestPermissionResponse{Outcome: NewSelectedOutcome("allow")}, nil
}

func (f *fullClient) ReadTextFile(_ context.Context, req *ReadTextFileRequest) (*ReadTextFileResponse, error) {
	return &ReadTextFileResponse{Content: "file:" + req.Path}, nil
}

func (f *fullClient) WriteTextFile(_ context.Context, _ *WriteTextFileRequest) (*WriteTextFileResponse, error) {
	return &WriteTextFileResponse{}, nil
}

func (f *fullClient) CreateTerminal(_ context.Context, _ *CreateTerminalRequest) (*CreateTerminalResponse, error) {
	return &CreateTerminalResponse{TerminalID: "t1"}, nil
}

func (f *fullClient) TerminalOutput(_ context.Context, _ *TerminalOutputRequest) (*TerminalOutputResponse, error) {
	return &TerminalOutputResponse{Output: "out", Truncated: false}, nil
}

func (f *fullClient) WaitForTerminalExit(_ context.Context, _ *WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error) {
	code := int64(0)
	return &WaitForTerminalExitResponse{ExitCode: &code}, nil
}

func (f *fullClient) KillTerminal(_ context.Context, _ *KillTerminalRequest) (*KillTerminalResponse, error) {
	return &KillTerminalResponse{}, nil
}

func (f *fullClient) ReleaseTerminal(_ context.Context, _ *ReleaseTerminalRequest) (*ReleaseTerminalResponse, error) {
	return &ReleaseTerminalResponse{}, nil
}

func (f *fullClient) CreateElicitation(_ context.Context, _ *CreateElicitationRequest) (*CreateElicitationResponse, error) {
	return &CreateElicitationResponse{Action: ElicitationDecline}, nil
}

func (f *fullClient) SessionUpdate(_ context.Context, n *SessionNotification) {
	f.updates <- *n
}

// fullAgent implements every agent method for the wrapper tests.
type fullAgent struct {
	UnimplementedAgent
	cancelled chan SessionID
}

func (fullAgent) Authenticate(_ context.Context, _ *AuthenticateRequest) (*AuthenticateResponse, error) {
	return &AuthenticateResponse{}, nil
}

func (fullAgent) Logout(_ context.Context, _ *LogoutRequest) (*LogoutResponse, error) {
	return &LogoutResponse{}, nil
}

func (fullAgent) LoadSession(_ context.Context, _ *LoadSessionRequest) (*LoadSessionResponse, error) {
	return &LoadSessionResponse{}, nil
}

func (fullAgent) ResumeSession(_ context.Context, _ *ResumeSessionRequest) (*ResumeSessionResponse, error) {
	return &ResumeSessionResponse{}, nil
}

func (fullAgent) CloseSession(_ context.Context, _ *CloseSessionRequest) (*CloseSessionResponse, error) {
	return &CloseSessionResponse{}, nil
}

func (fullAgent) ListSessions(_ context.Context, _ *ListSessionsRequest) (*ListSessionsResponse, error) {
	return &ListSessionsResponse{Sessions: []SessionInfo{{SessionID: "s1", CWD: "/w"}}}, nil
}

func (fullAgent) DeleteSession(_ context.Context, _ *DeleteSessionRequest) (*DeleteSessionResponse, error) {
	return &DeleteSessionResponse{}, nil
}

func (fullAgent) SetSessionMode(_ context.Context, _ *SetSessionModeRequest) (*SetSessionModeResponse, error) {
	return &SetSessionModeResponse{}, nil
}

func (fullAgent) SetSessionConfigOption(_ context.Context, _ *SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	return &SetSessionConfigOptionResponse{ConfigOptions: []SessionConfigOption{}}, nil
}

func (a *fullAgent) Cancel(_ context.Context, n *CancelNotification) {
	a.cancelled <- n.SessionID
}

func TestAgentSideWrappers(t *testing.T) {
	ta, tb := pipePair()
	fc := &fullClient{updates: make(chan SessionNotification, 16)}
	as := NewAgentSide(ta, UnimplementedAgent{})
	cs := NewClientSide(tb, fc)
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })
	ctx := context.Background()

	perm, err := as.RequestPermission(ctx, &RequestPermissionRequest{
		SessionID: "s", ToolCall: ToolCallUpdate{ToolCallID: "t"},
		Options: []PermissionOption{{OptionID: "allow", Name: "a", Kind: PermissionAllowOnce}},
	})
	if err != nil || perm.Outcome.Selected == nil {
		t.Fatalf("permission: %v %+v", err, perm)
	}

	rf, err := as.ReadTextFile(ctx, &ReadTextFileRequest{SessionID: "s", Path: "/f"})
	if err != nil || rf.Content != "file:/f" {
		t.Fatalf("read: %v %+v", err, rf)
	}
	if _, err := as.WriteTextFile(ctx, &WriteTextFileRequest{SessionID: "s", Path: "/f", Content: "c"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	term, err := as.CreateTerminal(ctx, &CreateTerminalRequest{SessionID: "s", Command: "ls"})
	if err != nil || term.TerminalID != "t1" {
		t.Fatalf("terminal create: %v %+v", err, term)
	}
	out, err := as.TerminalOutput(ctx, &TerminalOutputRequest{SessionID: "s", TerminalID: "t1"})
	if err != nil || out.Output != "out" {
		t.Fatalf("terminal output: %v %+v", err, out)
	}
	wt, err := as.WaitForTerminalExit(ctx, &WaitForTerminalExitRequest{SessionID: "s", TerminalID: "t1"})
	if err != nil || wt.ExitCode == nil || *wt.ExitCode != 0 {
		t.Fatalf("wait: %v %+v", err, wt)
	}
	if _, err := as.KillTerminal(ctx, &KillTerminalRequest{SessionID: "s", TerminalID: "t1"}); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if _, err := as.ReleaseTerminal(ctx, &ReleaseTerminalRequest{SessionID: "s", TerminalID: "t1"}); err != nil {
		t.Fatalf("release: %v", err)
	}

	el, err := as.CreateElicitation(ctx, &CreateElicitationRequest{
		Mode:    ElicitationModeForm,
		Session: &ElicitationSessionScope{SessionID: "s"},
	})
	if err != nil || el.Action != ElicitationDecline {
		t.Fatalf("elicitation: %v %+v", err, el)
	}

	if err := as.SessionUpdate(ctx, "s", NewAgentMessageChunkUpdate(NewTextBlock("u"))); err != nil {
		t.Fatalf("session update: %v", err)
	}
	select {
	case n := <-fc.updates:
		if n.SessionID != "s" || n.Update.AgentMessageChunk == nil {
			t.Fatalf("bad update: %+v", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("update not received")
	}

	if err := as.NotifyElicitationComplete(ctx, CompleteElicitationNotification{ElicitationID: "e1"}); err != nil {
		t.Fatalf("elicitation complete: %v", err)
	}
}

func TestClientSideWrappers(t *testing.T) {
	ta, tb := pipePair()
	fa := &fullAgent{cancelled: make(chan SessionID, 1)}
	as := NewAgentSide(ta, fa)
	cs := NewClientSide(tb, UnimplementedClient{})
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })
	ctx := context.Background()

	if _, err := cs.Authenticate(ctx, &AuthenticateRequest{MethodID: "m"}); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if _, err := cs.Logout(ctx, &LogoutRequest{}); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := cs.LoadSession(ctx, &LoadSessionRequest{SessionID: "s", CWD: "/w"}); err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := cs.ResumeSession(ctx, &ResumeSessionRequest{SessionID: "s", CWD: "/w"}); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, err := cs.CloseSession(ctx, &CloseSessionRequest{SessionID: "s"}); err != nil {
		t.Fatalf("close: %v", err)
	}
	list, err := cs.ListSessions(ctx, &ListSessionsRequest{})
	if err != nil || len(list.Sessions) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := cs.DeleteSession(ctx, &DeleteSessionRequest{SessionID: "s"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := cs.SetSessionMode(ctx, &SetSessionModeRequest{SessionID: "s", ModeID: "m"}); err != nil {
		t.Fatalf("set mode: %v", err)
	}
	cfg, err := cs.SetSessionConfigOption(ctx, &SetSessionConfigOptionRequest{SessionID: "s", ConfigID: "c", Value: "v"})
	if err != nil || cfg.ConfigOptions == nil {
		t.Fatalf("set config: %v %+v", err, cfg)
	}
	if err := cs.Cancel(ctx, "s1"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	select {
	case sid := <-fa.cancelled:
		if sid != "s1" {
			t.Fatalf("cancel for %q", sid)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel not delivered")
	}
}
