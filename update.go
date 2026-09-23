// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// SessionUpdateType discriminates SessionUpdate variants.
type SessionUpdateType string

// Session update types.
const (
	UpdateUserMessageChunk  SessionUpdateType = "user_message_chunk"
	UpdateAgentMessageChunk SessionUpdateType = "agent_message_chunk"
	UpdateAgentThoughtChunk SessionUpdateType = "agent_thought_chunk"
	UpdateToolCall          SessionUpdateType = "tool_call"
	UpdateToolCallUpdate    SessionUpdateType = "tool_call_update"
	UpdatePlan              SessionUpdateType = "plan"
	UpdateAvailableCommands SessionUpdateType = "available_commands_update"
	UpdateCurrentMode       SessionUpdateType = "current_mode_update"
	UpdateConfigOptions     SessionUpdateType = "config_option_update"
	UpdateSessionInfo       SessionUpdateType = "session_info_update"
	UpdateUsage             SessionUpdateType = "usage_update"
)

// SessionUpdate is a real-time progress update the agent sends during
// session processing. Exactly one variant field is populated, matching
// Update. Updates received with an unrecognized type are preserved in
// Raw.
type SessionUpdate struct {
	Update            SessionUpdateType
	UserMessageChunk  *ContentChunk
	AgentMessageChunk *ContentChunk
	AgentThoughtChunk *ContentChunk
	ToolCall          *ToolCall
	ToolCallUpdate    *ToolCallUpdate
	Plan              *Plan
	AvailableCommands *AvailableCommandsUpdate
	CurrentMode       *CurrentModeUpdate
	ConfigOptions     *ConfigOptionUpdate
	SessionInfo       *SessionInfoUpdate
	Usage             *UsageUpdate

	// Raw preserves the original JSON of an unrecognized update type.
	// Set it directly to send a non-standard update.
	Raw json.RawMessage
}

// AvailableCommand is a slash command the client can offer.
type AvailableCommand struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       *AvailableCommandInput `json:"input,omitempty"`
	Meta        Meta                   `json:"_meta,omitempty"`
}

// AvailableCommandInput describes the input a command accepts. The only
// defined variant is an unstructured hint.
type AvailableCommandInput struct {
	Hint string `json:"hint"`
	Meta Meta   `json:"_meta,omitempty"`
}

// AvailableCommandsUpdate reports the current slash commands.
type AvailableCommandsUpdate struct {
	AvailableCommands []AvailableCommand `json:"availableCommands"`
	Meta              Meta               `json:"_meta,omitempty"`
}

// CurrentModeUpdate reports the session's current mode.
type CurrentModeUpdate struct {
	CurrentModeID SessionModeID `json:"currentModeId"`
	Meta          Meta          `json:"_meta,omitempty"`
}

// ConfigOptionUpdate reports the session's current config options.
type ConfigOptionUpdate struct {
	ConfigOptions []SessionConfigOption `json:"configOptions"`
	Meta          Meta                  `json:"_meta,omitempty"`
}

