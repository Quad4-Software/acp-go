# Tagged unions

ACP models several types as tagged unions: a discriminator field plus
a merged object payload. acp-go represents each as a struct with a
discriminator, one pointer field per variant, custom JSON marshal and
unmarshal, and a `Raw` field for forward compatibility.

## Reading a union

Set `Type` (or the equivalent discriminator) directly or let
unmarshalling populate it. Exactly one variant pointer is non-nil:

```go
var b acp.ContentBlock
json.Unmarshal(data, &b)
switch b.TypeOf() {
case acp.ContentBlockText:
    fmt.Println(b.Text.Text)
case acp.ContentBlockImage:
    decode(b.Image.Data)
case acp.ContentBlockResourceLink:
    open(b.ResourceLink.URI)
default:
    // b.Raw holds unrecognized variants verbatim
}
```

`TypeOf` returns the discriminator: the explicit `Type` when set,
otherwise the variant inferred from the populated pointer.

## Writing a union

Use the constructors or set a variant pointer directly. Marshal infers
the discriminator from the populated field when `Type` is empty:

```go
blocks := []acp.ContentBlock{
    acp.NewTextBlock("hello"),
    acp.NewImageBlock(data, "image/png"),
    acp.NewResourceLinkBlock("file:///main.go", "main.go"),
    acp.NewResourceBlock(acp.ResourceContents{Text: &textRes}),
}
```

Updates work the same way:

```go
side.SessionUpdate(ctx, sid, acp.NewPlanUpdate(acp.Plan{
    Entries: []acp.PlanEntry{{Content: "step", Priority: acp.PriorityHigh}},
}))
```

## Union inventory

| Type | Discriminator | Variants |
| --- | --- | --- |
| `ContentBlock` | `type` | text, image, audio, resource_link, resource |
| `ToolCallContent` | `type` | content, diff, terminal |
| `SessionUpdate` | `sessionUpdate` | user/agent message chunk, thought chunk, tool_call, tool_call_update, plan, available_commands, current_mode, config_option, session_info, usage |
| `RequestPermissionOutcome` | `outcome` | selected, cancelled |
| `MCPServer` | `type` | http, sse, stdio |
| `AuthMethod` | `type` | terminal, agent |
| `SessionConfigOption` | `type` | select, boolean |
| `SetSessionConfigOptionRequest` | `type` | boolean, value |
| `ElicitationPropertySchema` | `type` | string, number, integer, boolean, array |
| `CreateElicitationRequest` | `mode` | form, url |
| `CreateElicitationResponse` | `action` | accept, decline, cancel |
| `SessionConfigSelectOptions` | shape | flat options list or grouped list |
| `MultiSelectItems` | shape | `enum` string list or `anyOf` titled options |

## Wire quirks worth knowing

- `MCPServer` **stdio has no `type` tag** on the wire; an untagged
  server decodes as stdio.
- `AuthMethod` **agent has no `type` tag**; terminal carries
  `type: "terminal"`.
- The value form of `SetSessionConfigOptionRequest` emits no `type`;
  only the boolean form does.
- Discriminators use snake_case (`agent_message_chunk`) while field
  names are camelCase (`sessionUpdate`, `currentModeId`).

## `Raw` preservation

Unknown discriminators never fail decoding. The original JSON lands in
`Raw`, and marshaling the value returns the bytes verbatim:

```go
var u acp.SessionUpdate
json.Unmarshal([]byte(`{"sessionUpdate":"_future/thing","x":1}`), &u)
// u.Raw == {"sessionUpdate":"_future/thing","x":1}
```

Set `Raw` yourself to emit a variant the library does not know yet.
