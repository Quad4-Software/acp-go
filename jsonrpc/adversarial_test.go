// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// adversarialServer returns a running conn and the raw peer transport
// the test uses to inject malformed frames and read raw responses.
func adversarialServer(t *testing.T) (*Conn, interface {
	Write(context.Context, json.RawMessage) error
	Read(context.Context) (json.RawMessage, error)
},
) {
	t.Helper()
	ta, tb := pipePair()
	srv := NewConn(ta)
	srv.Handle("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"pong": true}, nil
	})
	go func() { _ = srv.Run(context.Background()) }()
	t.Cleanup(func() { _ = srv.Close() })
	return srv, tb
}

func readResponse(t *testing.T, tb interface {
	Read(context.Context) (json.RawMessage, error)
}, timeout time.Duration,
) (map[string]any, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	raw, err := tb.Read(ctx)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("response is not JSON: %w", err)
	}
	return m, nil
}

func errorCode(t *testing.T, m map[string]any) float64 {
	t.Helper()
	e, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error response, got %v", m)
	}
	code, ok := e["code"].(float64)
	if !ok {
		t.Fatalf("error has no numeric code: %v", e)
	}
	return code
}

func TestAdversarialFrames(t *testing.T) {
	cases := []struct {
		name     string
		frame    string
		wantCode float64 // 0 means a successful result
		wantID   string  // raw expected id, "" means skip check
	}{
		{"unknown method", `{"jsonrpc":"2.0","id":1,"method":"_bogus"}`, -32601, "1"},
		{"fractional id", `{"jsonrpc":"2.0","id":1.5,"method":"ping"}`, -32600, "1.5"},
		{"bool id", `{"jsonrpc":"2.0","id":true,"method":"ping"}`, -32600, "true"},
		{"object id", `{"jsonrpc":"2.0","id":{"x":1},"method":"ping"}`, -32600, `{"x":1}`},
		{"array id", `{"jsonrpc":"2.0","id":[1],"method":"ping"}`, -32600, "[1]"},
		{"wrong version", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, -32600, "1"},
		{"missing version", `{"id":1,"method":"ping"}`, -32600, "1"},
		{"batch array", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, -32700, "null"},
		{"truncated json", `{"jsonrpc":"2.0","id":1,"meth`, -32700, "null"},
		{"not an object", `42`, -32700, "null"},
		{"negative id", `{"jsonrpc":"2.0","id":-3,"method":"ping"}`, 0, "-3"},
		{"string id", `{"jsonrpc":"2.0","id":"s1","method":"ping"}`, 0, `"s1"`},
		{"huge params", fmt.Sprintf(`{"jsonrpc":"2.0","id":9,"method":"ping","params":{"blob":"%s"}}`, strings.Repeat("x", 1<<20)), 0, "9"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, tb := adversarialServer(t)
			if err := tb.Write(context.Background(), json.RawMessage(c.frame)); err != nil {
				t.Fatal(err)
			}
			m, err := readResponse(t, tb, 2*time.Second)
			if err != nil {
				t.Fatalf("no response: %v", err)
			}
			if c.wantCode != 0 {
				if code := errorCode(t, m); code != c.wantCode {
					t.Fatalf("code %v, want %v (msg %v)", code, c.wantCode, m)
				}
			} else if _, isErr := m["error"]; isErr {
				t.Fatalf("expected result, got %v", m)
			}
			if c.wantID != "" {
				got, _ := json.Marshal(m["id"])
				if string(got) != c.wantID {
					t.Fatalf("id %s, want %s", got, c.wantID)
				}
			}
		})
	}
}

