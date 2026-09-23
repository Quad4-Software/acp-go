// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSideAccessors(t *testing.T) {
	ta, tb := pipePair()
	as := NewAgentSide(ta, UnimplementedAgent{})
	cs := NewClientSide(tb, UnimplementedClient{})
	if as.Conn() == nil || as.Done() == nil || cs.Conn() == nil || cs.Done() == nil {
		t.Fatal("accessors broken")
	}
	_ = as.Close()
	_ = cs.Close()
}

func TestUnionMarshalErrors(t *testing.T) {
	bad := make(chan int)
	cases := []any{
		ContentBlock{Type: ContentBlockType("_x")},
		ToolCallContent{Type: ToolCallContentType("_x")},
		SessionUpdate{Update: SessionUpdateType("_x")},
		RequestPermissionOutcome{Outcome: RequestPermissionOutcomeType("_x")},
		MCPServer{Type: MCPServerType("_x")},
		SessionConfigOption{Type: SessionConfigOptionType("_x")},
		ElicitationPropertySchema{Type: ElicitationPropertyType("_x")},
		CreateElicitationRequest{Mode: ElicitationMode("_x")},
		CreateElicitationResponse{Action: ElicitationAction("_x")},
	}
	for i, v := range cases {
		if _, err := json.Marshal(v); err == nil {
			t.Fatalf("case %d: unknown type marshaled", i)
		}
	}
	// A variant value that cannot marshal must surface the error.
	b := ContentBlock{Type: ContentBlockResource, Resource: &EmbeddedResource{
		Resource: ResourceContents{Text: &TextResourceContents{URI: "u", Text: "x", Meta: Meta{"bad": bad}}},
	}}
	if _, err := json.Marshal(b); err == nil {
		t.Fatal("unmarshalable variant marshaled")
	}
	_ = bad
}

