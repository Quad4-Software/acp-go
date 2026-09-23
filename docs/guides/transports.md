# Transports

A `transport.Transport` is a bidirectional channel of complete
JSON-RPC messages. `AgentSide`, `ClientSide`, and `jsonrpc.Conn` all
work on any implementation.

```go
type Transport interface {
    Read(ctx context.Context) (json.RawMessage, error)
    Write(ctx context.Context, msg json.RawMessage) error
    Close() error
}
```

## Built-in transports

| Constructor | Use |
| --- | --- |
| `transport.NewLine(r, w, closer)` | Newline-delimited JSON over any reader/writer pair |
| `transport.Stdio()` | `NewLine` on `os.Stdin`/`os.Stdout`; the ACP agent transport |
| `transport.NewCommandTransport(cmd, stderr)` | Spawn a child process, wire its stdin/stdout, forward stderr |
| `transport.Command(ctx, name, args...)` | Shorthand for `exec.CommandContext` + `NewCommandTransport` |

`transport.Line` frames messages exactly as the ACP spec requires:
one UTF-8 JSON value per line, `\r\n` tolerant, no embedded newlines,
capped at `transport.MaxMessageSize` (64 MiB).

## Subprocess lifecycle

`NewCommandTransport` starts the command and gives you:

- `Read` / `Write` for protocol messages
- `Process()` for the underlying `*os.Process`
- `Wait()` for the exit status
- `Close()` which closes stdin first, then kills the child if it is
  still running

```go
tr, err := transport.NewCommandTransport(
    exec.CommandContext(ctx, "my-agent", "--serve"), os.Stderr)
if err != nil {
    return err
}
defer tr.Close()
```

The command must have `Stdin`, `Stdout`, and `Stderr` unset so the
transport can wire them; a pre-wired command is rejected.

## Writing a custom transport

Anything that delivers whole `json.RawMessage` values works: TLS
sockets, named pipes, an in-memory `io.Pipe`, a WebSocket adapter.

```go
type sock struct {
    conn net.Conn
    r    *bufio.Reader
}

func (s *sock) Read(ctx context.Context) (json.RawMessage, error) {
    line, err := s.r.ReadBytes('\n')
    if err != nil {
        return nil, err
    }
    return bytes.TrimRight(line, "\r\n"), nil
}

func (s *sock) Write(ctx context.Context, msg json.RawMessage) error {
    _, err := s.conn.Write(append(msg, '\n'))
    return err
}

func (s *sock) Close() error { return s.conn.Close() }
```

Contract details:

- `Read` returns one complete message per call and `io.EOF` on a
  clean peer shutdown.
- `Write` sends one complete message. The connection layer may call
  it from several goroutines, so implementations must serialize
  writes internally (`transport.Line` does this with a mutex).
- Honor `ctx` for cancellation where possible; `Close` must unblock
  pending operations so `Conn` can shut down cleanly.
- ACP stdio transports forbid embedded newlines; keep them out of
  the payload.
