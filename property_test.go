// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"strconv"
	"testing"
	"testing/quick"
)

// TestPropertyTextBlockRoundTrip holds for arbitrary strings including
// control characters, unicode, and NUL bytes: marshal then unmarshal
// preserves text and discriminator.
func TestPropertyTextBlockRoundTrip(t *testing.T) {
	err := quick.Check(func(s string) bool {
		b := NewTextBlock(s)
		data, err := json.Marshal(b)
		if err != nil {
			return false
		}
		var out ContentBlock
		if err := json.Unmarshal(data, &out); err != nil {
			return false
		}
		return out.Text != nil && out.Text.Text == s && out.TypeOf() == ContentBlockText
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// TestPropertyRequestIDRoundTrip: numeric and string IDs survive a
// marshal/unmarshal cycle with the same canonical key.
func TestPropertyRequestIDRoundTrip(t *testing.T) {
	err := quick.Check(func(n int64) bool {
		data, err := json.Marshal(IntID(n))
		if err != nil {
			return false
		}
		var out RequestID
		if err := json.Unmarshal(data, &out); err != nil {
			return false
		}
		return out.String() == IntID(n).String()
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = quick.Check(func(s string) bool {
		data, err := json.Marshal(StringID(s))
		if err != nil {
			return false
		}
		var out RequestID
		if err := json.Unmarshal(data, &out); err != nil {
			return false
		}
		return out.String() == StringID(s).String()
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// TestPropertyMetaRoundTrip: arbitrary _meta maps survive on both
// directions of the wire.
func TestPropertyMetaRoundTrip(t *testing.T) {
	err := quick.Check(func(m map[string]string) bool {
		meta := Meta{}
		for k, v := range m {
			meta[k] = v
		}
		req := &NewSessionRequest{CWD: ".", Meta: meta}
		data, err := json.Marshal(req)
		if err != nil {
			return false
		}
		var out NewSessionRequest
		if err := json.Unmarshal(data, &out); err != nil {
			return false
		}
		if len(m) == 0 {
			return true // nil vs empty map is fine
		}
		for k, v := range m {
			got, ok := out.Meta[k].(string)
			if !ok || got != v {
				return false
			}
		}
		return true
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// TestPropertyPromptRoundTrip: a prompt of arbitrary text blocks
// round-trips intact through the request envelope.
func TestPropertyPromptRoundTrip(t *testing.T) {
	err := quick.Check(func(sid, text string) bool {
		req := &PromptRequest{
			SessionID: SessionID(sid),
			Prompt:    []ContentBlock{NewTextBlock(text)},
		}
		data, err := json.Marshal(req)
		if err != nil {
			return false
		}
		var out PromptRequest
		if err := json.Unmarshal(data, &out); err != nil {
			return false
		}
		return string(out.SessionID) == sid &&
			len(out.Prompt) == 1 &&
			out.Prompt[0].Text != nil &&
			out.Prompt[0].Text.Text == text
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// TestPropertySessionUpdateUnionNeverPanics: unmarshaling arbitrary
// JSON-shaped inputs into every tagged union must never panic. Correct
// or rejected are both fine; crashing is not.
func TestPropertySessionUpdateUnionNeverPanics(t *testing.T) {
	err := quick.Check(func(tag, payload string) bool {
		raw := []byte(`{"sessionUpdate":` + jsonString(tag) + `,"data":` + jsonString(payload) + `}`)
		var u SessionUpdate
		_ = json.Unmarshal(raw, &u)
		var b ContentBlock
		_ = json.Unmarshal(raw, &b)
		var m MCPServer
		_ = json.Unmarshal(raw, &m)
		return true
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestPropertyUnionRawPreserved: an unknown discriminator always lands
// in Raw and re-marshals to the identical bytes.
func TestPropertyUnionRawPreserved(t *testing.T) {
	err := quick.Check(func(tag string, n int64) bool {
		raw := []byte(`{"type":` + jsonString("_"+tag) + `,"n":` + jsonString(
			strconv.FormatInt(n, 10)) + `}`)
		var b ContentBlock
		if err := json.Unmarshal(raw, &b); err != nil {
			return false
		}
		if b.Raw == nil {
			return false
		}
		out, err := json.Marshal(b)
		if err != nil {
			return false
		}
		return string(out) == string(raw)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
