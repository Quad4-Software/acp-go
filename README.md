# acp-go

[![coverage](docs/coverage.svg)](docs/coverage.svg)
[![go](docs/go.svg)](go.mod)
[![license](docs/license.svg)](LICENSE)
[![protocol](docs/acp.svg)](https://agentclientprotocol.com)
[![docs](https://img.shields.io/badge/docs-acp--go-8b5cf6)](https://quad4-software.github.io/acp-go/)

Go implementation of the Agent Client Protocol (ACP) v1, the JSON-RPC
protocol between code editors and coding agents.

[Documentation](https://quad4-software.github.io/acp-go/) |
[API reference](https://pkg.go.dev/github.com/Quad4-Software/acp-go) |
[Examples](examples/)

## Install

    go get github.com/Quad4-Software/acp-go

## Layout

    acp-go/       protocol types, Agent/Client interfaces, side wrappers
    jsonrpc/      JSON-RPC 2.0 connection engine
    transport/    stdio and subprocess byte transports
    examples/     runnable echo agent and client
    tools/badge/  void-style SVG badge generator
    docs/         documentation site source

## Usage

An agent serves a connection on stdio:

```go
import (
    "github.com/Quad4-Software/acp-go"
    "github.com/Quad4-Software/acp-go/transport"
)

type agent struct {
    acp.UnimplementedAgent
    side *acp.AgentSide
}

func main() {
    a := &agent{}
    a.side = acp.NewAgentSide(transport.Stdio(), a)
    a.side.Run(context.Background())
}
```

Implement `acp.Agent` methods such as `Initialize`, `NewSession` and
`Prompt`. Inside `Prompt`, stream progress with
`side.SessionUpdate(...)` and call back into the client with
`side.ReadTextFile`, `side.RequestPermission`, `side.CreateTerminal`,
and friends. The prompt context is canceled automatically when the
client sends `session/cancel`.

A client spawns an agent subprocess and drives it:

```go
t, _ := transport.NewCommandTransport(exec.CommandContext(ctx, "my-agent"), os.Stderr)
side := acp.NewClientSide(t, myClient) // myClient implements acp.Client
go side.Run(ctx)

init, _ := side.Initialize(ctx, &acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersion1})
sess, _ := side.NewSession(ctx, &acp.NewSessionRequest{CWD: "."})
resp, _ := side.Prompt(ctx, &acp.PromptRequest{
    SessionID: sess.SessionID,
    Prompt:    []acp.ContentBlock{acp.NewTextBlock("hi")},
})
```

Runnable examples live under `examples/`:

    go build -o .bin/echo-agent ./examples/echo-agent
    go run ./examples/echo-client -agent .bin/echo-agent

Transports are newline-delimited JSON per the ACP stdio spec.
`transport.NewLine(r, w, nil)` adapts any byte stream. Extension
methods and notifications register through `side.Conn().Handle` and
`SetFallback`; `_meta` fields carry custom data.

## License

[0BSD](LICENSE). Do what you like, no warranty.
