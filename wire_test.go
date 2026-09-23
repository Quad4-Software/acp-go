// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"testing"
)

// assertWire marshals v and compares against the expected JSON
// object fields, then unmarshals back.
func assertWire(t *testing.T, v any, want map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("wire is not an object: %s", b)
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("%s missing key %q in %s", b, k, b)
		}
		gb, _ := json.Marshal(gv)
		wb, _ := json.Marshal(wv)
		if string(gb) != string(wb) {
			t.Fatalf("key %q: got %s want %s (full %s)", k, gb, wb, b)
		}
	}
	return b
}

func TestWireInitialize(t *testing.T) {
	assertWire(t, InitializeRequest{
		ProtocolVersion: ProtocolVersion1,
		ClientCapabilities: ClientCapabilities{
			FS:       FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
			Terminal: true,
		},
	}, map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": true, "writeTextFile": true},
			"terminal": true,
			"auth":     map[string]any{},
		},
	})
}

func TestWireSessionNotification(t *testing.T) {
	n := SessionNotification{
		SessionID: "s1",
		Update:    NewAgentMessageChunkUpdate(NewTextBlock("hi")),
	}
	assertWire(t, n, map[string]any{
		"sessionId": "s1",
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "hi"},
		},
	})
}

func TestWirePromptResponse(t *testing.T) {
	assertWire(t, PromptResponse{StopReason: StopCancelled},
		map[string]any{"stopReason": "cancelled"})
}

func TestWireRequestPermission(t *testing.T) {
	req := RequestPermissionRequest{
		SessionID: "s1",
		ToolCall:  ToolCallUpdate{ToolCallID: "tc1", Status: ToolCallPending},
		Options: []PermissionOption{
			{OptionID: "allow", Name: "Allow", Kind: PermissionAllowOnce},
		},
	}
	b := assertWire(t, req, map[string]any{
		"sessionId": "s1",
		"toolCall":  map[string]any{"toolCallId": "tc1", "status": "pending"},
	})
	var out RequestPermissionRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Options[0].Kind != PermissionAllowOnce {
		t.Fatalf("option lost: %+v", out.Options[0])
	}
}

func TestWireToolCallDiff(t *testing.T) {
	tc := ToolCall{
		ToolCallID: "tc1",
		Title:      "Edit file",
		Kind:       ToolKindEdit,
		Content: []ToolCallContent{
			NewDiffToolCallContent(Diff{Path: "/f.go", NewText: "new", OldText: ptr("old")}),
		},
		Locations: []ToolCallLocation{{Path: "/f.go", Line: ptr(int64(3))}},
	}
	b := assertWire(t, tc, map[string]any{
		"toolCallId": "tc1",
		"kind":       "edit",
	})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	content := m["content"].([]any)[0].(map[string]any)
	if content["type"] != "diff" || content["path"] != "/f.go" || content["newText"] != "new" {
		t.Fatalf("bad diff wire form: %s", b)
	}
}

func ptr[T any](v T) *T { return &v }

func TestAllSessionUpdateVariants(t *testing.T) {
	chunk := &ContentChunk{Content: NewTextBlock("x")}
	variants := []struct {
		tag string
		u   SessionUpdate
	}{
		{"user_message_chunk", SessionUpdate{UserMessageChunk: chunk}},
		{"agent_message_chunk", SessionUpdate{AgentMessageChunk: chunk}},
		{"agent_thought_chunk", SessionUpdate{AgentThoughtChunk: chunk}},
		{"tool_call", SessionUpdate{ToolCall: &ToolCall{ToolCallID: "t", Title: "t"}}},
		{"tool_call_update", SessionUpdate{ToolCallUpdate: &ToolCallUpdate{ToolCallID: "t"}}},
		{"plan", SessionUpdate{Plan: &Plan{Entries: []PlanEntry{}}}},
		{"available_commands_update", SessionUpdate{AvailableCommands: &AvailableCommandsUpdate{}}},
		{"current_mode_update", SessionUpdate{CurrentMode: &CurrentModeUpdate{CurrentModeID: "m"}}},
		{"config_option_update", SessionUpdate{ConfigOptions: &ConfigOptionUpdate{}}},
		{"session_info_update", SessionUpdate{SessionInfo: &SessionInfoUpdate{Title: "x"}}},
		{"usage_update", SessionUpdate{Usage: &UsageUpdate{Used: 1, Size: 10}}},
	}
	for _, tc := range variants {
		b, err := json.Marshal(tc.u)
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.tag, err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s: unmarshal wire: %v", tc.tag, err)
		}
		if m["sessionUpdate"] != tc.tag {
			t.Fatalf("got tag %v, want %q in %s", m["sessionUpdate"], tc.tag, b)
		}
		var back SessionUpdate
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.tag, err)
		}
		if back.Update != SessionUpdateType(tc.tag) {
			t.Fatalf("round trip type %q != %q", back.Update, tc.tag)
		}
	}
}

