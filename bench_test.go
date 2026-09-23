// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/Quad4-Software/acp-go/transport"
)

func BenchmarkTextBlockMarshal(b *testing.B) {
	blk := NewTextBlock("benchmark text")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(blk); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSessionUpdateRoundTrip(b *testing.B) {
	u := NewAgentMessageChunkUpdate(NewTextBlock("chunk"))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(u)
		if err != nil {
			b.Fatal(err)
		}
		var out SessionUpdate
		if err := json.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConnCallRoundTrip measures one request/response over an
// in-memory pipe pair.
func BenchmarkConnCallRoundTrip(b *testing.B) {
	aR, aW := io.Pipe()
	bR, bW := io.Pipe()
	srv := NewAgentSide(transport.NewLine(aR, bW, nil), UnimplementedAgent{})
	cli := NewClientSide(transport.NewLine(bR, aW, nil), UnimplementedClient{})
	ctx := context.Background()
	go func() { _ = srv.Run(ctx) }()
	go func() { _ = cli.Run(ctx) }()
	b.Cleanup(func() {
		_ = srv.Close()
		_ = cli.Close()
		_ = aR.Close()
		_ = aW.Close()
		_ = bR.Close()
		_ = bW.Close()
	})

	req := &InitializeRequest{ProtocolVersion: ProtocolVersion1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// UnimplementedAgent rejects; the call still round-trips.
		_, _ = cli.Initialize(ctx, req)
	}
}
