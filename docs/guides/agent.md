# Writing an agent

An agent serves an `acp.AgentSide` connection on a transport, usually
`transport.Stdio()`. The `acp.Agent` interface has one Go method per
ACP agent method.

## Interface

| Method | ACP method | Typical use |
| --- | --- | --- |
| `Initialize` | `initialize` | Negotiate protocol version, advertise capabilities |
| `Authenticate` | `authenticate` | Run an auth method the client selected |
| `Logout` | `logout` | Drop the authenticated state |
| `NewSession` | `session/new` | Create a session, return its ID and modes |
| `LoadSession` | `session/load` | Resume with conversation replay |
| `ResumeSession` | `session/resume` | Resume without replay |
| `CloseSession` | `session/close` | Free session resources |
| `ListSessions` | `session/list` | Enumerate known sessions |
| `DeleteSession` | `session/delete` | Drop session history |
| `SetSessionMode` | `session/set_mode` | Switch operating mode |
| `SetSessionConfigOption` | `session/set_config_option` | Apply a config change |
| `Prompt` | `session/prompt` | Run one prompt turn |
| `Cancel` | `session/cancel` | Notification: abort the running turn |

Embed `acp.UnimplementedAgent` so unimplemented methods return a clean
"not implemented" error instead of crashing.

```go
type myAgent struct {
    acp.UnimplementedAgent
    side *acp.AgentSide
}
```

## Serving

```go
side := acp.NewAgentSide(transport.Stdio(), &myAgent{})
err := side.Run(ctx)
```

`Run` blocks until the client disconnects, the context is canceled, or
`side.Close()` is called. Pass `nil` as the agent to serve a stub that
rejects everything.

## Prompt turns

`Prompt` runs one turn. Stream progress back to the client with
`SessionUpdate`:

```go
func (a *myAgent) Prompt(ctx context.Context, req *acp.PromptRequest) (*acp.PromptResponse, error) {
    // Report a tool call starting.
    _ = a.side.SessionUpdate(ctx, req.SessionID, acp.NewToolCallUpdate(acp.ToolCall{
        ToolCallID: "tc-1", Title: "Reading file", Kind: acp.ToolKindRead,
        Status: acp.ToolCallInProgress,
    }))
    // Stream an answer.
    _ = a.side.SessionUpdate(ctx, req.SessionID,
        acp.NewAgentMessageChunkUpdate(acp.NewTextBlock("working on it")))
    // Finish the tool call and the turn.
    _ = a.side.SessionUpdate(ctx, req.SessionID, acp.NewToolCallProgressUpdate(acp.ToolCallUpdate{
        ToolCallID: "tc-1", Status: acp.ToolCallCompleted,
    }))
    return &acp.PromptResponse{StopReason: acp.StopEndTurn}, nil
}
```

Stop reasons: `acp.StopEndTurn`, `acp.StopMaxTokens`,
`acp.StopMaxTurnRequests`, `acp.StopRefusal`, `acp.StopCancelled`.

## Cancellation

Two mechanisms abort a prompt:

- The client sends `session/cancel` for the session. `AgentSide`
  tracks the prompt per session and **cancels the prompt context
  automatically** before calling `Agent.Cancel` as a notification.
- The client sends the JSON-RPC `$/cancel_request` for the request ID.
  The connection layer cancels the handler context directly.

A well-behaved `Prompt` checks `ctx.Done()` and returns
`StopCancelled`:

```go
select {
case <-ctx.Done():
    return &acp.PromptResponse{StopReason: acp.StopCancelled}, nil
case <-workDone:
}
```

## Calling back into the client

`AgentSide` exposes typed methods for every client-side capability:

| Method | Purpose |
| --- | --- |
| `RequestPermission` | Ask the user to authorize a tool call |
| `ReadTextFile` / `WriteTextFile` | Read and write files via the client |
| `CreateTerminal`, `TerminalOutput`, `WaitForTerminalExit`, `KillTerminal`, `ReleaseTerminal` | Run commands in client terminals |
| `CreateElicitation` | Collect structured input or send the user to a URL |
| `NotifyElicitationComplete` | Report a URL elicitation finished out of band |
| `SessionUpdate` | Stream `session/update` notifications |

```go
perm, err := a.side.RequestPermission(ctx, &acp.RequestPermissionRequest{
    SessionID: req.SessionID,
    ToolCall:  acp.ToolCallUpdate{ToolCallID: "tc-1", Title: "Delete file"},
    Options: []acp.PermissionOption{
        {OptionID: "allow", Name: "Allow once", Kind: acp.PermissionAllowOnce},
        {OptionID: "deny", Name: "Deny", Kind: acp.PermissionRejectOnce},
    },
})
if perm.Outcome.Selected == nil {
    // cancelled or denied
}
```

## Capabilities

Advertise what the agent supports in `Initialize` so clients do not
call dead ends:

```go
AgentCapabilities: acp.AgentCapabilities{
    LoadSession:        true,
    PromptCapabilities: acp.PromptCapabilities{Image: true, EmbeddedContext: true},
    MCPCapabilities:    acp.MCPCapabilities{HTTP: true},
    AuthMethods:        []acp.AuthMethod{{Agent: &acp.AuthMethodAgent{ID: "oauth", Name: "OAuth"}}},
}
```

## Errors

Return `*acp.Error` to control the code on the wire:

```go
return nil, acp.ResourceNotFound(req.SessionID.String())
return nil, acp.AuthRequired()
return nil, acp.NewError(acp.ErrCodeInvalidParams, "bad prompt")
```

Any other error is sent as `ErrCodeInternalError` with the message
text. See [errors](../protocol/errors.md) for the full code table.
