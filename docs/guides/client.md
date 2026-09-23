# Writing a client

A client is the editor or UI. It launches or attaches to an agent,
then drives it through `acp.ClientSide`. It also answers requests from
the agent (file access, terminals, permissions, elicitations) through
the `acp.Client` interface.

## Connecting

```go
tr, err := transport.NewCommandTransport(
    exec.CommandContext(ctx, "my-agent"), os.Stderr)
if err != nil {
    return err
}
side := acp.NewClientSide(tr, &myClient{})
go side.Run(ctx)
defer side.Close()
```

The child's stderr is forwarded to the writer you pass (`os.Stderr`
here) since agent logs are meant for the user, not the protocol. Use
`transport.Command(ctx, "my-agent")` for the shorthand.

## The handshake

```go
init, err := side.Initialize(ctx, &acp.InitializeRequest{
    ProtocolVersion: acp.ProtocolVersion1,
    ClientCapabilities: acp.ClientCapabilities{
        FS:       acp.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
        Terminal: true,
    },
    ClientInfo: &acp.Implementation{Name: "my-editor", Version: "1.0"},
})
```

Always check `init.ProtocolVersion` against what you sent; the agent
picks the version it supports. If the agent requires auth, call
`side.Authenticate` with one of `init.AuthMethods`, and `side.Logout`
to end it.

## Sessions and prompts

```go
sess, err := side.NewSession(ctx, &acp.NewSessionRequest{
    CWD: workDir,
    MCPServers: []acp.MCPServer{{Stdio: &acp.MCPServerStdio{
        Name: "fs", Command: "mcp-fs", Args: []string{"--root", workDir},
    }}},
})

resp, err := side.Prompt(ctx, &acp.PromptRequest{
    SessionID: sess.SessionID,
    Prompt:    []acp.ContentBlock{acp.NewTextBlock("fix the build")},
})
```

`Prompt` blocks for the whole turn; run it in a goroutine if the UI
must stay interactive, and call `side.Cancel(ctx, sess.SessionID)` to
abort. Session lifecycle calls: `LoadSession`, `ResumeSession`,
`ListSessions`, `CloseSession`, `DeleteSession`, `SetSessionMode`,
`SetSessionConfigOption`.

## Handling updates

`SessionUpdate` notifications arrive on the `Client` implementation as
turn progress:

```go
func (c *myClient) SessionUpdate(_ context.Context, n *acp.SessionNotification) {
    switch u := n.Update; {
    case u.AgentMessageChunk != nil && u.AgentMessageChunk.Content.Text != nil:
        appendToView(u.AgentMessageChunk.Content.Text.Text)
    case u.AgentThoughtChunk != nil:
        // reasoning text
    case u.ToolCall != nil, u.ToolCallUpdate != nil:
        refreshToolPanel(u)
    case u.Plan != nil:
        renderPlan(u.Plan.Entries)
    case u.AvailableCommands != nil:
        registerCommands(u.AvailableCommands.AvailableCommands)
    case u.CurrentMode != nil:
        setMode(u.CurrentMode.CurrentModeID)
    case u.Raw != nil:
        // extension update: forward or ignore
    }
}
```

Delivery is ordered and sequential per connection.

## Answering agent requests

When the agent calls back, the matching `Client` method runs on the
connection:

```go
func (c *myClient) RequestPermission(_ context.Context, req *acp.RequestPermissionRequest) (*acp.RequestPermissionResponse, error) {
    id := showDialog(req.ToolCall, req.Options)
    if id == "" {
        return &acp.RequestPermissionResponse{Outcome: acp.NewCancelledOutcome()}, nil
    }
    return &acp.RequestPermissionResponse{Outcome: acp.NewSelectedOutcome(id)}, nil
}

func (c *myClient) ReadTextFile(_ context.Context, req *acp.ReadTextFileRequest) (*acp.ReadTextFileResponse, error) {
    b, err := os.ReadFile(req.Path) // paths are absolute per the spec
    if err != nil {
        return nil, acp.ResourceNotFound(req.Path)
    }
    return &acp.ReadTextFileResponse{Content: string(b)}, nil
}
```

Implement `CreateTerminal`, `TerminalOutput`, `WaitForTerminalExit`,
`KillTerminal`, `ReleaseTerminal` to offer real terminals, and
`CreateElicitation` to render form or URL elicitations.
`ElicitationComplete` fires when a URL elicitation completes out of
band.

Embed `acp.UnimplementedClient` for the methods you do not support;
capability flags in `Initialize` should match what you implement.
