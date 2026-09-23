# Extending the protocol

ACP reserves three extension channels and acp-go supports all of
them: `_meta` fields, underscore-prefixed methods and notifications,
and unknown union variants.

## `_meta` fields

Every request, response, and notification type carries a `Meta` field
(`map[string]any`) that marshals as `_meta`. Use it for
implementation-specific data that does not belong in the schema:

```go
resp := &acp.NewSessionResponse{
    SessionID: "s1",
    Meta:      acp.Meta{"vendor/trace-id": traceID},
}
```

## Extension methods

Methods starting with `_` are extensions. On the receiving side,
register handlers on the raw connection:

```go
side.Conn().Handle("_vendor/stats", func(ctx context.Context, params json.RawMessage) (any, error) {
    return map[string]int{"turns": n}, nil
})

side.Conn().HandleNotification("_vendor/poke", func(ctx context.Context, params json.RawMessage) {
    log.Println("poked")
})
```

Or catch every unregistered method at once:

```go
side.Conn().SetFallback(func(ctx context.Context, params json.RawMessage) (any, error) {
    return nil, acp.MethodNotFound("_unknown")
})
side.Conn().SetNotificationFallback(func(ctx context.Context, params json.RawMessage) {
    forwardToPlugin(params)
})
```

Without a fallback, unknown methods get `ErrCodeMethodNotFound`, which
is also what a correct peer expects.

Calling an extension method is a normal call:

```go
var stats struct{ Turns int `json:"turns"` }
err := side.Conn().Call(ctx, "_vendor/stats", nil, &stats)
```

## Unknown union variants

All tagged unions keep unrecognized variants in a `Raw` field holding
the original JSON. That makes proxies lossless and lets you experiment
with new update types before they land in the schema:

```go
// Send a custom session update.
_ = side.SessionUpdate(ctx, sid, acp.SessionUpdate{
    Raw: json.RawMessage(`{"sessionUpdate":"_vendor/metric","value":42}`),
})

// Receive one.
if u.Raw != nil {
    handleExtension(u.Raw)
}
```

The same applies to `ContentBlock`, `ToolCallContent`, `MCPServer`,
`AuthMethod`, `SessionConfigOption`, elicitation types, and the
permission outcome.

## Custom transports

Any `transport.Transport` implementation can carry ACP: sockets, named
pipes, WebSockets, test harnesses. See
[transports](transports.md#writing-a-custom-transport) for the
contract.