func TestAllContentBlockVariants(t *testing.T) {
	variants := []struct {
		tag string
		b   ContentBlock
	}{
		{"text", NewTextBlock("t")},
		{"image", NewImageBlock("AA==", "image/png")},
		{"audio", NewAudioBlock("AA==", "audio/wav")},
		{"resource_link", NewResourceLinkBlock("file:///f", "f")},
		{"resource", NewResourceBlock(ResourceContents{Text: &TextResourceContents{URI: "file:///f", Text: "x"}})},
	}
	for _, tc := range variants {
		b, err := json.Marshal(tc.b)
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.tag, err)
		}
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if m["type"] != tc.tag {
			t.Fatalf("got %v want %q in %s", m["type"], tc.tag, b)
		}
		var back ContentBlock
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.tag, err)
		}
		if string(back.Type) != tc.tag {
			t.Fatalf("round trip %q != %q", back.Type, tc.tag)
		}
	}
}

func TestResourceContentsBlob(t *testing.T) {
	raw := `{"uri":"file:///f","blob":"AA==","mimeType":"application/octet-stream"}`
	var r ResourceContents
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Blob == nil || r.Blob.Blob != "AA==" {
		t.Fatalf("blob lost: %+v", r)
	}
}

func TestUnionTypeInference(t *testing.T) {
	// Variants populated without Type must still marshal correctly.
	b, err := json.Marshal(ContentBlock{Text: &TextContent{Text: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["type"] != "text" {
		t.Fatalf("inference failed: %s", b)
	}

	b, err = json.Marshal(SessionUpdate{Plan: &Plan{}})
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(b, &m)
	if m["sessionUpdate"] != "plan" {
		t.Fatalf("inference failed: %s", b)
	}

	b, err = json.Marshal(ToolCallContent{Diff: &Diff{Path: "/f", NewText: "n"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(b, &m)
	if m["type"] != "diff" {
		t.Fatalf("inference failed: %s", b)
	}

	b, err = json.Marshal(RequestPermissionOutcome{Selected: &SelectedPermissionOutcome{OptionID: "o"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(b, &m)
	if m["outcome"] != "selected" {
		t.Fatalf("inference failed: %s", b)
	}

	b, err = json.Marshal(ElicitationPropertySchema{Boolean: &BooleanPropertySchema{Title: "ok"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(b, &m)
	if m["type"] != "boolean" {
		t.Fatalf("inference failed: %s", b)
	}
}

func TestSessionUpdateUnknownRaw(t *testing.T) {
	raw := `{"sessionUpdate":"_custom","data":{"a":1}}`
	var u SessionUpdate
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		t.Fatal(err)
	}
	if u.Raw == nil {
		t.Fatal("raw not preserved")
	}
	out, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != raw {
		t.Fatalf("got %s want %s", out, raw)
	}
}

func TestMCPServerAllVariants(t *testing.T) {
	for _, s := range []MCPServer{
		{Stdio: &MCPServerStdio{Name: "a", Command: "c"}},
		{Type: MCPTypeHTTP, HTTP: &MCPServerHTTP{Name: "a", URL: "u"}},
		{Type: MCPTypeSSE, SSE: &MCPServerSSE{Name: "a", URL: "u"}},
	} {
		out := roundTrip(t, s)
		switch {
		case s.Stdio != nil && out.Stdio == nil:
			t.Fatalf("stdio lost: %+v", out)
		case s.HTTP != nil && out.HTTP == nil:
			t.Fatalf("http lost: %+v", out)
		case s.SSE != nil && out.SSE == nil:
			t.Fatalf("sse lost: %+v", out)
		}
	}
}

func TestElicitationRequestScope(t *testing.T) {
	req := CreateElicitationRequest{
		Mode:    ElicitationModeForm,
		Message: "m",
		Request: &ElicitationRequestScope{RequestID: IntID(42)},
		RequestedSchema: &ElicitationSchema{
			Properties: map[string]ElicitationPropertySchema{
				"n": {Type: ElicitationPropInteger, Integer: &IntegerPropertySchema{Minimum: ptr(int64(0))}},
			},
		},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["requestId"] != float64(42) {
		t.Fatalf("request scope lost: %s", b)
	}
	out := roundTrip(t, req)
	if out.Request == nil || out.Request.RequestID.String() != "n:42" {
		t.Fatalf("round trip failed: %+v", out)
	}
}

func TestSetSessionModeWire(t *testing.T) {
	assertWire(t, SetSessionModeRequest{SessionID: "s", ModeID: "ask"},
		map[string]any{"sessionId": "s", "modeId": "ask"})
}
