// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"testing"
)

func assertTag(t *testing.T, v any, key, want string) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("wire not object: %s", b)
	}
	if m[key] != want {
		t.Fatalf("tag %s = %v, want %q in %s", key, m[key], want, b)
	}
}

func TestUnimplementedAgent(t *testing.T) {
	a := UnimplementedAgent{}
	ctx := context.Background()
	calls := []func() error{
		func() error { _, e := a.Initialize(ctx, &InitializeRequest{}); return e },
		func() error { _, e := a.Authenticate(ctx, &AuthenticateRequest{}); return e },
		func() error { _, e := a.Logout(ctx, &LogoutRequest{}); return e },
		func() error { _, e := a.NewSession(ctx, &NewSessionRequest{}); return e },
		func() error { _, e := a.LoadSession(ctx, &LoadSessionRequest{}); return e },
		func() error { _, e := a.ResumeSession(ctx, &ResumeSessionRequest{}); return e },
		func() error { _, e := a.CloseSession(ctx, &CloseSessionRequest{}); return e },
		func() error { _, e := a.ListSessions(ctx, &ListSessionsRequest{}); return e },
		func() error { _, e := a.DeleteSession(ctx, &DeleteSessionRequest{}); return e },
		func() error { _, e := a.SetSessionMode(ctx, &SetSessionModeRequest{}); return e },
		func() error { _, e := a.SetSessionConfigOption(ctx, &SetSessionConfigOptionRequest{}); return e },
		func() error { _, e := a.Prompt(ctx, &PromptRequest{}); return e },
	}
	for i, c := range calls {
		if err := c(); !IsErrorCode(err, ErrCodeInternalError) {
			t.Fatalf("call %d: got %v", i, err)
		}
	}
	a.Cancel(ctx, &CancelNotification{})
}

func TestUnimplementedClient(t *testing.T) {
	c := UnimplementedClient{}
	ctx := context.Background()
	calls := []func() error{
		func() error { _, e := c.RequestPermission(ctx, &RequestPermissionRequest{}); return e },
		func() error { _, e := c.ReadTextFile(ctx, &ReadTextFileRequest{}); return e },
		func() error { _, e := c.WriteTextFile(ctx, &WriteTextFileRequest{}); return e },
		func() error { _, e := c.CreateTerminal(ctx, &CreateTerminalRequest{}); return e },
		func() error { _, e := c.TerminalOutput(ctx, &TerminalOutputRequest{}); return e },
		func() error { _, e := c.WaitForTerminalExit(ctx, &WaitForTerminalExitRequest{}); return e },
		func() error { _, e := c.KillTerminal(ctx, &KillTerminalRequest{}); return e },
		func() error { _, e := c.ReleaseTerminal(ctx, &ReleaseTerminalRequest{}); return e },
		func() error { _, e := c.CreateElicitation(ctx, &CreateElicitationRequest{}); return e },
	}
	for i, call := range calls {
		if err := call(); !IsErrorCode(err, ErrCodeInternalError) {
			t.Fatalf("call %d: got %v", i, err)
		}
	}
	c.SessionUpdate(ctx, &SessionNotification{})
	c.ElicitationComplete(ctx, &CompleteElicitationNotification{})
}

func TestDepsWrappers(t *testing.T) {
	ta, _ := pipePair()
	c := NewConn(ta)
	if c == nil || c.Done() == nil {
		t.Fatal("NewConn broken")
	}
	_ = c.Close()

	b, err := IntID(4).MarshalJSON()
	if err != nil || string(b) != "4" {
		t.Fatalf("IntID: %s %v", b, err)
	}
	b, err = StringID("x").MarshalJSON()
	if err != nil || string(b) != `"x"` {
		t.Fatalf("StringID: %s %v", b, err)
	}
}

