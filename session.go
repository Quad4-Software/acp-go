// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// NewSessionRequest asks the agent to create a session.
type NewSessionRequest struct {
	CWD                   string      `json:"cwd"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
	MCPServers            []MCPServer `json:"mcpServers"`
	Meta                  Meta        `json:"_meta,omitempty"`
}

// NewSessionResponse returns the created session.
type NewSessionResponse struct {
	SessionID     SessionID             `json:"sessionId"`
	Modes         *SessionModeState     `json:"modes,omitempty"`
	ConfigOptions []SessionConfigOption `json:"configOptions,omitempty"`
	Meta          Meta                  `json:"_meta,omitempty"`
}

// LoadSessionRequest resumes an existing session. Requires the
// LoadSession agent capability.
type LoadSessionRequest struct {
	SessionID             SessionID   `json:"sessionId"`
	CWD                   string      `json:"cwd"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
	MCPServers            []MCPServer `json:"mcpServers"`
	Meta                  Meta        `json:"_meta,omitempty"`
}

// LoadSessionResponse reports session state after loading.
type LoadSessionResponse struct {
	Modes         *SessionModeState     `json:"modes,omitempty"`
	ConfigOptions []SessionConfigOption `json:"configOptions,omitempty"`
	Meta          Meta                  `json:"_meta,omitempty"`
}

// ResumeSessionRequest resumes a session without replaying history.
// Requires the resume session capability.
type ResumeSessionRequest struct {
	SessionID             SessionID   `json:"sessionId"`
	CWD                   string      `json:"cwd"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
	MCPServers            []MCPServer `json:"mcpServers"`
	Meta                  Meta        `json:"_meta,omitempty"`
}

// ResumeSessionResponse reports session state after resuming.
type ResumeSessionResponse struct {
	Modes         *SessionModeState     `json:"modes,omitempty"`
	ConfigOptions []SessionConfigOption `json:"configOptions,omitempty"`
	Meta          Meta                  `json:"_meta,omitempty"`
}

// CloseSessionRequest frees resources of an active session. Requires
// the close session capability.
type CloseSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
	Meta      Meta      `json:"_meta,omitempty"`
}

// CloseSessionResponse acknowledges the session close.
type CloseSessionResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}

// ListSessionsRequest lists existing sessions, optionally filtered by
// working directory. Requires the list session capability.
type ListSessionsRequest struct {
	CWD    string `json:"cwd,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Meta   Meta   `json:"_meta,omitempty"`
}

// ListSessionsResponse returns a page of sessions.
type ListSessionsResponse struct {
	Sessions   []SessionInfo `json:"sessions"`
	NextCursor string        `json:"nextCursor,omitempty"`
	Meta       Meta          `json:"_meta,omitempty"`
}

// SessionInfo describes a stored session.
type SessionInfo struct {
	SessionID             SessionID `json:"sessionId"`
	CWD                   string    `json:"cwd"`
	AdditionalDirectories []string  `json:"additionalDirectories,omitempty"`
	Title                 string    `json:"title,omitempty"`
	UpdatedAt             string    `json:"updatedAt,omitempty"`
	Meta                  Meta      `json:"_meta,omitempty"`
}

// DeleteSessionRequest removes a session from history. Requires the
// delete session capability.
type DeleteSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
	Meta      Meta      `json:"_meta,omitempty"`
}

// DeleteSessionResponse acknowledges the session delete.
type DeleteSessionResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SetSessionModeRequest switches the session's operating mode.
type SetSessionModeRequest struct {
	SessionID SessionID     `json:"sessionId"`
	ModeID    SessionModeID `json:"modeId"`
	Meta      Meta          `json:"_meta,omitempty"`
}

// SetSessionModeResponse acknowledges the mode switch.
type SetSessionModeResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}

// SessionMode is an operating mode the agent supports.
type SessionMode struct {
	ID          SessionModeID `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Meta        Meta          `json:"_meta,omitempty"`
}

// SessionModeState lists available modes and the current one.
type SessionModeState struct {
	CurrentModeID  SessionModeID `json:"currentModeId"`
	AvailableModes []SessionMode `json:"availableModes"`
	Meta           Meta          `json:"_meta,omitempty"`
}

// PromptRequest sends user content to the agent, starting a turn.
type PromptRequest struct {
	SessionID SessionID      `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
	Meta      Meta           `json:"_meta,omitempty"`
}

