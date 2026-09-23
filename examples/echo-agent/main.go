// SPDX-License-Identifier: 0BSD

// Command echo-agent is a minimal ACP agent speaking newline-delimited
// JSON-RPC on stdio. It echoes prompts back to the client and reports
// progress through session updates and a tool call.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"

	"github.com/Quad4-Software/acp-go"
	"github.com/Quad4-Software/acp-go/transport"
)

type echoAgent struct {
	acp.UnimplementedAgent
	side   *acp.AgentSide
	nextID atomic.Int64
}

func (a *echoAgent) Initialize(_ context.Context, _ *acp.InitializeRequest) (*acp.InitializeResponse, error) {
	return &acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersion1,
		AgentCapabilities: acp.AgentCapabilities{
			PromptCapabilities: acp.PromptCapabilities{EmbeddedContext: true},
		},
		AgentInfo: &acp.Implementation{Name: "echo-agent", Version: "0.1.0"},
	}, nil
}

func (a *echoAgent) NewSession(_ context.Context, _ *acp.NewSessionRequest) (*acp.NewSessionResponse, error) {
	id := fmt.Sprintf("session-%d", a.nextID.Add(1))
	return &acp.NewSessionResponse{
		SessionID: acp.SessionID(id),
		Modes: &acp.SessionModeState{
			CurrentModeID: "echo",
			AvailableModes: []acp.SessionMode{
				{ID: "echo", Name: "Echo"},
			},
		},
	}, nil
}

func (a *echoAgent) Prompt(ctx context.Context, req *acp.PromptRequest) (*acp.PromptResponse, error) {
	var text strings.Builder
	for _, b := range req.Prompt {
		if b.Text != nil {
			text.WriteString(b.Text.Text)
		}
	}

	// Report a tool call so clients can render tool execution UI.
	if err := a.side.SessionUpdate(ctx, req.SessionID, acp.NewToolCallUpdate(acp.ToolCall{
		ToolCallID: "echo-1",
		Title:      "Echoing prompt",
		Kind:       acp.ToolKindThink,
		Status:     acp.ToolCallInProgress,
	})); err != nil {
		return nil, err
	}

	if err := a.side.SessionUpdate(ctx, req.SessionID,
		acp.NewAgentMessageChunkUpdate(acp.NewTextBlock("echo: "+text.String()))); err != nil {
		return nil, err
	}

	if err := a.side.SessionUpdate(ctx, req.SessionID, acp.NewToolCallProgressUpdate(acp.ToolCallUpdate{
		ToolCallID: "echo-1",
		Status:     acp.ToolCallCompleted,
	})); err != nil {
		return nil, err
	}
	return &acp.PromptResponse{StopReason: acp.StopEndTurn}, nil
}

func main() {
	log.SetOutput(os.Stderr)
	agent := &echoAgent{}
	side := acp.NewAgentSide(transport.Stdio(), agent)
	agent.side = side
	if err := side.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
