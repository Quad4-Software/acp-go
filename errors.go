// SPDX-License-Identifier: 0BSD

package acp

import "github.com/Quad4-Software/acp-go/jsonrpc"

// Error is a JSON-RPC 2.0 error object. It aliases jsonrpc.Error so
// callers can work with errors without importing the wire package.
type Error = jsonrpc.Error

// ErrorCode is a JSON-RPC 2.0 error code.
type ErrorCode = jsonrpc.ErrorCode

// Standard JSON-RPC 2.0 and ACP error codes.
const (
	// ErrCodeParseError means invalid JSON was received by the peer.
	ErrCodeParseError = jsonrpc.ErrCodeParseError
	// ErrCodeInvalidRequest means the message is not a valid request object.
	ErrCodeInvalidRequest = jsonrpc.ErrCodeInvalidRequest
	// ErrCodeMethodNotFound means the method does not exist or is unavailable.
	ErrCodeMethodNotFound = jsonrpc.ErrCodeMethodNotFound
	// ErrCodeInvalidParams means the method parameters are invalid.
	ErrCodeInvalidParams = jsonrpc.ErrCodeInvalidParams
	// ErrCodeInternalError means an internal JSON-RPC error occurred.
	ErrCodeInternalError = jsonrpc.ErrCodeInternalError
	// ErrCodeRequestCancelled means execution was aborted by a
	// cancellation request from the caller or by shutdown.
	ErrCodeRequestCancelled = jsonrpc.ErrCodeRequestCancelled
	// ErrCodeAuthRequired means authentication is required first.
	ErrCodeAuthRequired = jsonrpc.ErrCodeAuthRequired
	// ErrCodeResourceNotFound means a resource such as a file was not found.
	ErrCodeResourceNotFound = jsonrpc.ErrCodeResourceNotFound
)

// NewError returns an *Error with the given code and message.
func NewError(code ErrorCode, message string) *Error {
	return jsonrpc.NewError(code, message)
}

// NewErrorf returns an *Error with a formatted message.
func NewErrorf(code ErrorCode, format string, args ...any) *Error {
	return jsonrpc.NewErrorf(code, format, args...)
}

// AuthRequired returns the error an agent sends when a request needs a
// prior successful authenticate call.
func AuthRequired() *Error {
	return NewError(ErrCodeAuthRequired, "authentication required")
}

// ResourceNotFound returns an error reporting a missing resource such
// as a session or file.
func ResourceNotFound(what string) *Error {
	return NewErrorf(ErrCodeResourceNotFound, "resource not found: %s", what)
}

// MethodNotFound returns an error for an unrecognized method.
func MethodNotFound(method string) *Error {
	return jsonrpc.MethodNotFound(method)
}

// InvalidParams returns an error for malformed request parameters.
func InvalidParams(msg string) *Error {
	return jsonrpc.InvalidParams(msg)
}

// ErrorCodeOf extracts the JSON-RPC error code from err. It returns
// the code of the first *Error found in the chain, or
// ErrCodeInternalError if err is a non-RPC error, or 0 if err is nil.
func ErrorCodeOf(err error) ErrorCode {
	return jsonrpc.ErrorCodeOf(err)
}

// IsErrorCode reports whether err carries the given JSON-RPC error code.
func IsErrorCode(err error, code ErrorCode) bool {
	return jsonrpc.IsErrorCode(err, code)
}
