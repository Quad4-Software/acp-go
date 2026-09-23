// SPDX-License-Identifier: 0BSD

package acp

// ProtocolVersion is the ACP protocol version. The only released
// version is 1.
type ProtocolVersion int

// ProtocolVersion1 is the protocol version this library implements.
const ProtocolVersion1 ProtocolVersion = 1

// Meta carries extension metadata under the _meta key. ACP reserves
// this key for client and agent extensions; implementations must not
// assume anything about the values it holds.
type Meta map[string]any

// SessionID identifies a session.
type SessionID string

// ToolCallID identifies a tool call within a session.
type ToolCallID string

// TerminalID identifies a terminal created via terminal/create.
type TerminalID string

// SessionModeID identifies an agent operating mode.
type SessionModeID string

// SessionConfigID identifies a session configuration option.
type SessionConfigID string

// SessionConfigValueID identifies one value of a session configuration
// option.
type SessionConfigValueID string

// SessionConfigGroupID identifies a group of select options.
type SessionConfigGroupID string

// AuthMethodID identifies an authentication method advertised during
// initialization.
type AuthMethodID string

// PermissionOptionID identifies a permission option.
type PermissionOptionID string

// ElicitationID identifies an elicitation.
type ElicitationID string

// MessageID correlates streamed message chunks.
type MessageID string

// Role is the sender or recipient of messages and data in a
// conversation.
type Role string

// Conversation roles.
const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

// Implementation describes a client or agent name and version.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title,omitempty"`
	Meta    Meta   `json:"_meta,omitempty"`
}

// Annotations inform the client how objects are used or displayed.
type Annotations struct {
	Audience     []Role   `json:"audience,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
	Priority     *float64 `json:"priority,omitempty"`
	Meta         Meta     `json:"_meta,omitempty"`
}

// EnvVariable is a name-value pair passed to a subprocess.
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Meta  Meta   `json:"_meta,omitempty"`
}

// HTTPHeader is a name-value pair sent with HTTP MCP server requests.
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Meta  Meta   `json:"_meta,omitempty"`
}