// StopReason explains why the agent stopped a prompt turn.
type StopReason string

// Stop reasons.
const (
	StopEndTurn         StopReason = "end_turn"
	StopMaxTokens       StopReason = "max_tokens"
	StopMaxTurnRequests StopReason = "max_turn_requests"
	StopRefusal         StopReason = "refusal"
	StopCancelled       StopReason = "cancelled"
)

// PromptResponse ends a prompt turn.
type PromptResponse struct {
	StopReason StopReason `json:"stopReason"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// SessionConfigOptionCategory is a semantic hint for where a config
// option belongs in the UI. Values starting with an underscore are
// custom extensions.
type SessionConfigOptionCategory string

// Session config option categories.
const (
	ConfigCategoryMode         SessionConfigOptionCategory = "mode"
	ConfigCategoryModel        SessionConfigOptionCategory = "model"
	ConfigCategoryModelConfig  SessionConfigOptionCategory = "model_config"
	ConfigCategoryThoughtLevel SessionConfigOptionCategory = "thought_level"
)

// SessionConfigOptionType discriminates SessionConfigOption variants.
type SessionConfigOptionType string

// Session config option types.
const (
	ConfigOptionSelect  SessionConfigOptionType = "select"
	ConfigOptionBoolean SessionConfigOptionType = "boolean"
)

// SessionConfigOption is a session configuration selector and its
// current state. Exactly one variant field is populated, matching Type.
type SessionConfigOption struct {
	ID          SessionConfigID
	Name        string
	Description string
	Category    SessionConfigOptionCategory
	Meta        Meta
	Type        SessionConfigOptionType
	Select      *SessionConfigSelect
	Boolean     *SessionConfigBoolean

	// Raw preserves the original JSON of an unrecognized option type.
	Raw json.RawMessage
}

// SessionConfigSelect is a single-value selector.
type SessionConfigSelect struct {
	CurrentValue SessionConfigValueID       `json:"currentValue"`
	Options      SessionConfigSelectOptions `json:"options"`
}

// SessionConfigBoolean is an on/off toggle.
type SessionConfigBoolean struct {
	CurrentValue bool `json:"currentValue"`
}

// SessionConfigSelectOption is one selectable value.
type SessionConfigSelectOption struct {
	Value       SessionConfigValueID `json:"value"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Meta        Meta                 `json:"_meta,omitempty"`
}

// SessionConfigSelectGroup is a named group of selectable values.
type SessionConfigSelectGroup struct {
	Group   SessionConfigGroupID        `json:"group"`
	Name    string                      `json:"name"`
	Options []SessionConfigSelectOption `json:"options"`
	Meta    Meta                        `json:"_meta,omitempty"`
}

// SessionConfigSelectOptions holds either a flat option list or
// grouped options.
type SessionConfigSelectOptions struct {
	Options []SessionConfigSelectOption
	Groups  []SessionConfigSelectGroup
}

// MarshalJSON emits the grouped or flat variant.
func (o SessionConfigSelectOptions) MarshalJSON() ([]byte, error) {
	if o.Groups != nil {
		return json.Marshal(o.Groups)
	}
	if o.Options == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(o.Options)
}

// UnmarshalJSON decodes grouped or flat options by shape.
func (o *SessionConfigSelectOptions) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*o = SessionConfigSelectOptions{Options: []SessionConfigSelectOption{}}
	if len(raw) == 0 {
		return nil
	}
	var probe struct {
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(raw[0], &probe); err != nil {
		return err
	}
	if probe.Options != nil {
		return json.Unmarshal(b, &o.Groups)
	}
	return json.Unmarshal(b, &o.Options)
}