func TestSessionUpdateVariants(t *testing.T) {
	chunk := NewTextBlock("x")
	tc := ToolCall{ToolCallID: "t1", Title: "T", Kind: ToolKindEdit, Status: ToolCallInProgress}
	tcu := ToolCallUpdate{ToolCallID: "t1", Title: "T2"}
	opts := []SessionConfigOption{{ID: "o", Name: "O", Type: ConfigOptionBoolean, Boolean: &SessionConfigBoolean{CurrentValue: true}}}

	cases := []struct {
		name string
		u    SessionUpdate
		tag  string
	}{
		{"user chunk", NewUserMessageChunkUpdate(chunk), "user_message_chunk"},
		{"agent chunk", NewAgentMessageChunkUpdate(chunk), "agent_message_chunk"},
		{"thought chunk", NewAgentThoughtChunkUpdate(chunk), "agent_thought_chunk"},
		{"tool call", NewToolCallUpdate(tc), "tool_call"},
		{"tool progress", NewToolCallProgressUpdate(tcu), "tool_call_update"},
		{"plan", NewPlanUpdate(Plan{Entries: []PlanEntry{{Content: "step", Priority: PriorityHigh, Status: PlanPending}}}), "plan"},
		{"commands", NewAvailableCommandsUpdate([]AvailableCommand{{Name: "run", Input: &AvailableCommandInput{Hint: "args"}}}), "available_commands_update"},
		{"mode", NewCurrentModeUpdate("code"), "current_mode_update"},
		{"config", NewConfigOptionsUpdate(opts), "config_option_update"},
		{"session info", SessionUpdate{SessionInfo: &SessionInfoUpdate{Title: "t", UpdatedAt: "now"}}, "session_info_update"},
		{"usage", NewUsageUpdate(UsageUpdate{Used: 10, Size: 100, Cost: &Cost{Amount: 0.01, Currency: "USD"}}), "usage_update"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertTag(t, c.u, "sessionUpdate", c.tag)
			out := roundTrip(t, c.u)
			if out.Update != SessionUpdateType(c.tag) {
				t.Fatalf("round trip update type %q, want %q", out.Update, c.tag)
			}
			if out.TypeOf() != SessionUpdateType(c.tag) {
				t.Fatalf("TypeOf %q", out.TypeOf())
			}
		})
	}
}

func TestSessionUpdateInferenceAndRaw(t *testing.T) {
	// Populated variant with unset discriminator still marshals the tag.
	u := SessionUpdate{Plan: &Plan{Entries: []PlanEntry{{Content: "p"}}}}
	assertTag(t, u, "sessionUpdate", "plan")

	var unk SessionUpdate
	if err := json.Unmarshal([]byte(`{"sessionUpdate":"_ext","data":1}`), &unk); err != nil {
		t.Fatal(err)
	}
	if unk.Raw == nil {
		t.Fatal("raw not preserved")
	}
	if err := json.Unmarshal([]byte(`{bad`), &unk); err == nil {
		t.Fatal("bad JSON accepted")
	}
	if err := json.Unmarshal([]byte(`{"sessionUpdate":42}`), &unk); err == nil {
		t.Fatal("non-string tag accepted")
	}
}

func TestContentBlockVariants(t *testing.T) {
	blob := BlobResourceContents{URI: "u", Blob: "AA==", MimeType: "m"}
	textRes := TextResourceContents{URI: "u", Text: "body", MimeType: "m"}
	cases := []struct {
		name string
		b    ContentBlock
		tag  string
	}{
		{"text", NewTextBlock("hi"), "text"},
		{"image", NewImageBlock("AA==", "image/png"), "image"},
		{"audio", NewAudioBlock("AA==", "audio/wav"), "audio"},
		{"link", NewResourceLinkBlock("file:///f", "f"), "resource_link"},
		{"blob res", NewResourceBlock(ResourceContents{Blob: &blob}), "resource"},
		{"text res", NewResourceBlock(ResourceContents{Text: &textRes}), "resource"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertTag(t, c.b, "type", c.tag)
			out := roundTrip(t, c.b)
			if out.Type != ContentBlockType(c.tag) || out.TypeOf() != ContentBlockType(c.tag) {
				t.Fatalf("round trip: %+v", out)
			}
		})
	}
}

