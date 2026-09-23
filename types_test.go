// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"testing"
)

func roundTrip[T any](t *testing.T, v T) T {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	return out
}

func TestContentBlockText(t *testing.T) {
	in := NewTextBlock("hello")
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"text":"hello","type":"text"}`
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	var w map[string]any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "text" || got["text"] != "hello" {
		t.Fatalf("bad wire form: %s", b)
	}
	out := roundTrip(t, in)
	if out.Type != ContentBlockText || out.Text == nil || out.Text.Text != "hello" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestContentBlockUnknownType(t *testing.T) {
	raw := `{"type":"_custom","foo":1}`
	var b ContentBlock
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		t.Fatal(err)
	}
	if b.Raw == nil {
		t.Fatal("raw not preserved")
	}
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != raw {
		t.Fatalf("got %s, want %s", out, raw)
	}
}

func TestSessionUpdateRoundTrip(t *testing.T) {
	u := NewAgentMessageChunkUpdate(NewTextBlock("chunk"))
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["sessionUpdate"] != "agent_message_chunk" {
		t.Fatalf("bad wire form: %s", b)
	}
	out := roundTrip(t, u)
	if out.Update != UpdateAgentMessageChunk || out.AgentMessageChunk == nil {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestSessionUpdateToolCall(t *testing.T) {
	u := NewToolCallUpdate(ToolCall{
		ToolCallID: "tc1",
		Title:      "Read file",
		Kind:       ToolKindRead,
		Status:     ToolCallCompleted,
		Content:    []ToolCallContent{NewDiffToolCallContent(Diff{Path: "/a", NewText: "x"})},
	})
	out := roundTrip(t, u)
	if out.ToolCall == nil || out.ToolCall.ToolCallID != "tc1" {
		t.Fatalf("round trip failed: %+v", out)
	}
	if out.ToolCall.Content[0].Diff == nil || out.ToolCall.Content[0].Diff.Path != "/a" {
		t.Fatalf("diff lost: %+v", out.ToolCall.Content[0])
	}
}

func TestMCPServerStdioHasNoTypeTag(t *testing.T) {
	s := MCPServer{Type: MCPTypeStdio, Stdio: &MCPServerStdio{
		Name: "fs", Command: "mcp-fs", Args: []string{"--root", "/"},
	}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, hasType := m["type"]; hasType {
		t.Fatalf("stdio server must not carry a type tag: %s", b)
	}
	out := roundTrip(t, s)
	if out.Stdio == nil || out.Stdio.Command != "mcp-fs" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestMCPServerHTTP(t *testing.T) {
	s := MCPServer{Type: MCPTypeHTTP, HTTP: &MCPServerHTTP{
		Name: "web", URL: "https://mcp.example.com",
		Headers: []HTTPHeader{{Name: "Authorization", Value: "x"}},
	}}
	out := roundTrip(t, s)
	if out.HTTP == nil || out.HTTP.URL != "https://mcp.example.com" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestAuthMethodRoundTrip(t *testing.T) {
	term := AuthMethod{Type: AuthMethodTerminalType, Terminal: &AuthMethodTerminal{
		ID: "oauth", Name: "OAuth", Args: []string{"login"},
	}}
	out := roundTrip(t, term)
	if out.Terminal == nil || out.Terminal.ID != "oauth" {
		t.Fatalf("round trip failed: %+v", out)
	}

	agent := AuthMethod{Agent: &AuthMethodAgent{ID: "agent", Name: "Agent auth"}}
	b, _ := json.Marshal(agent)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, hasType := m["type"]; hasType {
		t.Fatalf("agent auth method must not carry a type tag: %s", b)
	}
	out = roundTrip(t, agent)
	if out.Agent == nil || out.Agent.Name != "Agent auth" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestPermissionOutcome(t *testing.T) {
	out := roundTrip(t, NewSelectedOutcome("allow"))
	if out.Selected == nil || out.Selected.OptionID != "allow" {
		t.Fatalf("round trip failed: %+v", out)
	}
	b, _ := json.Marshal(NewCancelledOutcome())
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["outcome"] != "cancelled" {
		t.Fatalf("bad wire form: %s", b)
	}
}

func TestSessionConfigOptionSelect(t *testing.T) {
	o := SessionConfigOption{
		ID:   "model",
		Name: "Model",
		Type: ConfigOptionSelect,
		Select: &SessionConfigSelect{
			CurrentValue: "gpt-5",
			Options: SessionConfigSelectOptions{
				Options: []SessionConfigSelectOption{{Value: "gpt-5", Name: "GPT-5"}},
			},
		},
	}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["type"] != "select" || m["id"] != "model" || m["currentValue"] != "gpt-5" {
		t.Fatalf("bad wire form: %s", b)
	}
	out := roundTrip(t, o)
	if out.Select == nil || out.Select.CurrentValue != "gpt-5" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestSessionConfigOptionGrouped(t *testing.T) {
	raw := `{"id":"m","name":"M","type":"select","currentValue":"a","options":[{"group":"g","name":"G","options":[{"value":"a","name":"A"}]}]}`
	var o SessionConfigOption
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		t.Fatal(err)
	}
	if o.Select == nil || len(o.Select.Options.Groups) != 1 || o.Select.Options.Groups[0].Group != "g" {
		t.Fatalf("grouped options lost: %+v", o)
	}
}

func TestSetSessionConfigOptionRequest(t *testing.T) {
	b, _ := json.Marshal(NewSetConfigBoolRequest("s", "c", true))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["type"] != "boolean" || m["value"] != true {
		t.Fatalf("bad wire form: %s", b)
	}
	var r SetSessionConfigOptionRequest
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if r.BoolValue == nil || !*r.BoolValue {
		t.Fatalf("bool value lost: %+v", r)
	}

	b, _ = json.Marshal(NewSetConfigValueRequest("s", "c", "v1"))
	m = map[string]any{}
	_ = json.Unmarshal(b, &m)
	if _, hasType := m["type"]; hasType {
		t.Fatalf("value_id form must not carry a type tag: %s", b)
	}
	r = SetSessionConfigOptionRequest{}
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if r.Value != "v1" || r.BoolValue != nil {
		t.Fatalf("value lost: %+v", r)
	}
}

func TestElicitationRequestForm(t *testing.T) {
	req := NewFormElicitation("s1", "need input", ElicitationSchema{
		Type: ElicitationSchemaObject,
		Properties: map[string]ElicitationPropertySchema{
			"name": {Type: ElicitationPropString, String: &StringPropertySchema{Title: "Name"}},
		},
		Required: []string{"name"},
	})
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["mode"] != "form" || m["sessionId"] != "s1" || m["message"] != "need input" {
		t.Fatalf("bad wire form: %s", b)
	}
	out := roundTrip(t, req)
	if out.Mode != ElicitationModeForm || out.RequestedSchema == nil || out.Session == nil {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestElicitationRequestURL(t *testing.T) {
	req := NewURLElicitation("s1", "open this", "e1", "https://example.com/auth")
	out := roundTrip(t, req)
	if out.Mode != ElicitationModeURL || out.URL != "https://example.com/auth" || out.ElicitationID != "e1" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestElicitationResponse(t *testing.T) {
	resp := CreateElicitationResponse{Action: ElicitationAccept, Content: map[string]any{"name": "x"}}
	out := roundTrip(t, resp)
	if out.Action != ElicitationAccept || out.Content["name"] != "x" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestToolCallContentTerminal(t *testing.T) {
	c := NewTerminalToolCallContent("term-1")
	b, _ := json.Marshal(c)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["type"] != "terminal" || m["terminalId"] != "term-1" {
		t.Fatalf("bad wire form: %s", b)
	}
	out := roundTrip(t, c)
	if out.Terminal == nil || out.Terminal.TerminalID != "term-1" {
		t.Fatalf("round trip failed: %+v", out)
	}
}
