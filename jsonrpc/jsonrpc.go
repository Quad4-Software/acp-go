// SPDX-License-Identifier: 0BSD

// Package jsonrpc implements the JSON-RPC 2.0 connection engine that
// carries ACP messages between agents and clients. It is transport
// agnostic: pair it with any transport.Transport.
package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JSONRPCVersion is the JSON-RPC version string used by ACP.
const JSONRPCVersion = "2.0"

// MethodCancelRequest is the protocol level notification used to cancel
// an in-flight request by ID.
const MethodCancelRequest = "$/cancel_request"

// RequestID is a JSON-RPC request identifier: an integer, a string, or
// null. The zero value is a null ID.
type RequestID struct {
	num int64
	str string
	is  idKind
}

type idKind uint8

const (
	idNull idKind = iota
	idNum
	idStr
)

// IntID returns a numeric request ID.
func IntID(n int64) RequestID { return RequestID{num: n, is: idNum} }

// StringID returns a string request ID.
func StringID(s string) RequestID { return RequestID{str: s, is: idStr} }

// IsNull reports whether the ID is null or absent.
func (r RequestID) IsNull() bool { return r.is == idNull }

// String returns a stable string form usable as a map key. Numeric and
// string IDs never collide.
func (r RequestID) String() string {
	switch r.is {
	case idNum:
		return fmt.Sprintf("n:%d", r.num)
	case idStr:
		return "s:" + r.str
	default:
		return "null"
	}
}

// MarshalJSON encodes the ID per the JSON-RPC specification.
func (r RequestID) MarshalJSON() ([]byte, error) {
	switch r.is {
	case idNum:
		return json.Marshal(r.num)
	case idStr:
		return json.Marshal(r.str)
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON decodes a string, integer, or null request ID.
func (r *RequestID) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	switch {
	case bytes.Equal(trimmed, []byte("null")):
		*r = RequestID{}
		return nil
	case len(trimmed) > 0 && trimmed[0] == '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		*r = StringID(s)
		return nil
	default:
		var n json.Number
		if err := json.Unmarshal(trimmed, &n); err != nil {
			return err
		}
		v, err := n.Int64()
		if err != nil {
			return fmt.Errorf("jsonrpc: request id must be an integer: %w", err)
		}
		*r = IntID(v)
		return nil
	}
}

// rawMessage is the envelope used to classify an incoming JSON-RPC
// message before full decoding.
type rawMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params"`
	Result  json.RawMessage  `json:"result"`
	Err     *Error           `json:"error"`
}

func (m rawMessage) isRequest() bool      { return m.Method != "" && m.ID != nil }
func (m rawMessage) isNotification() bool { return m.Method != "" && m.ID == nil }
func (m rawMessage) isResponse() bool     { return m.Method == "" && m.ID != nil }

// rpcRequest is the outbound request wire form.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcNotification is the outbound notification wire form.
type rpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is the outbound response wire form. Exactly one of Result
// or Err is meaningful.
type rpcResponse struct {
	ID     json.RawMessage
	Result json.RawMessage
	Err    *Error
}

// MarshalJSON emits exactly one of result or error as required by
// JSON-RPC 2.0.
func (r rpcResponse) MarshalJSON() ([]byte, error) {
	type response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Err     *Error          `json:"error,omitempty"`
	}
	if r.Err != nil {
		return json.Marshal(response{JSONRPC: JSONRPCVersion, ID: r.ID, Err: r.Err})
	}
	res := r.Result
	if len(res) == 0 {
		res = json.RawMessage("null")
	}
	return json.Marshal(response{JSONRPC: JSONRPCVersion, ID: r.ID, Result: res})
}

// CancelRequestNotification is the params of the $/cancel_request
// notification.
type CancelRequestNotification struct {
	RequestID RequestID      `json:"requestId"`
	Meta      map[string]any `json:"_meta,omitempty"`
}
