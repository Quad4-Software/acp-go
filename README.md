# acp-go

[![coverage](docs/coverage.svg)](docs/coverage.svg)
[![go](docs/go.svg)](go.mod)
[![license](docs/license.svg)](LICENSE)
[![protocol](docs/acp.svg)](https://agentclientprotocol.com)

Go implementation of the Agent Client Protocol (ACP) v1, the JSON-RPC
protocol between code editors and coding agents.

## Install

    go get github.com/Quad4-Software/acp-go

## Usage

An agent serves a connection on stdio:

```go
type agent struct {
    acp.UnimplementedAgent
    side *acp.AgentSide
}

func main() {
    a := &agent{}
    a.side = acp.NewAgentSide(acp.StdioTransport(), a)
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
t, _ := acp.NewCommandTransport(exec.CommandContext(ctx, "my-agent"), os.Stderr)
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
`acp.NewLineTransport(r, w, nil)` adapts any byte stream. Extension
methods and notifications register through `side.Conn().Handle` and
`SetFallback`; `_meta` fields carry custom data.

License: 0BSD.
