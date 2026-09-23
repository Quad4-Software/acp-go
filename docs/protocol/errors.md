# Errors

Protocol failures travel as JSON-RPC error objects:

```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32002,"message":"resource not found: /tmp/x"}}
```

`acp.Error` (an alias of `jsonrpc.Error`) models the object:

```go
type Error struct {
    Code    ErrorCode       `json:"code"`
    Message string          `json:"message"`
    Data    json.RawMessage `json:"data,omitempty"`
}
```

## Codes

| Code | Constant | Meaning |
| --- | --- | --- |
| -32700 | `ErrCodeParseError` | Invalid JSON received |
| -32600 | `ErrCodeInvalidRequest` | Not a valid request object |
| -32601 | `ErrCodeMethodNotFound` | Method does not exist |
| -32602 | `ErrCodeInvalidParams` | Bad parameters |
| -32603 | `ErrCodeInternalError` | Internal error |
| -32800 | `ErrCodeRequestCancelled` | Canceled by peer or shutdown |
| -32000 | `ErrCodeAuthRequired` | `authenticate` needed first |
| -32002 | `ErrCodeResourceNotFound` | Session, file, or terminal missing |

Codes below -32000 follow the JSON-RPC reservation; ACP-specific
codes live in the implementation-defined server range. Vendor codes
can use any implementation-defined value.

## Returning errors from handlers

Return `*acp.Error` to control code and data:

```go
func (c *client) ReadTextFile(_ context.Context, req *acp.ReadTextFileRequest) (*acp.ReadTextFileResponse, error) {
    if !allowed(req.Path) {
        return nil, acp.ResourceNotFound(req.Path)
    }
    ...
}
```

Helpers:

| Helper | Produces |
| --- | --- |
| `acp.NewError(code, msg)` | bare error |
| `acp.NewErrorf(code, fmt, args...)` | formatted message |
| `e.WithData(v)` | attaches JSON `data` |
| `acp.AuthRequired()` | -32000 |
| `acp.ResourceNotFound(what)` | -32002 |
| `acp.MethodNotFound(m)` | -32601 |
| `acp.InvalidParams(msg)` | -32602 |

A non-RPC `error` from a handler is sent as `ErrCodeInternalError`
with the error text as the message.

## Inspecting errors on the caller side

```go
_, err := side.ReadTextFile(ctx, req)
if acp.IsErrorCode(err, acp.ErrCodeResourceNotFound) {
    // recover
}
var e *acp.Error
if errors.As(err, &e) {
    log.Printf("code %d: %s data %s", e.Code, e.Message, e.Data)
}
```

`acp.ErrorCodeOf(err)` returns the code, `ErrCodeInternalError` for
non-RPC errors, or 0 for nil.

`jsonrpc.ErrClosed` is returned by calls made on a closed connection
and by calls pending when the transport dies.
