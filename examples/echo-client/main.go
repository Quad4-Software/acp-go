// SPDX-License-Identifier: 0BSD

// Command echo-client spawns an ACP agent subprocess and drives it
// through the initialize, session/new, and session/prompt flow,
// printing session updates to stdout.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/Quad4-Software/acp-go"
	"github.com/Quad4-Software/acp-go/transport"
)

type printingClient struct {
	acp.UnimplementedClient
}

func (printingClient) SessionUpdate(_ context.Context, n *acp.SessionNotification) {
	u := n.Update
	switch {
	case u.AgentMessageChunk != nil && u.AgentMessageChunk.Content.Text != nil:
		fmt.Println(u.AgentMessageChunk.Content.Text.Text)
	case u.ToolCall != nil:
		fmt.Printf("[tool %s] %s (%s)\n", u.ToolCall.ToolCallID, u.ToolCall.Title, u.ToolCall.Status)
	case u.ToolCallUpdate != nil:
		fmt.Printf("[tool %s] -> %s\n", u.ToolCallUpdate.ToolCallID, u.ToolCallUpdate.Status)
	}
}

func run(ctx context.Context, agentCmd, prompt string) error {
	// The agent path is user-supplied on purpose: this example launches
	// whatever agent binary the caller passes.
	cmd := exec.CommandContext(ctx, agentCmd) // #nosec G204 -- intentional, this is an agent launcher
	tr, err := transport.NewCommandTransport(cmd, os.Stderr)
	if err != nil {
		return err
	}
	defer func() { _ = tr.Close() }()

	side := acp.NewClientSide(tr, printingClient{})
	go func() { _ = side.Run(ctx) }()

	init, err := side.Initialize(ctx, &acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersion1,
		ClientCapabilities: acp.ClientCapabilities{FS: acp.FileSystemCapabilities{ReadTextFile: true}},
		ClientInfo:         &acp.Implementation{Name: "echo-client", Version: "0.1.0"},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	name, version := "agent", "?"
	if init.AgentInfo != nil {
		name, version = init.AgentInfo.Name, init.AgentInfo.Version
	}
	fmt.Printf("connected to %s %s (protocol v%d)\n", name, version, init.ProtocolVersion)

	sess, err := side.NewSession(ctx, &acp.NewSessionRequest{CWD: "."})
	if err != nil {
		return fmt.Errorf("session/new: %w", err)
	}

	resp, err := side.Prompt(ctx, &acp.PromptRequest{
		SessionID: sess.SessionID,
		Prompt:    []acp.ContentBlock{acp.NewTextBlock(prompt)},
	})
	if err != nil {
		return fmt.Errorf("session/prompt: %w", err)
	}
	fmt.Printf("turn ended: %s\n", resp.StopReason)
	return nil
}

func main() {
	agentCmd := flag.String("agent", "", "agent command to spawn")
	prompt := flag.String("prompt", "hello acp", "prompt text")
	flag.Parse()
	if *agentCmd == "" {
		log.Fatal("pass -agent <command>")
	}
	if err := run(context.Background(), *agentCmd, *prompt); err != nil {
		log.Fatal(err)
	}
}
