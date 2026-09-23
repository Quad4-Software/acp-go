// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"sync"
)

// Agent is the interface a coding agent implements to serve ACP.
// Methods map one-to-one to the protocol's agent methods. Implement
// Agent by embedding UnimplementedAgent and overriding the methods the
// agent supports.
//
// The context passed to Prompt is canceled when the client sends a
// session/cancel notification for the prompt's session, or a
// $/cancel_request for the prompt request. A canceled Prompt should
// return a PromptResponse with StopReason StopCancelled.
type Agent interface {
	// Initialize negotiates the protocol version and capabilities.
	Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error)
	// Authenticate runs the selected authentication method.
	Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error)
	// Logout ends the authenticated state.
	Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error)
	// NewSession creates a session.
	NewSession(ctx context.Context, req *NewSessionRequest) (*NewSessionResponse, error)
	// LoadSession resumes an existing session with replay.
	LoadSession(ctx context.Context, req *LoadSessionRequest) (*LoadSessionResponse, error)
	// ResumeSession resumes an existing session without replay.
	ResumeSession(ctx context.Context, req *ResumeSessionRequest) (*ResumeSessionResponse, error)
	// CloseSession frees a session's resources.
	CloseSession(ctx context.Context, req *CloseSessionRequest) (*CloseSessionResponse, error)
	// ListSessions returns known sessions.
	ListSessions(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error)
	// DeleteSession removes a session from history.
	DeleteSession(ctx context.Context, req *DeleteSessionRequest) (*DeleteSessionResponse, error)
	// SetSessionMode switches the session mode.
	SetSessionMode(ctx context.Context, req *SetSessionModeRequest) (*SetSessionModeResponse, error)
	// SetSessionConfigOption changes a session config option.
	SetSessionConfigOption(ctx context.Context, req *SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error)
	// Prompt runs one prompt turn.
	Prompt(ctx context.Context, req *PromptRequest) (*PromptResponse, error)
	// Cancel is called when the client sends session/cancel. The
	// matching Prompt context is already canceled by the time this
	// runs; use it to release other resources.
	Cancel(ctx context.Context, n *CancelNotification)
}

// UnimplementedAgent returns "not implemented" errors for every
// method. Embed it to satisfy Agent while overriding a subset.
type UnimplementedAgent struct{}

func (UnimplementedAgent) Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error) {
	return nil, notImplemented("initialize")
}

func (UnimplementedAgent) Authenticate(context.Context, *AuthenticateRequest) (*AuthenticateResponse, error) {
	return nil, notImplemented("authenticate")
}

func (UnimplementedAgent) Logout(context.Context, *LogoutRequest) (*LogoutResponse, error) {
	return nil, notImplemented("logout")
}

func (UnimplementedAgent) NewSession(context.Context, *NewSessionRequest) (*NewSessionResponse, error) {
	return nil, notImplemented("session/new")
}

func (UnimplementedAgent) LoadSession(context.Context, *LoadSessionRequest) (*LoadSessionResponse, error) {
	return nil, notImplemented("session/load")
}

func (UnimplementedAgent) ResumeSession(context.Context, *ResumeSessionRequest) (*ResumeSessionResponse, error) {
	return nil, notImplemented("session/resume")
}

func (UnimplementedAgent) CloseSession(context.Context, *CloseSessionRequest) (*CloseSessionResponse, error) {
	return nil, notImplemented("session/close")
}

func (UnimplementedAgent) ListSessions(context.Context, *ListSessionsRequest) (*ListSessionsResponse, error) {
	return nil, notImplemented("session/list")
}

func (UnimplementedAgent) DeleteSession(context.Context, *DeleteSessionRequest) (*DeleteSessionResponse, error) {
	return nil, notImplemented("session/delete")
}

func (UnimplementedAgent) SetSessionMode(context.Context, *SetSessionModeRequest) (*SetSessionModeResponse, error) {
	return nil, notImplemented("session/set_mode")
}

