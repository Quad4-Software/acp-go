// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"fmt"
)

// ToolKind categorizes a tool call so clients can pick icons and
// optimize progress display.
type ToolKind string

// Tool kinds.
const (
	ToolKindRead       ToolKind = "read"
	ToolKindEdit       ToolKind = "edit"
	ToolKindDelete     ToolKind = "delete"
	ToolKindMove       ToolKind = "move"
	ToolKindSearch     ToolKind = "search"
	ToolKindExecute    ToolKind = "execute"
	ToolKindThink      ToolKind = "think"
	ToolKindFetch      ToolKind = "fetch"
	ToolKindSwitchMode ToolKind = "switch_mode"
	ToolKindOther      ToolKind = "other"
)

// ToolCallStatus is the execution state of a tool call.
type ToolCallStatus string

// Tool call statuses.
const (
	ToolCallPending    ToolCallStatus = "pending"
	ToolCallInProgress ToolCallStatus = "in_progress"
	ToolCallCompleted  ToolCallStatus = "completed"
	ToolCallFailed     ToolCallStatus = "failed"
)

// ToolCall reports a new tool invocation to the client.
type ToolCall struct {
	ToolCallID ToolCallID         `json:"toolCallId"`
	Title      string             `json:"title"`
	Name       string             `json:"name,omitempty"`
	Kind       ToolKind           `json:"kind,omitempty"`
	Status     ToolCallStatus     `json:"status,omitempty"`
	Content    []ToolCallContent  `json:"content,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
	RawInput   any                `json:"rawInput,omitempty"`
	RawOutput  any                `json:"rawOutput,omitempty"`
	Meta       Meta               `json:"_meta,omitempty"`
}

// ToolCallUpdate reports progress or results for an existing tool call.
// Only the fields being changed need to be set.
type ToolCallUpdate struct {
	ToolCallID ToolCallID         `json:"toolCallId"`
	Title      string             `json:"title,omitempty"`
	Name       string             `json:"name,omitempty"`
	Kind       ToolKind           `json:"kind,omitempty"`
	Status     ToolCallStatus     `json:"status,omitempty"`
	Content    []ToolCallContent  `json:"content,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
	RawInput   any                `json:"rawInput,omitempty"`
	RawOutput  any                `json:"rawOutput,omitempty"`
	Meta       Meta               `json:"_meta,omitempty"`
}

// ToolCallLocation points at a file the tool call touches. Line numbers
// are 1-based.
type ToolCallLocation struct {
	Path string `json:"path"`
	Line *int64 `json:"line,omitempty"`
	Meta Meta   `json:"_meta,omitempty"`
}

// ToolCallContentType discriminates ToolCallContent variants.
type ToolCallContentType string

// Tool call content types.
const (
	ToolCallContentBlock    ToolCallContentType = "content"
	ToolCallContentDiff     ToolCallContentType = "diff"
	ToolCallContentTerminal ToolCallContentType = "terminal"
)

// ToolCallContent is content produced by a tool call. Exactly one
// variant field is populated, matching Type.
type ToolCallContent struct {
	Type     ToolCallContentType
	Content  *Content
	Diff     *Diff
	Terminal *Terminal

	// Raw preserves the original JSON of an unrecognized content type.
	Raw json.RawMessage
}

// Content wraps a ContentBlock inside tool call output.
type Content struct {
	Content ContentBlock `json:"content"`
	Meta    Meta         `json:"_meta,omitempty"`
}

// Diff is a file modification shown as a diff. A nil OldText means the
// file did not exist before.
type Diff struct {
	Path    string  `json:"path"`
	NewText string  `json:"newText"`
	OldText *string `json:"oldText,omitempty"`
	Meta    Meta    `json:"_meta,omitempty"`
}

