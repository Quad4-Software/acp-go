# Wire format

ACP runs JSON-RPC 2.0 over newline-delimited UTF-8 JSON.

## Framing

- Each request, response, or notification is one JSON value on one
  line, terminated by `\n`.
- Messages must not contain embedded newlines; `encoding/json` output
  satisfies this by default.
- On stdio, the agent writes only protocol lines to stdout. Logs go
  to stderr. acp-go caps a single message at 64 MiB.

## Envelopes

Request:

```json
{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/repo"}}
```

Response (exactly one of `result` or `error`, which
`jsonrpc.Conn` guarantees):

```json
{"jsonrpc":"2.0","id":1,"result":{"sessionId":"s1"}}
```

Notification (no `id`):

```json
{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{...}}}
```

`RequestID` supports integer, string, and null IDs per the JSON-RPC
spec.

## Cancellation

`$/cancel_request` asks the peer to abort an in-flight request:

```json
{"jsonrpc":"2.0","method":"$/cancel_request","params":{"requestId":1}}
```

`Conn` handles this internally: canceling the context of an
in-flight `Call` sends the notification, and receiving one cancels the
matching handler's context. Agents additionally get
`session/cancel` for prompt turns; `AgentSide` wires it to the
prompt's context automatically.

## Message classification

`Conn` classifies each incoming line:

| Shape | Dispatch |
| --- | --- |
| `method` + `id` | request: handler runs in a goroutine, response follows |
| `method`, no `id` | notification: delivered sequentially in arrival order |
| `id`, no `method` | response: resolves a pending call |

Malformed JSON gets a parse-error response with a `null` id.
Requests to unregistered methods get `ErrCodeMethodNotFound` unless a
fallback handler is set. Responses to unknown IDs are dropped.

## Concurrency

Outbound calls can run concurrently and responses are matched by ID.
Notification handlers run one at a time in order; a handler may issue
outbound calls on the same connection without deadlocking.
