// SPDX-License-Identifier: 0BSD

package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrorCode is a JSON-RPC 2.0 error code. Standard codes use the range
// reserved by the JSON-RPC specification. ACP specific codes use the
// implementation defined server error range.
type ErrorCode int32

// Standard JSON-RPC 2.0 and ACP error codes.
const (
	// ErrCodeParseError means invalid JSON was received by the peer.
	ErrCodeParseError ErrorCode = -32700
	// ErrCodeInvalidRequest means the message is not a valid request object.
	ErrCodeInvalidRequest ErrorCode = -32600
	// ErrCodeMethodNotFound means the method does not exist or is unavailable.
	ErrCodeMethodNotFound ErrorCode = -32601
	// ErrCodeInvalidParams means the method parameters are invalid.
	ErrCodeInvalidParams ErrorCode = -32602
	// ErrCodeInternalError means an internal JSON-RPC error occurred.
	ErrCodeInternalError ErrorCode = -32603
	// ErrCodeRequestCancelled means execution was aborted by a cancellation
	// request from the caller or by shutdown.
	ErrCodeRequestCancelled ErrorCode = -32800
	// ErrCodeAuthRequired means authentication is required first.
	ErrCodeAuthRequired ErrorCode = -32000
	// ErrCodeResourceNotFound means a resource such as a file was not found.
	ErrCodeResourceNotFound ErrorCode = -32002
)

// Error is a JSON-RPC 2.0 error object. Handlers may return *Error values
// to control the code and data sent to the peer. Any other error is sent
// as ErrCodeInternalError with the error text as the message.
type Error struct {
	Code    ErrorCode       `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error returns the error message with its code.
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc: %s (code %d)", e.Message, e.Code)
}

// NewError returns an *Error with the given code and message.
func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewErrorf returns an *Error with a formatted message.
func NewErrorf(code ErrorCode, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WithData attaches structured data to the error.
func (e *Error) WithData(v any) *Error {
	b, err := json.Marshal(v)
	if err != nil {
		return e
	}
	e.Data = b
	return e
}

// MethodNotFound returns an error for an unrecognized method.
func MethodNotFound(method string) *Error {
	return NewErrorf(ErrCodeMethodNotFound, "method not found: %s", method)
}

// InvalidParams returns an error for malformed request parameters.
func InvalidParams(msg string) *Error {
	return NewError(ErrCodeInvalidParams, msg)
}

// ErrorCodeOf extracts the JSON-RPC error code from err. It returns
// the code of the first *Error found in the chain, or
// ErrCodeInternalError if err is a non-RPC error, or 0 if err is nil.
func ErrorCodeOf(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	if err != nil {
		return ErrCodeInternalError
	}
	return 0
}

// IsErrorCode reports whether err carries the given JSON-RPC error code.
func IsErrorCode(err error, code ErrorCode) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}
