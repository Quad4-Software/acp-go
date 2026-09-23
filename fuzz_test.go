// SPDX-License-Identifier: 0BSD

package acp

import (
	"encoding/json"
	"testing"
)

// FuzzUnionUnmarshal ensures arbitrary input never panics the union
// decoders and that surviving values re-marshal without error.
func FuzzUnionUnmarshal(f *testing.F) {
	seeds := []string{
		`{"type":"text","text":"hi"}`,
		`{"sessionUpdate":"plan","entries":[]}`,
		`{"outcome":"selected","optionId":"o"}`,
		`{"type":"http","name":"m","url":"u","headers":[]}`,
		`{"name":"m","command":"c","args":[],"env":[]}`,
		`{"type":"_unknown","x":1}`,
		`{"mode":"url","elicitationId":"e","url":"u","sessionId":"s"}`,
		`{"action":"accept","content":{"k":"v"}}`,
		`{}`, `null`, `[]`, `"s"`, `123`, `true`, `{`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		data := []byte(s)
		for _, v := range []any{
			&ContentBlock{},
			&SessionUpdate{},
			&ToolCallContent{},
			&RequestPermissionOutcome{},
			&MCPServer{},
			&AuthMethod{},
			&SessionConfigOption{},
			&ElicitationPropertySchema{},
			&CreateElicitationRequest{},
			&CreateElicitationResponse{},
		} {
			if err := json.Unmarshal(data, v); err != nil {
				continue
			}
			if _, err := json.Marshal(v); err != nil {
				t.Fatalf("remarshal %T of %q: %v", v, s, err)
			}
		}
	})
}