func TestUnionRawMarshal(t *testing.T) {
	raws := []any{
		ContentBlock{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		ToolCallContent{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		SessionUpdate{Raw: json.RawMessage(`{"sessionUpdate":"_e","v":1}`)},
		RequestPermissionOutcome{Raw: json.RawMessage(`{"outcome":"_e","v":1}`)},
		MCPServer{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		AuthMethod{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		SessionConfigOption{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		ElicitationPropertySchema{Raw: json.RawMessage(`{"type":"_e","v":1}`)},
		CreateElicitationRequest{Raw: json.RawMessage(`{"mode":"_e","v":1}`)},
		CreateElicitationResponse{Raw: json.RawMessage(`{"action":"_e","v":1}`)},
	}
	for i, v := range raws {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("case %d marshal raw: %v", i, err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil || m["v"] != float64(1) {
			t.Fatalf("case %d raw lost: %s", i, b)
		}
	}
}

func TestContentBlockTypeOf(t *testing.T) {
	cases := []struct {
		b    ContentBlock
		want ContentBlockType
	}{
		{ContentBlock{Type: ContentBlockAudio}, ContentBlockAudio},
		{ContentBlock{Image: &ImageContent{}}, ContentBlockImage},
		{ContentBlock{Audio: &AudioContent{}}, ContentBlockAudio},
		{ContentBlock{ResourceLink: &ResourceLink{}}, ContentBlockResourceLink},
		{ContentBlock{Resource: &EmbeddedResource{}}, ContentBlockResource},
		{ContentBlock{}, ""},
	}
	for i, c := range cases {
		if got := c.b.TypeOf(); got != c.want {
			t.Fatalf("case %d: TypeOf %q want %q", i, got, c.want)
		}
	}
}

func TestToolCallContentTypeOf(t *testing.T) {
	cases := []struct {
		c    ToolCallContent
		want ToolCallContentType
	}{
		{ToolCallContent{Content: &Content{}}, ToolCallContentBlock},
		{ToolCallContent{Diff: &Diff{}}, ToolCallContentDiff},
		{ToolCallContent{Terminal: &Terminal{}}, ToolCallContentTerminal},
		{ToolCallContent{}, ""},
	}
	for i, c := range cases {
		if got := c.c.TypeOf(); got != c.want {
			t.Fatalf("case %d: TypeOf %q want %q", i, got, c.want)
		}
	}
}

func TestSessionUpdateTypeOf(t *testing.T) {
	cases := []struct {
		u    SessionUpdate
		want SessionUpdateType
	}{
		{SessionUpdate{UserMessageChunk: &ContentChunk{}}, UpdateUserMessageChunk},
		{SessionUpdate{AgentMessageChunk: &ContentChunk{}}, UpdateAgentMessageChunk},
		{SessionUpdate{AgentThoughtChunk: &ContentChunk{}}, UpdateAgentThoughtChunk},
		{SessionUpdate{ToolCall: &ToolCall{}}, UpdateToolCall},
		{SessionUpdate{ToolCallUpdate: &ToolCallUpdate{}}, UpdateToolCallUpdate},
		{SessionUpdate{Plan: &Plan{}}, UpdatePlan},
		{SessionUpdate{AvailableCommands: &AvailableCommandsUpdate{}}, UpdateAvailableCommands},
		{SessionUpdate{CurrentMode: &CurrentModeUpdate{}}, UpdateCurrentMode},
		{SessionUpdate{ConfigOptions: &ConfigOptionUpdate{}}, UpdateConfigOptions},
		{SessionUpdate{SessionInfo: &SessionInfoUpdate{}}, UpdateSessionInfo},
		{SessionUpdate{Usage: &UsageUpdate{}}, UpdateUsage},
		{SessionUpdate{}, ""},
	}
	for i, c := range cases {
		if got := c.u.TypeOf(); got != c.want {
			t.Fatalf("case %d: TypeOf %q want %q", i, got, c.want)
		}
	}
}

func TestElicitationMarshalBranches(t *testing.T) {
	// Form marshal.
	f := NewFormElicitation("s", "m", ElicitationSchema{Type: ElicitationSchemaObject})
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["mode"] != "form" || m["requestedSchema"] == nil {
		t.Fatalf("form marshal: %s", b)
	}
	// URL marshal.
	u := NewURLElicitation("s", "m", "e", "http://x")
	b, _ = json.Marshal(u)
	_ = json.Unmarshal(b, &m)
	if m["mode"] != "url" || m["elicitationId"] != "e" || m["url"] != "http://x" {
		t.Fatalf("url marshal: %s", b)
	}
	// Request-scoped marshal emits requestId.
	rs := CreateElicitationRequest{Mode: ElicitationModeURL, Request: &ElicitationRequestScope{RequestID: IntID(3)}, URL: "u", ElicitationID: "e"}
	b, _ = json.Marshal(rs)
	_ = json.Unmarshal(b, &m)
	if m["requestId"] != float64(3) {
		t.Fatalf("request scope: %s", b)
	}
	// Response marshal for each action.
	for _, a := range []ElicitationAction{ElicitationAccept, ElicitationDecline, ElicitationCancel} {
		r := CreateElicitationResponse{Action: a}
		if a == ElicitationAccept {
			r.Content = map[string]any{"k": "v"}
		}
		assertTag(t, r, "action", string(a))
	}
}

func TestSelectOptionsMarshal(t *testing.T) {
	g := SessionConfigSelectOptions{Groups: []SessionConfigSelectGroup{{Group: "g", Name: "G"}}}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `[{"group":"g","name":"G","options":null}]` {
		t.Fatalf("groups wire: %s", b)
	}
	empty := SessionConfigSelectOptions{}
	if _, err := json.Marshal(empty); err != nil {
		t.Fatalf("empty options: %v", err)
	}
}

func TestAgentCancelNotificationRoundTrip(t *testing.T) {
	ta, tb := pipePair()
	got := make(chan SessionID, 1)
	agent := &cancelSpy{ch: got}
	as := NewAgentSide(ta, agent)
	cs := NewClientSide(tb, UnimplementedClient{})
	go func() { _ = as.Run(context.Background()) }()
	go func() { _ = cs.Run(context.Background()) }()
	t.Cleanup(func() { _ = as.Close(); _ = cs.Close() })

	if err := cs.Cancel(context.Background(), "sid-9"); err != nil {
		t.Fatal(err)
	}
	select {
	case sid := <-got:
		if sid != "sid-9" {
			t.Fatalf("cancel sid %q", sid)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel not delivered")
	}
}

type cancelSpy struct {
	UnimplementedAgent
	ch chan SessionID
}

func (a *cancelSpy) Cancel(_ context.Context, n *CancelNotification) { a.ch <- n.SessionID }
