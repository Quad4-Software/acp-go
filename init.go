// SPDX-License-Identifier: 0BSD

package acp

import "encoding/json"

// InitializeRequest opens the connection and negotiates the protocol
// version and capabilities.
type InitializeRequest struct {
	ProtocolVersion    ProtocolVersion    `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities,omitempty"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
	Meta               Meta               `json:"_meta,omitempty"`
}

// InitializeResponse carries the negotiated version and agent
// capabilities. The client should disconnect if it does not support
// the returned version.
type InitializeResponse struct {
	ProtocolVersion   ProtocolVersion   `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities,omitempty"`
	AuthMethods       []AuthMethod      `json:"authMethods,omitempty"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
	Meta              Meta              `json:"_meta,omitempty"`
}

// ClientCapabilities describes what the client supports.
type ClientCapabilities struct {
	FS          FileSystemCapabilities     `json:"fs,omitempty"`
	Terminal    bool                       `json:"terminal,omitempty"`
	Session     *ClientSessionCapabilities `json:"session,omitempty"`
	Auth        AuthCapabilities           `json:"auth,omitempty"`
	Elicitation *ElicitationCapabilities   `json:"elicitation,omitempty"`
	Meta        Meta                       `json:"_meta,omitempty"`
}

// FileSystemCapabilities flags file access support.
type FileSystemCapabilities struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
	Meta          Meta `json:"_meta,omitempty"`
}

// ClientSessionCapabilities flags session features the client supports.
type ClientSessionCapabilities struct {
	ConfigOptions *SessionConfigOptionsCapabilities `json:"configOptions,omitempty"`
	Meta          Meta                              `json:"_meta,omitempty"`
}

// SessionConfigOptionsCapabilities flags supported config option kinds.
type SessionConfigOptionsCapabilities struct {
	Boolean *BooleanConfigOptionCapabilities `json:"boolean,omitempty"`
	Meta    Meta                             `json:"_meta,omitempty"`
}

// BooleanConfigOptionCapabilities marks boolean config option support.
type BooleanConfigOptionCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// AuthCapabilities flags client authentication features.
type AuthCapabilities struct {
	Terminal bool `json:"terminal,omitempty"`
	Meta     Meta `json:"_meta,omitempty"`
}

// ElicitationCapabilities flags which elicitation modes the client
// supports.
type ElicitationCapabilities struct {
	Form *ElicitationFormCapabilities `json:"form,omitempty"`
	URL  *ElicitationURLCapabilities  `json:"url,omitempty"`
	Meta Meta                         `json:"_meta,omitempty"`
}

// ElicitationFormCapabilities marks form elicitation support.
type ElicitationFormCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// ElicitationURLCapabilities marks URL elicitation support.
type ElicitationURLCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// AgentCapabilities describes what the agent supports.
type AgentCapabilities struct {
	LoadSession        bool                  `json:"loadSession,omitempty"`
	PromptCapabilities PromptCapabilities    `json:"promptCapabilities,omitempty"`
	MCPCapabilities    MCPCapabilities       `json:"mcpCapabilities,omitempty"`
	Session            SessionCapabilities   `json:"sessionCapabilities,omitempty"`
	Auth               AgentAuthCapabilities `json:"auth,omitempty"`
	Meta               Meta                  `json:"_meta,omitempty"`
}

// PromptCapabilities flags prompt content types the agent accepts.
type PromptCapabilities struct {
	Image           bool `json:"image,omitempty"`
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
	Meta            Meta `json:"_meta,omitempty"`
}

// MCPCapabilities flags MCP transports the agent accepts.
type MCPCapabilities struct {
	HTTP bool `json:"http,omitempty"`
	SSE  bool `json:"sse,omitempty"`
	Meta Meta `json:"_meta,omitempty"`
}

// SessionCapabilities flags optional session methods. Each non-nil
// field advertises the matching session method.
type SessionCapabilities struct {
	List                  *SessionListCapabilities                  `json:"list,omitempty"`
	Delete                *SessionDeleteCapabilities                `json:"delete,omitempty"`
	AdditionalDirectories *SessionAdditionalDirectoriesCapabilities `json:"additionalDirectories,omitempty"`
	Resume                *SessionResumeCapabilities                `json:"resume,omitempty"`
	Close                 *SessionCloseCapabilities                 `json:"close,omitempty"`
	Meta                  Meta                                      `json:"_meta,omitempty"`
}

// SessionListCapabilities advertises session/list support.
type SessionListCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SessionDeleteCapabilities advertises session/delete support.
type SessionDeleteCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SessionAdditionalDirectoriesCapabilities advertises support for
// additional working directories.
type SessionAdditionalDirectoriesCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SessionResumeCapabilities advertises session/resume support.
type SessionResumeCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SessionCloseCapabilities advertises session/close support.
type SessionCloseCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// AgentAuthCapabilities flags agent authentication features.
type AgentAuthCapabilities struct {
	Logout *LogoutCapabilities `json:"logout,omitempty"`
	Meta   Meta                `json:"_meta,omitempty"`
}

// LogoutCapabilities advertises logout support.
type LogoutCapabilities struct {
	Meta Meta `json:"_meta,omitempty"`
}

// AuthMethodType discriminates AuthMethod variants.
type AuthMethodType string

// Authentication method types.
const (
	AuthMethodTerminalType AuthMethodType = "terminal"
)

// AuthMethod describes one way to authenticate with the agent. The
// terminal variant carries a type tag on the wire; the plain agent
// method carries none.
type AuthMethod struct {
	Type     AuthMethodType
	Terminal *AuthMethodTerminal
	Agent    *AuthMethodAgent

	// Raw preserves the original JSON of an unrecognized method type.
	Raw json.RawMessage
}

// AuthMethodTerminal authenticates by running a terminal command the
// client executes on the user's behalf.
type AuthMethodTerminal struct {
	ID          AuthMethodID      `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Meta        Meta              `json:"_meta,omitempty"`
}