// SessionInfoUpdate reports changed session metadata.
type SessionInfoUpdate struct {
	Title     string `json:"title,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Meta      Meta   `json:"_meta,omitempty"`
}

// Cost is a monetary amount associated with session usage.
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Meta     Meta    `json:"_meta,omitempty"`
}

// UsageUpdate reports context window usage for a session.
type UsageUpdate struct {
	Used int64 `json:"used"`
	Size int64 `json:"size"`
	Cost *Cost `json:"cost,omitempty"`
	Meta Meta  `json:"_meta,omitempty"`
}

// NewUserMessageChunkUpdate returns an update streaming user text.
func NewUserMessageChunkUpdate(c ContentBlock) SessionUpdate {
	return SessionUpdate{Update: UpdateUserMessageChunk, UserMessageChunk: &ContentChunk{Content: c}}
}

// NewAgentMessageChunkUpdate returns an update streaming agent output.
func NewAgentMessageChunkUpdate(c ContentBlock) SessionUpdate {
	return SessionUpdate{Update: UpdateAgentMessageChunk, AgentMessageChunk: &ContentChunk{Content: c}}
}

// NewAgentThoughtChunkUpdate returns an update streaming agent reasoning.
func NewAgentThoughtChunkUpdate(c ContentBlock) SessionUpdate {
	return SessionUpdate{Update: UpdateAgentThoughtChunk, AgentThoughtChunk: &ContentChunk{Content: c}}
}

// NewToolCallUpdate returns an update reporting a new tool call.
func NewToolCallUpdate(tc ToolCall) SessionUpdate {
	return SessionUpdate{Update: UpdateToolCall, ToolCall: &tc}
}

// NewToolCallProgressUpdate returns an update for an existing tool call.
func NewToolCallProgressUpdate(tc ToolCallUpdate) SessionUpdate {
	return SessionUpdate{Update: UpdateToolCallUpdate, ToolCallUpdate: &tc}
}

// NewPlanUpdate returns an update carrying the agent's plan.
func NewPlanUpdate(p Plan) SessionUpdate {
	return SessionUpdate{Update: UpdatePlan, Plan: &p}
}

// NewAvailableCommandsUpdate returns an update listing slash commands.
func NewAvailableCommandsUpdate(cmds []AvailableCommand) SessionUpdate {
	return SessionUpdate{Update: UpdateAvailableCommands, AvailableCommands: &AvailableCommandsUpdate{AvailableCommands: cmds}}
}

// NewCurrentModeUpdate returns an update reporting the active mode.
func NewCurrentModeUpdate(id SessionModeID) SessionUpdate {
	return SessionUpdate{Update: UpdateCurrentMode, CurrentMode: &CurrentModeUpdate{CurrentModeID: id}}
}

// NewConfigOptionsUpdate returns an update listing config options.
func NewConfigOptionsUpdate(opts []SessionConfigOption) SessionUpdate {
	return SessionUpdate{Update: UpdateConfigOptions, ConfigOptions: &ConfigOptionUpdate{ConfigOptions: opts}}
}

// NewUsageUpdate returns an update reporting context usage.
func NewUsageUpdate(u UsageUpdate) SessionUpdate {
	return SessionUpdate{Update: UpdateUsage, Usage: &u}
}

// TypeOf returns the update's effective type, inferring it from the
// populated variant when Update is unset.
func (u SessionUpdate) TypeOf() SessionUpdateType {
	if u.Update != "" {
		return u.Update
	}
	switch {
	case u.UserMessageChunk != nil:
		return UpdateUserMessageChunk
	case u.AgentMessageChunk != nil:
		return UpdateAgentMessageChunk
	case u.AgentThoughtChunk != nil:
		return UpdateAgentThoughtChunk
	case u.ToolCall != nil:
		return UpdateToolCall
	case u.ToolCallUpdate != nil:
		return UpdateToolCallUpdate
	case u.Plan != nil:
		return UpdatePlan
	case u.AvailableCommands != nil:
		return UpdateAvailableCommands
	case u.CurrentMode != nil:
		return UpdateCurrentMode
	case u.ConfigOptions != nil:
		return UpdateConfigOptions
	case u.SessionInfo != nil:
		return UpdateSessionInfo
	case u.Usage != nil:
		return UpdateUsage
	default:
		return ""
	}
}

// MarshalJSON emits the populated variant with its sessionUpdate tag.
func (u SessionUpdate) MarshalJSON() ([]byte, error) {
	if u.Raw != nil && u.UserMessageChunk == nil && u.AgentMessageChunk == nil &&
		u.AgentThoughtChunk == nil && u.ToolCall == nil && u.ToolCallUpdate == nil &&
		u.Plan == nil && u.AvailableCommands == nil && u.CurrentMode == nil &&
		u.ConfigOptions == nil && u.SessionInfo == nil && u.Usage == nil {
		return u.Raw, nil
	}
	var tag string
	var v any
	switch t := u.TypeOf(); t {
	case UpdateUserMessageChunk:
		tag, v = string(t), u.UserMessageChunk
	case UpdateAgentMessageChunk:
		tag, v = string(t), u.AgentMessageChunk
	case UpdateAgentThoughtChunk:
		tag, v = string(t), u.AgentThoughtChunk
	case UpdateToolCall:
		tag, v = string(t), u.ToolCall
	case UpdateToolCallUpdate:
		tag, v = string(t), u.ToolCallUpdate
	case UpdatePlan:
		tag, v = string(t), u.Plan
	case UpdateAvailableCommands:
		tag, v = string(t), u.AvailableCommands
	case UpdateCurrentMode:
		tag, v = string(t), u.CurrentMode
	case UpdateConfigOptions:
		tag, v = string(t), u.ConfigOptions
	case UpdateSessionInfo:
		tag, v = string(t), u.SessionInfo
	case UpdateUsage:
		tag, v = string(t), u.Usage
	default:
		if u.Raw != nil {
			return u.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown session update type %q", u.Update)
	}
	return marshalTagged("sessionUpdate", tag, v)
}

// UnmarshalJSON decodes an update by its sessionUpdate field.
func (u *SessionUpdate) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "sessionUpdate")
	if err != nil {
		return err
	}
	*u = SessionUpdate{Update: SessionUpdateType(tag)}
	switch u.Update {
	case UpdateUserMessageChunk:
		u.UserMessageChunk = new(ContentChunk)
		err = json.Unmarshal(data, u.UserMessageChunk)
	case UpdateAgentMessageChunk:
		u.AgentMessageChunk = new(ContentChunk)
		err = json.Unmarshal(data, u.AgentMessageChunk)
	case UpdateAgentThoughtChunk:
		u.AgentThoughtChunk = new(ContentChunk)
		err = json.Unmarshal(data, u.AgentThoughtChunk)
	case UpdateToolCall:
		u.ToolCall = new(ToolCall)
		err = json.Unmarshal(data, u.ToolCall)
	case UpdateToolCallUpdate:
		u.ToolCallUpdate = new(ToolCallUpdate)
		err = json.Unmarshal(data, u.ToolCallUpdate)
	case UpdatePlan:
		u.Plan = new(Plan)
		err = json.Unmarshal(data, u.Plan)
	case UpdateAvailableCommands:
		u.AvailableCommands = new(AvailableCommandsUpdate)
		err = json.Unmarshal(data, u.AvailableCommands)
	case UpdateCurrentMode:
		u.CurrentMode = new(CurrentModeUpdate)
		err = json.Unmarshal(data, u.CurrentMode)
	case UpdateConfigOptions:
		u.ConfigOptions = new(ConfigOptionUpdate)
		err = json.Unmarshal(data, u.ConfigOptions)
	case UpdateSessionInfo:
		u.SessionInfo = new(SessionInfoUpdate)
		err = json.Unmarshal(data, u.SessionInfo)
	case UpdateUsage:
		u.Usage = new(UsageUpdate)
		err = json.Unmarshal(data, u.Usage)
	default:
		u.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// SessionNotification is the params payload of the session/update
// notification.
type SessionNotification struct {
	SessionID SessionID     `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
	Meta      Meta          `json:"_meta,omitempty"`
}

// CancelNotification is the params payload of the session/cancel
// notification.
type CancelNotification struct {
	SessionID SessionID `json:"sessionId"`
	Meta      Meta      `json:"_meta,omitempty"`
}
