// SPDX-License-Identifier: 0BSD

package acp

import (
	"github.com/Quad4-Software/acp-go/jsonrpc"
	"github.com/Quad4-Software/acp-go/transport"
)

// Aliases for the wire packages so the ACP API surface is reachable
// from this package alone. The underlying types live in the jsonrpc
// and transport packages.

// Conn is the JSON-RPC connection beneath AgentSide and ClientSide.
type Conn = jsonrpc.Conn

// Handler processes an incoming JSON-RPC request.
type Handler = jsonrpc.Handler

// NotificationHandler processes an incoming JSON-RPC notification.
type NotificationHandler = jsonrpc.NotificationHandler

// Transport is a bidirectional channel for JSON-RPC messages.
type Transport = transport.Transport

// RequestID is a JSON-RPC request identifier.
type RequestID = jsonrpc.RequestID

// NewConn returns a connection on transport t.
func NewConn(t Transport) *Conn { return jsonrpc.NewConn(t) }

// IntID returns a numeric request ID.
func IntID(n int64) RequestID { return jsonrpc.IntID(n) }

// StringID returns a string request ID.
func StringID(s string) RequestID { return jsonrpc.StringID(s) }
