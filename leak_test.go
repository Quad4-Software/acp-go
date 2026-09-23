// SPDX-License-Identifier: 0BSD

package acp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Quad4-Software/acp-go/transport"
)

// goroutineCount forces a GC and returns the live goroutine count.
func goroutineCount() int {
	runtime.GC()
	runtime.Gosched()
	return runtime.NumGoroutine()
}

// assertNoLeak polls until the goroutine count returns to baseline or
// the deadline passes, then dumps stacks on failure.
func assertNoLeak(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if goroutineCount() <= baseline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	t.Fatalf("goroutine leak: baseline %d, now %d\n%s",
		baseline, runtime.NumGoroutine(), buf[:min(n, 8192)])
}

// buildEchoAgent compiles the example agent into a temp dir so tests
// can spawn it as a real subprocess.
func buildEchoAgent(t *testing.T) string {
	t.Helper()
	name := "echo-agent"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", out, "./examples/echo-agent") // #nosec G204 -- builds in-repo example
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("cannot build echo-agent: %v\n%s", err, b)
	}
	return out
}

// TestNoLeakConnCycles runs repeated connect-call-close cycles over
// pipes. Closing the pipe ends simulates a well-behaved transport
// whose Close unblocks reads.
func TestNoLeakConnCycles(t *testing.T) {
	baseline := goroutineCount()
	for i := 0; i < 10; i++ {
		aR, aW := io.Pipe()
		bR, bW := io.Pipe()
		ta := transport.NewLine(aR, bW, nil)
		tb := transport.NewLine(bR, aW, nil)
		agent := NewAgentSide(ta, UnimplementedAgent{})
		client := NewClientSide(tb, UnimplementedClient{})
		ctx := context.Background()
		go func() { _ = agent.Run(ctx) }()
		go func() { _ = client.Run(ctx) }()

		// Round-trip a call; the unimplemented agent still responds.
		_, _ = client.Initialize(ctx, &InitializeRequest{ProtocolVersion: ProtocolVersion1})

		_ = agent.Close()
		_ = client.Close()
		// A nil closer means pipe teardown is the caller's job.
		_ = aR.Close()
		_ = aW.Close()
		_ = bR.Close()
		_ = bW.Close()
	}
	assertNoLeak(t, baseline)
}

// TestNoLeakFullSession exercises a full agent-client session over
// pipes and verifies every goroutine exits.
func TestNoLeakFullSession(t *testing.T) {
	baseline := goroutineCount()

	aR, aW := io.Pipe()
	bR, bW := io.Pipe()
	agent := NewAgentSide(transport.NewLine(aR, bW, nil), UnimplementedAgent{})
	client := NewClientSide(transport.NewLine(bR, aW, nil), UnimplementedClient{})
	ctx := context.Background()
	go func() { _ = agent.Run(ctx) }()
	go func() { _ = client.Run(ctx) }()

	_, _ = client.Initialize(ctx, &InitializeRequest{ProtocolVersion: ProtocolVersion1})
	_, _ = client.NewSession(ctx, &NewSessionRequest{CWD: "."})
	_ = client.Cancel(ctx, "session-1")

	_ = agent.Close()
	_ = client.Close()
	_ = aR.Close()
	_ = aW.Close()
	_ = bR.Close()
	_ = bW.Close()

	assertNoLeak(t, baseline)
}

// TestNoLeakCommandTransport spawns and kills a real agent subprocess.
// The stderr forwarder, wait goroutine, and read loop must all exit.
func TestNoLeakCommandTransport(t *testing.T) {
	bin := buildEchoAgent(t)
	baseline := goroutineCount()

	tr, err := transport.NewCommandTransport(exec.Command(bin), io.Discard) // #nosec G204 -- test-built binary
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	side := NewClientSide(tr, UnimplementedClient{})
	ctx := context.Background()
	go func() { _ = side.Run(ctx) }()

	if _, err := side.Initialize(ctx, &InitializeRequest{
		ProtocolVersion: ProtocolVersion1,
	}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	_ = side.Close() // kills the child
	_ = tr.Wait()

	assertNoLeak(t, baseline)
}