func TestContentBlockErrors(t *testing.T) {
	var b ContentBlock
	for _, raw := range []string{
		`{bad`,
		`{"type":42}`,
		`{"type":"text","text":123}`,
	} {
		if err := json.Unmarshal([]byte(raw), &b); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestResourceContents(t *testing.T) {
	var tr ResourceContents
	if err := json.Unmarshal([]byte(`{"uri":"u","text":"x"}`), &tr); err != nil {
		t.Fatal(err)
	}
	if tr.Text == nil || tr.Text.Text != "x" {
		t.Fatalf("text variant lost: %+v", tr)
	}
	var br ResourceContents
	if err := json.Unmarshal([]byte(`{"uri":"u","blob":"AA=="}`), &br); err != nil {
		t.Fatal(err)
	}
	if br.Blob == nil || br.Blob.Blob != "AA==" {
		t.Fatalf("blob variant lost: %+v", br)
	}
	out := roundTrip(t, tr)
	if out.Text == nil {
		t.Fatal("round trip lost text")
	}
}

func TestToolCallContentVariants(t *testing.T) {
	cases := []struct {
		name string
		c    ToolCallContent
		tag  string
	}{
		{"content", NewContentToolCallContent(NewTextBlock("x")), "content"},
		{"diff", NewDiffToolCallContent(Diff{Path: "/f", NewText: "n", OldText: ptr("o")}), "diff"},
		{"terminal", NewTerminalToolCallContent("term-1"), "terminal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertTag(t, c.c, "type", c.tag)
			out := roundTrip(t, c.c)
			if out.TypeOf() != ToolCallContentType(c.tag) {
				t.Fatalf("TypeOf %q", out.TypeOf())
			}
		})
	}

	var c ToolCallContent
	if err := json.Unmarshal([]byte(`{"type":"_x","v":1}`), &c); err != nil || c.Raw == nil {
		t.Fatalf("unknown variant: %v %+v", err, c)
	}
	if err := json.Unmarshal([]byte(`{"type":"diff","path":123}`), &c); err == nil {
		t.Fatal("bad diff accepted")
	}
}

func TestPermissionOutcomeVariants(t *testing.T) {
	assertTag(t, NewSelectedOutcome("o1"), "outcome", "selected")
	assertTag(t, NewCancelledOutcome(), "outcome", "cancelled")
	out := roundTrip(t, NewCancelledOutcome())
	if out.Outcome != PermissionOutcomeCancelled || out.Selected != nil {
		t.Fatalf("cancelled lost: %+v", out)
	}
	var o RequestPermissionOutcome
	if err := json.Unmarshal([]byte(`{"outcome":"_x","v":1}`), &o); err != nil || o.Raw == nil {
		t.Fatalf("unknown outcome: %v %+v", err, o)
	}
	if err := json.Unmarshal([]byte(`{"outcome":42}`), &o); err == nil {
		t.Fatal("non-string outcome accepted")
	}
}

func TestMCPServerVariants(t *testing.T) {
	cases := []struct {
		name string
		s    MCPServer
		want MCPServerType
	}{
		{"http", MCPServer{HTTP: &MCPServerHTTP{Name: "n", URL: "u", Headers: []HTTPHeader{{Name: "h", Value: "v"}}}}, MCPTypeHTTP},
		{"sse", MCPServer{SSE: &MCPServerSSE{Name: "n", URL: "u", Headers: []HTTPHeader{}}}, MCPTypeSSE},
		{"stdio", MCPServer{Stdio: &MCPServerStdio{Name: "n", Command: "c", Args: []string{"a"}, Env: []EnvVariable{{Name: "E", Value: "v"}}}}, MCPTypeStdio},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := roundTrip(t, c.s)
			if out.Type != c.want {
				t.Fatalf("Type %q want %q", out.Type, c.want)
			}
		})
	}
	// Bare stdio object without a type tag decodes as stdio.
	var s MCPServer
	if err := json.Unmarshal([]byte(`{"name":"n","command":"c","args":[],"env":[]}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Type != MCPTypeStdio || s.Stdio == nil {
		t.Fatalf("default stdio: %+v", s)
	}
	if err := json.Unmarshal([]byte(`{"type":"_x","v":1}`), &s); err != nil || s.Raw == nil {
		t.Fatalf("unknown mcp: %v %+v", err, s)
	}
	if err := json.Unmarshal([]byte(`{bad`), &s); err == nil {
		t.Fatal("bad JSON accepted")
	}
}

func TestAuthMethodVariants(t *testing.T) {
	term := AuthMethod{Terminal: &AuthMethodTerminal{ID: "t", Name: "T", Args: []string{"a"}, Env: map[string]string{}}}
	assertTag(t, term, "type", "terminal")
	agent := AuthMethod{Agent: &AuthMethodAgent{ID: "a", Name: "A"}}
	b, _ := json.Marshal(agent)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["type"]; ok {
		t.Fatalf("agent method must not carry type: %s", b)
	}
	var a AuthMethod
	if err := json.Unmarshal([]byte(`{"type":"_x","v":1}`), &a); err != nil || a.Raw == nil {
		t.Fatalf("unknown auth method: %v %+v", err, a)
	}
	if err := json.Unmarshal([]byte(`{"type":"terminal","id":"t","name":"n","args":[],"env":{}}`), &a); err != nil || a.Terminal == nil {
		t.Fatalf("terminal method: %v %+v", err, a)
	}
}

func TestSessionConfigOptionVariants(t *testing.T) {
	sel := SessionConfigOption{Select: &SessionConfigSelect{
		CurrentValue: "v",
		Options:      SessionConfigSelectOptions{Options: []SessionConfigSelectOption{{Value: "v", Name: "V"}}},
	}}
	assertTag(t, sel, "type", "select")
	if out := roundTrip(t, sel); out.Select == nil {
		t.Fatal("select lost")
	}

	bo := SessionConfigOption{Boolean: &SessionConfigBoolean{CurrentValue: true}}
	assertTag(t, bo, "type", "boolean")
	if out := roundTrip(t, bo); out.Boolean == nil || !out.Boolean.CurrentValue {
		t.Fatalf("boolean lost: %+v", out)
	}

	var o SessionConfigOption
	if err := json.Unmarshal([]byte(`{"id":"i","name":"n","type":"_x","v":1}`), &o); err != nil || o.Raw == nil {
		t.Fatalf("unknown option: %v %+v", err, o)
	}
	if err := json.Unmarshal([]byte(`{"type":"boolean","currentValue":"x"}`), &o); err == nil {
		t.Fatal("bad boolean accepted")
	}

	// Options group form.
	var so SessionConfigSelectOptions
	if err := json.Unmarshal([]byte(`[{"group":"g","name":"G","options":[]}]`), &so); err != nil {
		t.Fatal(err)
	}
	if len(so.Groups) != 1 {
		t.Fatalf("groups lost: %+v", so)
	}
	flat := SessionConfigSelectOptions{Options: []SessionConfigSelectOption{{Value: "v", Name: "N"}}}
	b, _ := json.Marshal(flat)
	if string(b) != `[{"value":"v","name":"N"}]` {
		t.Fatalf("flat options wire: %s", b)
	}
	if err := so.UnmarshalJSON([]byte(`"x"`)); err == nil {
		t.Fatal("non-array options accepted")
	}
}

func TestSetConfigOptionRequestBranches(t *testing.T) {
	// Boolean type with missing value, unknown type preserved, meta.
	raw := `{"sessionId":"s","configId":"c","type":"boolean","_meta":{"k":1}}`
	var r SetSessionConfigOptionRequest
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Type != "boolean" || r.BoolValue != nil || r.Meta["k"] == nil {
		t.Fatalf("bad decode: %+v", r)
	}
	if err := json.Unmarshal([]byte(`{"type":"custom","value":"v"}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Type != "custom" || r.Value != "v" {
		t.Fatalf("custom type: %+v", r)
	}
	if err := json.Unmarshal([]byte(`{"sessionId":123}`), &r); err == nil {
		t.Fatal("bad sessionId accepted")
	}
	if err := json.Unmarshal([]byte(`{"value":[1]}`), &r); err == nil {
		t.Fatal("non-string value accepted")
	}
	// Marshal with Type=boolean and nil BoolValue emits false.
	b, _ := json.Marshal(SetSessionConfigOptionRequest{Type: "boolean"})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["value"] != false {
		t.Fatalf("bool default: %s", b)
	}
}

func TestElicitationPropertySchemas(t *testing.T) {
	i64 := func(v int64) *int64 { return &v }
	f64 := func(v float64) *float64 { return &v }
	bl := func(v bool) *bool { return &v }

	schemas := []struct {
		name  string
		p     ElicitationPropertySchema
		tag   string
		check func(ElicitationPropertySchema) bool
	}{
		{
			"string",
			ElicitationPropertySchema{String: &StringPropertySchema{Title: "s", MinLength: i64(1), Format: FormatEmail, Enum: []string{"a"}}, Type: ElicitationPropString},
			"string",
			func(p ElicitationPropertySchema) bool { return p.String != nil && *p.String.MinLength == 1 },
		},
		{
			"oneof",
			ElicitationPropertySchema{Type: ElicitationPropString, String: &StringPropertySchema{OneOf: []EnumOption{{Const: "c", Title: "C"}}}},
			"string",
			func(p ElicitationPropertySchema) bool { return p.String != nil && len(p.String.OneOf) == 1 },
		},
		{
			"number",
			ElicitationPropertySchema{Number: &NumberPropertySchema{Minimum: f64(0)}, Type: ElicitationPropNumber},
			"number",
			func(p ElicitationPropertySchema) bool { return p.Number != nil },
		},
		{
			"integer",
			ElicitationPropertySchema{Integer: &IntegerPropertySchema{Maximum: i64(9)}, Type: ElicitationPropInteger},
			"integer",
			func(p ElicitationPropertySchema) bool { return p.Integer != nil },
		},
		{
			"boolean",
			ElicitationPropertySchema{Boolean: &BooleanPropertySchema{Default: bl(true)}, Type: ElicitationPropBoolean},
			"boolean",
			func(p ElicitationPropertySchema) bool { return p.Boolean != nil },
		},
		{
			"multiselect enum",
			ElicitationPropertySchema{Type: ElicitationPropArray, MultiSelect: &MultiSelectPropertySchema{Items: MultiSelectItems{Enum: []string{"a", "b"}}}},
			"array",
			func(p ElicitationPropertySchema) bool {
				return p.MultiSelect != nil && len(p.MultiSelect.Items.Enum) == 2
			},
		},
		{
			"multiselect anyof",
			ElicitationPropertySchema{Type: ElicitationPropArray, MultiSelect: &MultiSelectPropertySchema{Items: MultiSelectItems{AnyOf: []EnumOption{{Const: "x", Title: "X"}}}}},
			"array",
			func(p ElicitationPropertySchema) bool {
				return p.MultiSelect != nil && len(p.MultiSelect.Items.AnyOf) == 1
			},
		},
	}
	for _, c := range schemas {
		t.Run(c.name, func(t *testing.T) {
			assertTag(t, c.p, "type", c.tag)
			out := roundTrip(t, c.p)
			if out.Type != ElicitationPropertyType(c.tag) {
				t.Fatalf("type %q", out.Type)
			}
			if !c.check(out) {
				t.Fatalf("variant lost: %+v", out)
			}
		})
	}

	// Inference without explicit type.
	p := ElicitationPropertySchema{Boolean: &BooleanPropertySchema{}}
	assertTag(t, p, "type", "boolean")

	var items MultiSelectItems
	if err := items.UnmarshalJSON([]byte(`{"enum":["a","b"]}`)); err != nil || len(items.Enum) != 2 {
		t.Fatalf("enum items: %v %+v", err, items)
	}
	if err := items.UnmarshalJSON([]byte(`{"anyOf":[{"const":"c","title":"C"}]}`)); err != nil || len(items.AnyOf) != 1 {
		t.Fatalf("anyof items: %v %+v", err, items)
	}
	if err := items.UnmarshalJSON([]byte(`{"odd":1}`)); err != nil || items.Raw == nil {
		t.Fatalf("raw items: %v %+v", err, items)
	}

	var p2 ElicitationPropertySchema
	if err := json.Unmarshal([]byte(`{"type":"_x","v":1}`), &p2); err != nil || p2.Raw == nil {
		t.Fatalf("unknown prop schema: %v %+v", err, p2)
	}
	if err := json.Unmarshal([]byte(`{"type":"number","minimum":"x"}`), &p2); err == nil {
		t.Fatal("bad number schema accepted")
	}
}

func TestElicitationRequestResponseEdges(t *testing.T) {
	var r CreateElicitationRequest
	if err := json.Unmarshal([]byte(`{"mode":"url","elicitationId":"e","url":"u","sessionId":"s","requestId":7}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Mode != ElicitationModeURL || r.Request == nil || r.Request.RequestID != IntID(7) {
		t.Fatalf("url scope: %+v", r)
	}
	if err := json.Unmarshal([]byte(`{"mode":"form","message":"m","sessionId":"s","requestedSchema":{"type":"object","properties":{}}}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Session == nil || r.Session.SessionID != "s" {
		t.Fatalf("form scope: %+v", r)
	}
	if err := json.Unmarshal([]byte(`{"mode":"_x","v":1}`), &r); err != nil || r.Raw == nil {
		t.Fatalf("unknown mode not preserved: %v %+v", err, r)
	}
	if err := json.Unmarshal([]byte(`{bad`), &r); err == nil {
		t.Fatal("bad JSON accepted")
	}

	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{"action":"cancel"}`), &resp); err != nil || resp.Action != ElicitationCancel {
		t.Fatalf("cancel: %v %+v", err, resp)
	}
	if err := json.Unmarshal([]byte(`{"action":"accept","content":{"a":1}}`), &resp); err != nil || resp.Content["a"] == nil {
		t.Fatalf("accept content: %v %+v", err, resp)
	}
	if err := json.Unmarshal([]byte(`{"action":"_x","v":1}`), &resp); err != nil || resp.Raw == nil {
		t.Fatalf("unknown action not preserved: %v %+v", err, resp)
	}
	if err := json.Unmarshal([]byte(`{"action":"accept","content":"x"}`), &resp); err == nil {
		t.Fatal("non-object content accepted")
	}
}

func TestStopReasonsAndBasics(t *testing.T) {
	var p PromptResponse
	if err := json.Unmarshal([]byte(`{"stopReason":"refusal"}`), &p); err != nil || p.StopReason != StopRefusal {
		t.Fatalf("stop reason: %v %+v", err, p)
	}
	var n CancelNotification
	if err := json.Unmarshal([]byte(`{"sessionId":"s"}`), &n); err != nil || n.SessionID != "s" {
		t.Fatalf("cancel: %v %+v", err, n)
	}
}