func TestAdversarialDropped(t *testing.T) {
	cases := []struct {
		name  string
		frame string
	}{
		{"no method no id", `{"jsonrpc":"2.0","params":{}}`},
		{"null id request", `{"jsonrpc":"2.0","id":null,"method":"ping"}`},
		{"notification unknown method", `{"jsonrpc":"2.0","method":"_bogus"}`},
		{"notification wrong version", `{"jsonrpc":"1.0","method":"ping"}`},
		{"cancel malformed params", `{"jsonrpc":"2.0","method":"$/cancel_request","params":{"requestId":true}}`},
		{"cancel missing params", `{"jsonrpc":"2.0","method":"$/cancel_request"}`},
		{"response unknown id", `{"jsonrpc":"2.0","id":77,"result":{}}`},
		{"response both result and error", `{"jsonrpc":"2.0","id":77,"result":{},"error":{"code":-32603,"message":"x"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, tb := adversarialServer(t)
			if err := tb.Write(context.Background(), json.RawMessage(c.frame)); err != nil {
				t.Fatal(err)
			}
			// The server must not answer any of these. A short read
			// timeout proves silence; the conn must stay alive, which
			// the follow-up ping verifies.
			if m, err := readResponse(t, tb, 300*time.Millisecond); err == nil {
				t.Fatalf("unexpected response to dropped frame: %v", m)
			}
			if err := tb.Write(context.Background(),
				json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)); err != nil {
				t.Fatal(err)
			}
			m, err := readResponse(t, tb, 2*time.Second)
			if err != nil {
				t.Fatalf("conn died after adversarial frame: %v", err)
			}
			if res, ok := m["result"].(map[string]any); !ok || res["pong"] != true {
				t.Fatalf("bad ping response: %v", m)
			}
		})
	}
}

func TestAdversarialResponseBothFields(t *testing.T) {
	// A response carrying both result and error must surface the error.
	ta, tb := pipePair()
	srv := NewConn(ta)
	go func() { _ = srv.Run(context.Background()) }()
	t.Cleanup(func() { _ = srv.Close() })

	done := make(chan error, 1)
	go func() {
		done <- srv.Call(context.Background(), "ping", nil, nil)
	}()

	raw, err := tb.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	reply := fmt.Sprintf(`{"jsonrpc":"2.0","id":%v,"result":{"pong":true},"error":{"code":-32000,"message":"auth"}}`, req["id"])
	if err := tb.Write(context.Background(), json.RawMessage(reply)); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !IsErrorCode(err, ErrCodeAuthRequired) {
			t.Fatalf("expected auth error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("call did not return")
	}
}

func TestAdversarialDuplicateRequestID(t *testing.T) {
	// Two requests with the same id must not deadlock or panic. The
	// peer gets both responses; ordering is unspecified.
	_, tb := adversarialServer(t)
	frame := `{"jsonrpc":"2.0","id":42,"method":"ping"}`
	for i := 0; i < 2; i++ {
		if err := tb.Write(context.Background(), json.RawMessage(frame)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		m, err := readResponse(t, tb, 2*time.Second)
		if err != nil {
			t.Fatalf("missing response %d: %v", i, err)
		}
		if res, ok := m["result"].(map[string]any); !ok || res["pong"] != true {
			t.Fatalf("bad response %d: %v", i, m)
		}
	}
}

func TestAdversarialNonUTF8(t *testing.T) {
	_, tb := adversarialServer(t)
	if err := tb.Write(context.Background(), []byte{0xff, 0xfe, 0x00}); err != nil {
		t.Fatal(err)
	}
	m, err := readResponse(t, tb, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code := errorCode(t, m); code != -32700 {
		t.Fatalf("code %v, want -32700", code)
	}
}

func TestAdversarialUTF8InMethod(t *testing.T) {
	// Go's decoder replaces invalid UTF-8 inside strings with U+FFFD
	// rather than failing, so the frame parses and the mangled method
	// name is simply unknown.
	_, tb := adversarialServer(t)
	frame := append([]byte(`{"jsonrpc":"2.0","id":1,"method":"`), 0xff, 0xfe, '"', '}')
	if err := tb.Write(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	m, err := readResponse(t, tb, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code := errorCode(t, m); code != -32601 {
		t.Fatalf("code %v, want -32601", code)
	}
}