func (UnimplementedAgent) SetSessionConfigOption(context.Context, *SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	return nil, notImplemented("session/set_config_option")
}

func (UnimplementedAgent) Prompt(context.Context, *PromptRequest) (*PromptResponse, error) {
	return nil, notImplemented("session/prompt")
}
func (UnimplementedAgent) Cancel(context.Context, *CancelNotification) {}

func notImplemented(m string) *Error {
	return NewErrorf(ErrCodeInternalError, "%s not implemented", m)
}

// AgentSide is an agent's end of an ACP connection. It dispatches
// inbound client requests to the Agent and exposes typed methods for
// calling the client.
type AgentSide struct {
	conn  *Conn
	agent Agent

	mu      sync.Mutex
	prompts map[SessionID]context.CancelFunc
}

// NewAgentSide returns an agent-side connection dispatching to agent.
// A nil agent behaves like UnimplementedAgent.
func NewAgentSide(t Transport, agent Agent) *AgentSide {
	if agent == nil {
		agent = UnimplementedAgent{}
	}
	s := &AgentSide{conn: NewConn(t), agent: agent, prompts: map[SessionID]context.CancelFunc{}}
	c := s.conn
	c.Handle(MethodInitialize, bind(agent.Initialize))
	c.Handle(MethodAuthenticate, bind(agent.Authenticate))
	c.Handle(MethodLogout, bind(agent.Logout))
	c.Handle(MethodSessionNew, bind(agent.NewSession))
	c.Handle(MethodSessionLoad, bind(agent.LoadSession))
	c.Handle(MethodSessionResume, bind(agent.ResumeSession))
	c.Handle(MethodSessionClose, bind(agent.CloseSession))
	c.Handle(MethodSessionList, bind(agent.ListSessions))
	c.Handle(MethodSessionDelete, bind(agent.DeleteSession))
	c.Handle(MethodSessionSetMode, bind(agent.SetSessionMode))
	c.Handle(MethodSessionSetConfig, bind(agent.SetSessionConfigOption))
	c.Handle(MethodSessionPrompt, s.handlePrompt)
	c.HandleNotification(MethodSessionCancel, s.handleCancel)
	return s
}

func (s *AgentSide) handlePrompt(ctx context.Context, params json.RawMessage) (any, error) {
	req := new(PromptRequest)
	if err := json.Unmarshal(params, req); err != nil {
		return nil, InvalidParams(err.Error())
	}
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.prompts[req.SessionID] = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.prompts, req.SessionID)
		s.mu.Unlock()
		cancel()
	}()
	return s.agent.Prompt(ctx, req)
}

func (s *AgentSide) handleCancel(ctx context.Context, params json.RawMessage) {
	var n CancelNotification
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	s.mu.Lock()
	if cancel, ok := s.prompts[n.SessionID]; ok {
		cancel()
	}
	s.mu.Unlock()
	s.agent.Cancel(ctx, &n)
}

// Conn returns the underlying connection for registering extension
// methods or sending custom messages.
func (s *AgentSide) Conn() *Conn { return s.conn }

// Run serves the connection until it closes.
func (s *AgentSide) Run(ctx context.Context) error { return s.conn.Run(ctx) }

// Close shuts down the connection.
func (s *AgentSide) Close() error { return s.conn.Close() }

// Done returns a channel closed when the connection closes.
func (s *AgentSide) Done() <-chan struct{} { return s.conn.Done() }

// SessionUpdate sends a session/update notification to the client.
func (s *AgentSide) SessionUpdate(ctx context.Context, sid SessionID, u SessionUpdate) error {
	return s.conn.Notify(ctx, MethodSessionUpdate, SessionNotification{SessionID: sid, Update: u})
}

// NotifyElicitationComplete sends an elicitation/complete notification
// for a URL elicitation that finished out of band.
func (s *AgentSide) NotifyElicitationComplete(ctx context.Context, n CompleteElicitationNotification) error {
	return s.conn.Notify(ctx, MethodElicitationComplete, n)
}

