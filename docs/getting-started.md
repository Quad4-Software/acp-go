# Getting started

This page walks through a complete agent and client. The same code
ships as runnable programs under `examples/` in the repository.

## Install

```bash
go get github.com/Quad4-Software/acp-go
```

The library requires Go 1.27 or newer and has no dependencies.

## A minimal agent

An agent implements the `acp.Agent` interface. Embedding
`acp.UnimplementedAgent` satisfies every method you do not need.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/Quad4-Software/acp-go"
    "github.com/Quad4-Software/acp-go/transport"
)

type agent struct {
    acp.UnimplementedAgent
    side *acp.AgentSide
}

func (a *agent) Initialize(_ context.Context, _ *acp.InitializeRequest) (*acp.InitializeResponse, error) {
    return &acp.InitializeResponse{
        ProtocolVersion: acp.ProtocolVersion1,
        AgentInfo:       &acp.Implementation{Name: "echo-agent", Version: "0.1.0"},
    }, nil
}

func (a *agent) NewSession(_ context.Context, _ *acp.NewSessionRequest) (*acp.NewSessionResponse, error) {
    return &acp.NewSessionResponse{SessionID: "session-1"}, nil
}

func (a *agent) Prompt(ctx context.Context, req *acp.PromptRequest) (*acp.PromptResponse, error) {
    for _, block := range req.Prompt {
        if block.Text == nil {
            continue
        }
        err := a.side.SessionUpdate(ctx, req.SessionID,
            acp.NewAgentMessageChunkUpdate(acp.NewTextBlock("echo: "+block.Text.Text)))
        if err != nil {
            return nil, err
        }
    }
    return &acp.PromptResponse{StopReason: acp.StopEndTurn}, nil
}

func main() {
    log.SetOutput(os.Stderr) // stdout is protocol only
    a := &agent{}
    a.side = acp.NewAgentSide(transport.Stdio(), a)
    if err := a.side.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

!!! warning
    Agents must never write to stdout except protocol messages. Logs
    go to stderr, which ACP explicitly allows.

## A minimal client

The client launches the agent as a subprocess and drives it.

```go
tr, err := transport.NewCommandTransport(
    exec.CommandContext(ctx, "./echo-agent"), os.Stderr)
if err != nil {
    return err
}
defer tr.Close()

side := acp.NewClientSide(tr, acp.UnimplementedClient{})
go side.Run(ctx)

init, err := side.Initialize(ctx, &acp.InitializeRequest{
    ProtocolVersion: acp.ProtocolVersion1,
    ClientCapabilities: acp.ClientCapabilities{
        FS: acp.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
    },
})
if err != nil {
    return err
}
fmt.Println("protocol version:", init.ProtocolVersion)

sess, err := side.NewSession(ctx, &acp.NewSessionRequest{CWD: "."})
if err != nil {
    return err
}

resp, err := side.Prompt(ctx, &acp.PromptRequest{
    SessionID: sess.SessionID,
    Prompt:    []acp.ContentBlock{acp.NewTextBlock("hello")},
})
fmt.Println("stop reason:", resp.StopReason)
```

## Try it

```bash
git clone https://github.com/Quad4-Software/acp-go
cd acp-go
go build -o .bin/echo-agent ./examples/echo-agent
go run ./examples/echo-client -agent .bin/echo-agent
```

You should see the agent stream a tool call update and an echoed
message chunk, then finish the turn with `end_turn`.

## What happens on the wire

`Initialize` sends a JSON-RPC request over the child's stdin:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{...}}}
```

Each message is a single UTF-8 line. The agent replies on stdout with
the negotiated version, the client creates a session, then
`session/prompt` runs one turn while `session/update` notifications
stream progress back.

Continue to [writing an agent](guides/agent.md) for the full agent
surface, or [writing a client](guides/client.md) for the editor side.
