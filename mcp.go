// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// MCPServerType discriminates MCPServer transport variants.
type MCPServerType string

// MCP server transports.
const (
	MCPTypeHTTP  MCPServerType = "http"
	MCPTypeSSE   MCPServerType = "sse"
	MCPTypeStdio MCPServerType = "stdio"
)

// MCPServer configures a connection to a Model Context Protocol server
// the agent uses during prompts. Exactly one variant field is
// populated. The stdio variant has no type tag on the wire.
type MCPServer struct {
	Type  MCPServerType
	HTTP  *MCPServerHTTP
	SSE   *MCPServerSSE
	Stdio *MCPServerStdio

	// Raw preserves the original JSON of an unrecognized transport.
	Raw json.RawMessage
}

// MCPServerHTTP connects to an MCP server over HTTP. Requires the
// http MCP capability.
type MCPServerHTTP struct {
	Name    string       `json:"name"`
	URL     string       `json:"url"`
	Headers []HTTPHeader `json:"headers"`
	Meta    Meta         `json:"_meta,omitempty"`
}

// MCPServerSSE connects to an MCP server over server-sent events.
// Requires the sse MCP capability.
type MCPServerSSE struct {
	Name    string       `json:"name"`
	URL     string       `json:"url"`
	Headers []HTTPHeader `json:"headers"`
	Meta    Meta         `json:"_meta,omitempty"`
}

// MCPServerStdio launches an MCP server as a subprocess. All agents
// must support this transport.
type MCPServerStdio struct {
	Name    string        `json:"name"`
	Command string        `json:"command"`
	Args    []string      `json:"args"`
	Env     []EnvVariable `json:"env"`
	Meta    Meta          `json:"_meta,omitempty"`
}

// MarshalJSON emits the populated variant. HTTP and SSE carry a type
// tag; stdio does not.
func (s MCPServer) MarshalJSON() ([]byte, error) {
	if s.Raw != nil && s.HTTP == nil && s.SSE == nil && s.Stdio == nil {
		return s.Raw, nil
	}
	typ := s.Type
	if typ == "" {
		switch {
		case s.HTTP != nil:
			typ = MCPTypeHTTP
		case s.SSE != nil:
			typ = MCPTypeSSE
		case s.Stdio != nil:
			typ = MCPTypeStdio
		}
	}
	switch typ {
	case MCPTypeHTTP:
		return marshalTagged("type", string(typ), s.HTTP)
	case MCPTypeSSE:
		return marshalTagged("type", string(typ), s.SSE)
	case MCPTypeStdio:
		return json.Marshal(s.Stdio)
	default:
		if s.Raw != nil {
			return s.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown MCP server type %q", s.Type)
	}
}

// UnmarshalJSON decodes an MCP server config by its type field, or as
// stdio when no type is present.
func (s *MCPServer) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	*s = MCPServer{Type: MCPServerType(tag)}
	switch s.Type {
	case MCPTypeHTTP:
		s.HTTP = new(MCPServerHTTP)
		err = json.Unmarshal(data, s.HTTP)
	case MCPTypeSSE:
		s.SSE = new(MCPServerSSE)
		err = json.Unmarshal(data, s.SSE)
	case "":
		s.Type = MCPTypeStdio
		s.Stdio = new(MCPServerStdio)
		err = json.Unmarshal(data, s.Stdio)
	default:
		s.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}