// AuthMethodAgent authenticates through the agent itself.
type AuthMethodAgent struct {
	ID          AuthMethodID `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Meta        Meta         `json:"_meta,omitempty"`
}

// MarshalJSON emits the populated variant. Terminal methods carry a
// type tag; agent methods do not.
func (a AuthMethod) MarshalJSON() ([]byte, error) {
	if a.Raw != nil && a.Terminal == nil && a.Agent == nil {
		return a.Raw, nil
	}
	if a.Type == AuthMethodTerminalType || (a.Type == "" && a.Terminal != nil) {
		return marshalTagged("type", string(AuthMethodTerminalType), a.Terminal)
	}
	if a.Agent != nil {
		return json.Marshal(a.Agent)
	}
	if a.Raw != nil {
		return a.Raw, nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON decodes an auth method by its type field.
func (a *AuthMethod) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	*a = AuthMethod{Type: AuthMethodType(tag)}
	switch a.Type {
	case AuthMethodTerminalType:
		a.Terminal = new(AuthMethodTerminal)
		err = json.Unmarshal(data, a.Terminal)
	case "":
		a.Agent = new(AuthMethodAgent)
		err = json.Unmarshal(data, a.Agent)
	default:
		a.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// AuthenticateRequest selects an authentication method advertised in
// the initialize response.
type AuthenticateRequest struct {
	MethodID AuthMethodID `json:"methodId"`
	Meta     Meta         `json:"_meta,omitempty"`
}

// AuthenticateResponse acknowledges authentication.
type AuthenticateResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}

// LogoutRequest ends the authenticated state. Requires the logout
// agent capability.
type LogoutRequest struct {
	Meta Meta `json:"_meta,omitempty"`
}

// LogoutResponse acknowledges logout.
type LogoutResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}