type sessionConfigOptionBase struct {
	ID          SessionConfigID             `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Category    SessionConfigOptionCategory `json:"category,omitempty"`
	Meta        Meta                        `json:"_meta,omitempty"`
}

// MarshalJSON emits the option with its type tag and variant fields.
func (o SessionConfigOption) MarshalJSON() ([]byte, error) {
	if o.Raw != nil && o.Select == nil && o.Boolean == nil {
		return o.Raw, nil
	}
	base, err := json.Marshal(sessionConfigOptionBase{
		ID: o.ID, Name: o.Name, Description: o.Description,
		Category: o.Category, Meta: o.Meta,
	})
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}
	typ := o.Type
	if typ == "" {
		switch {
		case o.Select != nil:
			typ = ConfigOptionSelect
		case o.Boolean != nil:
			typ = ConfigOptionBoolean
		}
	}
	var tag string
	var v any
	switch typ {
	case ConfigOptionSelect:
		tag, v = string(typ), o.Select
	case ConfigOptionBoolean:
		tag, v = string(typ), o.Boolean
	default:
		if o.Raw != nil {
			return o.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown session config option type %q", o.Type)
	}
	variant, err := marshalTagged("type", tag, v)
	if err != nil {
		return nil, err
	}
	var vm map[string]json.RawMessage
	if err := json.Unmarshal(variant, &vm); err != nil {
		return nil, err
	}
	for k, val := range vm {
		m[k] = val
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes an option by its type field.
func (o *SessionConfigOption) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	var base sessionConfigOptionBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	*o = SessionConfigOption{
		ID: base.ID, Name: base.Name, Description: base.Description,
		Category: base.Category, Meta: base.Meta,
		Type: SessionConfigOptionType(tag),
	}
	switch o.Type {
	case ConfigOptionSelect:
		o.Select = new(SessionConfigSelect)
		err = json.Unmarshal(data, o.Select)
	case ConfigOptionBoolean:
		o.Boolean = new(SessionConfigBoolean)
		err = json.Unmarshal(data, o.Boolean)
	default:
		o.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// SetSessionConfigOptionRequest changes a session config option. A
// boolean option carries Type "boolean" and BoolValue. Any other value
// uses the default string value form with Value set; unknown Type
// values are preserved for extension compatibility.
type SetSessionConfigOptionRequest struct {
	SessionID SessionID
	ConfigID  SessionConfigID
	Type      string
	Value     SessionConfigValueID
	BoolValue *bool
	Meta      Meta
}

// NewSetConfigBoolRequest builds a boolean set-config request.
func NewSetConfigBoolRequest(sid SessionID, cid SessionConfigID, v bool) SetSessionConfigOptionRequest {
	return SetSessionConfigOptionRequest{SessionID: sid, ConfigID: cid, Type: "boolean", BoolValue: &v}
}

// NewSetConfigValueRequest builds a value-id set-config request.
func NewSetConfigValueRequest(sid SessionID, cid SessionConfigID, v SessionConfigValueID) SetSessionConfigOptionRequest {
	return SetSessionConfigOptionRequest{SessionID: sid, ConfigID: cid, Value: v}
}

// MarshalJSON emits the boolean or string value form.
func (r SetSessionConfigOptionRequest) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"sessionId": r.SessionID,
		"configId":  r.ConfigID,
	}
	if r.Meta != nil {
		m["_meta"] = r.Meta
	}
	if r.BoolValue != nil || r.Type == "boolean" {
		m["type"] = "boolean"
		m["value"] = r.BoolValue != nil && *r.BoolValue
		return json.Marshal(m)
	}
	m["value"] = r.Value
	if r.Type != "" {
		m["type"] = r.Type
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes either the boolean or string value form.
func (r *SetSessionConfigOptionRequest) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if v, ok := m["sessionId"]; ok {
		if err := json.Unmarshal(v, &r.SessionID); err != nil {
			return err
		}
	}
	if v, ok := m["configId"]; ok {
		if err := json.Unmarshal(v, &r.ConfigID); err != nil {
			return err
		}
	}
	if v, ok := m["_meta"]; ok {
		if err := json.Unmarshal(v, &r.Meta); err != nil {
			return err
		}
	}
	if v, ok := m["type"]; ok {
		if err := json.Unmarshal(v, &r.Type); err != nil {
			return err
		}
	}
	if v, ok := m["value"]; ok && r.Type == "boolean" {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return err
		}
		r.BoolValue = &b
		return nil
	}
	if v, ok := m["value"]; ok {
		if err := json.Unmarshal(v, &r.Value); err != nil {
			return err
		}
	}
	return nil
}

// SetSessionConfigOptionResponse reports the session's config options
// after a change.
type SetSessionConfigOptionResponse struct {
	ConfigOptions []SessionConfigOption `json:"configOptions"`
	Meta          Meta                  `json:"_meta,omitempty"`
}
