// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
)

// Client is the interface an editor or other UI implements to serve
// ACP. Methods map one-to-one to the protocol's client methods.
// Implement Client by embedding UnimplementedClient and overriding the
// methods the client supports.
type Client interface {
	// RequestPermission asks the user to authorize a tool call.
	RequestPermission(ctx context.Context, req *RequestPermissionRequest) (*RequestPermissionResponse, error)
	// ReadTextFile reads a file on behalf of the agent.
	ReadTextFile(ctx context.Context, req *ReadTextFileRequest) (*ReadTextFileResponse, error)
	// WriteTextFile writes a file on behalf of the agent.
	WriteTextFile(ctx context.Context, req *WriteTextFileRequest) (*WriteTextFileResponse, error)
	// CreateTerminal runs a command in a new terminal.
	CreateTerminal(ctx context.Context, req *CreateTerminalRequest) (*CreateTerminalResponse, error)
	// TerminalOutput returns a terminal's output and exit status.
	TerminalOutput(ctx context.Context, req *TerminalOutputRequest) (*TerminalOutputResponse, error)
	// WaitForTerminalExit blocks until a terminal command exits.
	WaitForTerminalExit(ctx context.Context, req *WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error)
	// KillTerminal kills a terminal command without releasing it.
	KillTerminal(ctx context.Context, req *KillTerminalRequest) (*KillTerminalResponse, error)
	// ReleaseTerminal releases a terminal.
	ReleaseTerminal(ctx context.Context, req *ReleaseTerminalRequest) (*ReleaseTerminalResponse, error)
	// CreateElicitation collects structured input from the user.
	CreateElicitation(ctx context.Context, req *CreateElicitationRequest) (*CreateElicitationResponse, error)
	// SessionUpdate receives real-time session updates.
	SessionUpdate(ctx context.Context, n *SessionNotification)
	// ElicitationComplete is notified when a URL elicitation finishes.
	ElicitationComplete(ctx context.Context, n *CompleteElicitationNotification)
}

// UnimplementedClient returns "not implemented" errors for every
// method. Embed it to satisfy Client while overriding a subset.
type UnimplementedClient struct{}

func (UnimplementedClient) RequestPermission(context.Context, *RequestPermissionRequest) (*RequestPermissionResponse, error) {
	return nil, notImplemented("session/request_permission")
}

func (UnimplementedClient) ReadTextFile(context.Context, *ReadTextFileRequest) (*ReadTextFileResponse, error) {
	return nil, notImplemented("fs/read_text_file")
}

func (UnimplementedClient) WriteTextFile(context.Context, *WriteTextFileRequest) (*WriteTextFileResponse, error) {
	return nil, notImplemented("fs/write_text_file")
}

func (UnimplementedClient) CreateTerminal(context.Context, *CreateTerminalRequest) (*CreateTerminalResponse, error) {
	return nil, notImplemented("terminal/create")
}

func (UnimplementedClient) TerminalOutput(context.Context, *TerminalOutputRequest) (*TerminalOutputResponse, error) {
	return nil, notImplemented("terminal/output")
}

func (UnimplementedClient) WaitForTerminalExit(context.Context, *WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error) {
	return nil, notImplemented("terminal/wait_for_exit")
}

func (UnimplementedClient) KillTerminal(context.Context, *KillTerminalRequest) (*KillTerminalResponse, error) {
	return nil, notImplemented("terminal/kill")
}

func (UnimplementedClient) ReleaseTerminal(context.Context, *ReleaseTerminalRequest) (*ReleaseTerminalResponse, error) {
	return nil, notImplemented("terminal/release")
}

func (UnimplementedClient) CreateElicitation(context.Context, *CreateElicitationRequest) (*CreateElicitationResponse, error) {
	return nil, notImplemented("elicitation/create")
}
func (UnimplementedClient) SessionUpdate(context.Context, *SessionNotification)                   {}
func (UnimplementedClient) ElicitationComplete(context.Context, *CompleteElicitationNotification) {}

// ClientSide is a client's end of an ACP connection. It dispatches
// inbound agent requests to the Client and exposes typed methods for
// calling the agent.
type ClientSide struct {
	conn   *Conn
	client Client
}

