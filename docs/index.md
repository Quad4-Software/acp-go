# acp-go

**acp-go** is a Go implementation of the
[Agent Client Protocol](https://agentclientprotocol.com) (ACP) version
1, the JSON-RPC protocol between code editors and coding agents.

```bash
go get github.com/Quad4-Software/acp-go
```

## What it gives you

- **Typed agent and client APIs.** Implement `acp.Agent` in your agent
  or `acp.Client` in your editor and the library handles every method
  in the protocol, including sessions, prompts, permissions,
  filesystem, terminals, and elicitations.
- **Correct tagged unions.** `ContentBlock`, `SessionUpdate`,
  `MCPServer`, and friends marshal and unmarshal with the exact wire
  shapes from the ACP schema, including extension quirks such as
  untagged stdio variants.
- **Full duplex JSON-RPC.** Concurrent calls in both directions,
  ordered notification delivery, and `$ /cancel_request` propagation
  through contexts.
- **Forward compatibility.** Unknown union variants and extension
  methods are preserved in `Raw` fields or routed to fallbacks, so
  proxies and future spec revisions keep working.
- **Zero dependencies.** Standard library only.

## Packages

| Package | Purpose |
| --- | --- |
| `acp` | Protocol model, `Agent` / `Client` interfaces, `AgentSide` / `ClientSide` |
| `acp-go/transport` | Byte stream adapters: stdio, subprocess, custom readers and writers |
| `acp-go/jsonrpc` | The connection engine underneath both sides |

## At a glance

An agent serving on stdio:

```go
side := acp.NewAgentSide(transport.Stdio(), myAgent)
side.Run(context.Background())
```

A client driving an agent subprocess:

```go
tr, _ := transport.NewCommandTransport(exec.CommandContext(ctx, "my-agent"), os.Stderr)
side := acp.NewClientSide(tr, myClient)
go side.Run(ctx)
resp, _ := side.Prompt(ctx, &acp.PromptRequest{
    SessionID: sessionID,
    Prompt:    []acp.ContentBlock{acp.NewTextBlock("hi")},
})
```

Next: [getting started](getting-started.md) builds a working agent and
client end to end.
