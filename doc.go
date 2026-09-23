// SPDX-License-Identifier: 0BSD

// Package acp implements the Agent Client Protocol (ACP) version 1.
//
// ACP standardizes communication between code editors (clients) and
// coding agents, similar to how LSP standardizes language servers.
// Messages are JSON-RPC 2.0, transported over newline-delimited JSON
// on stdio or any other bidirectional byte stream.
//
// Agents implement the Agent interface and serve a Conn created with
// NewAgentSide. Clients implement the Client interface and serve a
// Conn created with NewClientSide. Both sides can issue requests and
// notifications to the peer through the typed methods on each side.
//
// The module is split into three packages: this one carries the
// protocol model and the typed agent and client APIs, the transport
// package carries byte stream adapters (stdio, subprocess, custom
// readers and writers), and the jsonrpc package carries the
// connection engine underneath both sides. Most users only need this
// package plus transport.
package acp