// NewClientSide returns a client-side connection dispatching to
// client. A nil client behaves like UnimplementedClient.
func NewClientSide(t Transport, client Client) *ClientSide {
	if client == nil {
		client = UnimplementedClient{}
	}
	s := &ClientSide{conn: NewConn(t), client: client}
	c := s.conn
	c.Handle(MethodRequestPermission, bind(client.RequestPermission))
	c.Handle(MethodFSReadTextFile, bind(client.ReadTextFile))
	c.Handle(MethodFSWriteTextFile, bind(client.WriteTextFile))
	c.Handle(MethodTerminalCreate, bind(client.CreateTerminal))
	c.Handle(MethodTerminalOutput, bind(client.TerminalOutput))
	c.Handle(MethodTerminalWaitForExit, bind(client.WaitForTerminalExit))
	c.Handle(MethodTerminalKill, bind(client.KillTerminal))
	c.Handle(MethodTerminalRelease, bind(client.ReleaseTerminal))
	c.Handle(MethodElicitationCreate, bind(client.CreateElicitation))
	c.HandleNotification(MethodSessionUpdate, bindNotify(client.SessionUpdate))
	c.HandleNotification(MethodElicitationComplete, bindNotify(client.ElicitationComplete))
	return s
}

// Conn returns the underlying connection for registering extension
// methods or sending custom messages.
func (s *ClientSide) Conn() *Conn { return s.conn }

// Run serves the connection until it closes.
func (s *ClientSide) Run(ctx context.Context) error { return s.conn.Run(ctx) }

// Close shuts down the connection.
func (s *ClientSide) Close() error { return s.conn.Close() }

// Done returns a channel closed when the connection closes.
func (s *ClientSide) Done() <-chan struct{} { return s.conn.Done() }

// Initialize negotiates protocol version and capabilities.
func (s *ClientSide) Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
	var resp InitializeResponse
	err := s.conn.Call(ctx, MethodInitialize, req, &resp)
	return &resp, err
}

// Authenticate runs an authentication method on the agent.
func (s *ClientSide) Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error) {
	var resp AuthenticateResponse
	err := s.conn.Call(ctx, MethodAuthenticate, req, &resp)
	return &resp, err
}

// Logout ends the authenticated state.
func (s *ClientSide) Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	var resp LogoutResponse
	err := s.conn.Call(ctx, MethodLogout, req, &resp)
	return &resp, err
}

// NewSession creates a session.
func (s *ClientSide) NewSession(ctx context.Context, req *NewSessionRequest) (*NewSessionResponse, error) {
	var resp NewSessionResponse
	err := s.conn.Call(ctx, MethodSessionNew, req, &resp)
	return &resp, err
}

// LoadSession resumes an existing session with replay.
func (s *ClientSide) LoadSession(ctx context.Context, req *LoadSessionRequest) (*LoadSessionResponse, error) {
	var resp LoadSessionResponse
	err := s.conn.Call(ctx, MethodSessionLoad, req, &resp)
	return &resp, err
}

// ResumeSession resumes an existing session without replay.
func (s *ClientSide) ResumeSession(ctx context.Context, req *ResumeSessionRequest) (*ResumeSessionResponse, error) {
	var resp ResumeSessionResponse
	err := s.conn.Call(ctx, MethodSessionResume, req, &resp)
	return &resp, err
}

// CloseSession frees a session's resources.
func (s *ClientSide) CloseSession(ctx context.Context, req *CloseSessionRequest) (*CloseSessionResponse, error) {
	var resp CloseSessionResponse
	err := s.conn.Call(ctx, MethodSessionClose, req, &resp)
	return &resp, err
}

// ListSessions lists the agent's known sessions.
func (s *ClientSide) ListSessions(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error) {
	var resp ListSessionsResponse
	err := s.conn.Call(ctx, MethodSessionList, req, &resp)
	return &resp, err
}

// DeleteSession removes a session from history.
func (s *ClientSide) DeleteSession(ctx context.Context, req *DeleteSessionRequest) (*DeleteSessionResponse, error) {
	var resp DeleteSessionResponse
	err := s.conn.Call(ctx, MethodSessionDelete, req, &resp)
	return &resp, err
}

// SetSessionMode switches the session's operating mode.
func (s *ClientSide) SetSessionMode(ctx context.Context, req *SetSessionModeRequest) (*SetSessionModeResponse, error) {
	var resp SetSessionModeResponse
	err := s.conn.Call(ctx, MethodSessionSetMode, req, &resp)
	return &resp, err
}

// SetSessionConfigOption changes a session config option.
func (s *ClientSide) SetSessionConfigOption(ctx context.Context, req *SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	var resp SetSessionConfigOptionResponse
	err := s.conn.Call(ctx, MethodSessionSetConfig, req, &resp)
	return &resp, err
}

// Prompt sends user content and waits for the turn to end.
func (s *ClientSide) Prompt(ctx context.Context, req *PromptRequest) (*PromptResponse, error) {
	var resp PromptResponse
	err := s.conn.Call(ctx, MethodSessionPrompt, req, &resp)
	return &resp, err
}

// Cancel sends a session/cancel notification for an ongoing turn.
func (s *ClientSide) Cancel(ctx context.Context, sid SessionID) error {
	return s.conn.Notify(ctx, MethodSessionCancel, CancelNotification{SessionID: sid})
}