// RequestPermission asks the client to authorize a tool call.
func (s *AgentSide) RequestPermission(ctx context.Context, req *RequestPermissionRequest) (*RequestPermissionResponse, error) {
	var resp RequestPermissionResponse
	err := s.conn.Call(ctx, MethodRequestPermission, req, &resp)
	return &resp, err
}

// ReadTextFile asks the client to read a file.
func (s *AgentSide) ReadTextFile(ctx context.Context, req *ReadTextFileRequest) (*ReadTextFileResponse, error) {
	var resp ReadTextFileResponse
	err := s.conn.Call(ctx, MethodFSReadTextFile, req, &resp)
	return &resp, err
}

// WriteTextFile asks the client to write a file.
func (s *AgentSide) WriteTextFile(ctx context.Context, req *WriteTextFileRequest) (*WriteTextFileResponse, error) {
	var resp WriteTextFileResponse
	err := s.conn.Call(ctx, MethodFSWriteTextFile, req, &resp)
	return &resp, err
}

// CreateTerminal asks the client to run a command in a terminal.
func (s *AgentSide) CreateTerminal(ctx context.Context, req *CreateTerminalRequest) (*CreateTerminalResponse, error) {
	var resp CreateTerminalResponse
	err := s.conn.Call(ctx, MethodTerminalCreate, req, &resp)
	return &resp, err
}

// TerminalOutput asks the client for a terminal's output.
func (s *AgentSide) TerminalOutput(ctx context.Context, req *TerminalOutputRequest) (*TerminalOutputResponse, error) {
	var resp TerminalOutputResponse
	err := s.conn.Call(ctx, MethodTerminalOutput, req, &resp)
	return &resp, err
}

// WaitForTerminalExit waits for a terminal command to exit.
func (s *AgentSide) WaitForTerminalExit(ctx context.Context, req *WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error) {
	var resp WaitForTerminalExitResponse
	err := s.conn.Call(ctx, MethodTerminalWaitForExit, req, &resp)
	return &resp, err
}

// KillTerminal kills a terminal command without releasing it.
func (s *AgentSide) KillTerminal(ctx context.Context, req *KillTerminalRequest) (*KillTerminalResponse, error) {
	var resp KillTerminalResponse
	err := s.conn.Call(ctx, MethodTerminalKill, req, &resp)
	return &resp, err
}

// ReleaseTerminal releases a terminal.
func (s *AgentSide) ReleaseTerminal(ctx context.Context, req *ReleaseTerminalRequest) (*ReleaseTerminalResponse, error) {
	var resp ReleaseTerminalResponse
	err := s.conn.Call(ctx, MethodTerminalRelease, req, &resp)
	return &resp, err
}

// CreateElicitation asks the client to collect structured input.
func (s *AgentSide) CreateElicitation(ctx context.Context, req *CreateElicitationRequest) (*CreateElicitationResponse, error) {
	var resp CreateElicitationResponse
	err := s.conn.Call(ctx, MethodElicitationCreate, req, &resp)
	return &resp, err
}

// bind wraps a typed request handler as a raw JSON-RPC Handler.
func bind[Req, Resp any](h func(context.Context, *Req) (*Resp, error)) Handler {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		req := new(Req)
		if len(params) > 0 {
			if err := json.Unmarshal(params, req); err != nil {
				return nil, InvalidParams(err.Error())
			}
		}
		return h(ctx, req)
	}
}

// bindNotify wraps a typed notification handler.
func bindNotify[Req any](h func(context.Context, *Req)) NotificationHandler {
	return func(ctx context.Context, params json.RawMessage) {
		req := new(Req)
		if len(params) > 0 {
			if err := json.Unmarshal(params, req); err != nil {
				return
			}
		}
		h(ctx, req)
	}
}