// Terminal embeds a terminal created with terminal/create by ID.
type Terminal struct {
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// NewContentToolCallContent wraps a content block as tool call output.
func NewContentToolCallContent(b ContentBlock) ToolCallContent {
	return ToolCallContent{Type: ToolCallContentBlock, Content: &Content{Content: b}}
}

// NewDiffToolCallContent wraps a diff as tool call output.
func NewDiffToolCallContent(d Diff) ToolCallContent {
	return ToolCallContent{Type: ToolCallContentDiff, Diff: &d}
}

// NewTerminalToolCallContent embeds a terminal as tool call output.
func NewTerminalToolCallContent(id TerminalID) ToolCallContent {
	return ToolCallContent{Type: ToolCallContentTerminal, Terminal: &Terminal{TerminalID: id}}
}

// TypeOf returns the content's effective type, inferring it from the
// populated variant when Type is unset.
func (c ToolCallContent) TypeOf() ToolCallContentType {
	if c.Type != "" {
		return c.Type
	}
	switch {
	case c.Content != nil:
		return ToolCallContentBlock
	case c.Diff != nil:
		return ToolCallContentDiff
	case c.Terminal != nil:
		return ToolCallContentTerminal
	default:
		return ""
	}
}

// MarshalJSON emits the populated variant with its type tag.
func (c ToolCallContent) MarshalJSON() ([]byte, error) {
	if c.Raw != nil && c.Content == nil && c.Diff == nil && c.Terminal == nil {
		return c.Raw, nil
	}
	var tag string
	var v any
	switch t := c.TypeOf(); t {
	case ToolCallContentBlock:
		tag, v = string(t), c.Content
	case ToolCallContentDiff:
		tag, v = string(t), c.Diff
	case ToolCallContentTerminal:
		tag, v = string(t), c.Terminal
	default:
		if c.Raw != nil {
			return c.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown tool call content type %q", c.Type)
	}
	return marshalTagged("type", tag, v)
}

// UnmarshalJSON decodes tool call content by its type field.
func (c *ToolCallContent) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "type")
	if err != nil {
		return err
	}
	*c = ToolCallContent{Type: ToolCallContentType(tag)}
	switch c.Type {
	case ToolCallContentBlock:
		c.Content = new(Content)
		err = json.Unmarshal(data, c.Content)
	case ToolCallContentDiff:
		c.Diff = new(Diff)
		err = json.Unmarshal(data, c.Diff)
	case ToolCallContentTerminal:
		c.Terminal = new(Terminal)
		err = json.Unmarshal(data, c.Terminal)
	default:
		c.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}

// PermissionOptionKind describes the effect of a permission option.
type PermissionOptionKind string

// Permission option kinds.
const (
	PermissionAllowOnce    PermissionOptionKind = "allow_once"
	PermissionAllowAlways  PermissionOptionKind = "allow_always"
	PermissionRejectOnce   PermissionOptionKind = "reject_once"
	PermissionRejectAlways PermissionOptionKind = "reject_always"
)

// PermissionOption is one choice offered in a permission request.
type PermissionOption struct {
	OptionID PermissionOptionID   `json:"optionId"`
	Name     string               `json:"name"`
	Kind     PermissionOptionKind `json:"kind"`
	Meta     Meta                 `json:"_meta,omitempty"`
}

// RequestPermissionRequest asks the client to authorize a tool call.
type RequestPermissionRequest struct {
	SessionID SessionID          `json:"sessionId"`
	ToolCall  ToolCallUpdate     `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
	Meta      Meta               `json:"_meta,omitempty"`
}

// RequestPermissionResponse carries the user's decision.
type RequestPermissionResponse struct {
	Outcome RequestPermissionOutcome `json:"outcome"`
	Meta    Meta                     `json:"_meta,omitempty"`
}

// RequestPermissionOutcomeType discriminates RequestPermissionOutcome
// variants.
type RequestPermissionOutcomeType string

// Permission outcomes.
const (
	PermissionOutcomeCancelled RequestPermissionOutcomeType = "cancelled"
	PermissionOutcomeSelected  RequestPermissionOutcomeType = "selected"
)

// RequestPermissionOutcome is the result of a permission request:
// either the turn was cancelled or the user selected an option.
type RequestPermissionOutcome struct {
	Outcome  RequestPermissionOutcomeType
	Selected *SelectedPermissionOutcome

	// Raw preserves the original JSON of an unrecognized outcome.
	Raw json.RawMessage
}

// SelectedPermissionOutcome holds the chosen option ID.
type SelectedPermissionOutcome struct {
	OptionID PermissionOptionID `json:"optionId"`
	Meta     Meta               `json:"_meta,omitempty"`
}

// NewCancelledOutcome returns a cancelled permission outcome.
func NewCancelledOutcome() RequestPermissionOutcome {
	return RequestPermissionOutcome{Outcome: PermissionOutcomeCancelled}
}

// NewSelectedOutcome returns a selected permission outcome.
func NewSelectedOutcome(id PermissionOptionID) RequestPermissionOutcome {
	return RequestPermissionOutcome{
		Outcome:  PermissionOutcomeSelected,
		Selected: &SelectedPermissionOutcome{OptionID: id},
	}
}

// MarshalJSON emits the populated variant with its outcome tag. An
// outcome with a Selected option and no explicit Outcome marshals as
// selected.
func (o RequestPermissionOutcome) MarshalJSON() ([]byte, error) {
	if o.Raw != nil && o.Selected == nil && o.Outcome != PermissionOutcomeCancelled {
		return o.Raw, nil
	}
	outcome := o.Outcome
	if outcome == "" && o.Selected != nil {
		outcome = PermissionOutcomeSelected
	}
	switch outcome {
	case PermissionOutcomeSelected:
		return marshalTagged("outcome", string(outcome), o.Selected)
	case PermissionOutcomeCancelled, "":
		return marshalTagged("outcome", string(PermissionOutcomeCancelled), nil)
	default:
		if o.Raw != nil {
			return o.Raw, nil
		}
		return nil, fmt.Errorf("acp: unknown permission outcome %q", o.Outcome)
	}
}

// UnmarshalJSON decodes a permission outcome by its outcome field.
func (o *RequestPermissionOutcome) UnmarshalJSON(data []byte) error {
	tag, err := readTag(data, "outcome")
	if err != nil {
		return err
	}
	*o = RequestPermissionOutcome{Outcome: RequestPermissionOutcomeType(tag)}
	switch o.Outcome {
	case PermissionOutcomeSelected:
		o.Selected = new(SelectedPermissionOutcome)
		err = json.Unmarshal(data, o.Selected)
	case PermissionOutcomeCancelled:
	default:
		o.Raw = append(json.RawMessage(nil), data...)
	}
	return err
}
