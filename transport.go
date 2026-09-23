// SPDX-License-Identifier: 0BSD

package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// MaxMessageSize bounds a single JSON-RPC message. ACP messages must not
// contain embedded newlines, so this is also the maximum line length.
const MaxMessageSize = 64 << 20

// Transport is a bidirectional channel for JSON-RPC messages. Each call
// to Read returns one complete message. Write sends one complete message.
// Implementations must be safe for concurrent use by one reader and one
// writer.
type Transport interface {
	// Read returns the next message. It returns io.EOF when the peer
	// has closed the channel.
	Read(ctx context.Context) (json.RawMessage, error)
	// Write sends a message.
	Write(ctx context.Context, msg json.RawMessage) error
	// Close releases transport resources.
	Close() error
}

// LineTransport frames JSON-RPC messages as newline-delimited JSON over
// an io.Reader and io.Writer, as required by the ACP stdio transport.
// Messages must not contain embedded newlines.
type LineTransport struct {
	r *bufio.Reader
	w io.Writer

	wmu sync.Mutex
	c   io.Closer
}

// NewLineTransport returns a transport over r and w. If closer is
// non-nil it is called by Close.
func NewLineTransport(r io.Reader, w io.Writer, closer io.Closer) *LineTransport {
	return &LineTransport{
		r: bufio.NewReaderSize(r, 1<<20),
		w: w,
		c: closer,
	}
}

// StdioTransport returns the transport an agent uses: newline-delimited
// JSON on stdin and stdout. Nothing else may be written to stdout. Use
// stderr for logging.
func StdioTransport() *LineTransport {
	return NewLineTransport(os.Stdin, os.Stdout, nil)
}

// Read reads one newline-delimited message. The message is not
// validated; malformed JSON is reported to the peer by the Conn layer.
func (t *LineTransport) Read(ctx context.Context) (json.RawMessage, error) {
	line, err := readLine(ctx, t.r)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(line), nil
}

// Write writes one message followed by a newline.
func (t *LineTransport) Write(ctx context.Context, msg json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.wmu.Lock()
	defer t.wmu.Unlock()
	if _, err := t.w.Write(msg); err != nil {
		return err
	}
	_, err := t.w.Write([]byte{'\n'})
	return err
}

// Close releases transport resources.
func (t *LineTransport) Close() error {
	if t.c != nil {
		return t.c.Close()
	}
	return nil
}

// readLine reads a single line, honoring context cancellation between
// reads. bufio.Reader has no cancellable Read, so reads happen on a
// helper goroutine that is abandoned on cancellation; the next Read call
// continues from the same reader. Abandoned reads can consume partial
// input, so callers should prefer closing the transport to cancel.
func readLine(ctx context.Context, r *bufio.Reader) ([]byte, error) {
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadBytes('\n')
		if err == nil && len(line) > MaxMessageSize {
			err = fmt.Errorf("acp: message exceeds %d bytes", MaxMessageSize)
		}
		ch <- result{line, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		line := res.line
		if n := len(line); n > 0 && line[n-1] == '\n' {
			line = line[:n-1]
			if n > 1 && line[n-2] == '\r' {
				line = line[:n-2]
			}
		}
		return line, nil
	}
}

// CommandTransport runs a child process and speaks newline-delimited
// JSON over its stdin and stdout. The child's stderr is forwarded to
// the provided writer, or discarded when nil. This is the transport a
// client uses to talk to a local agent subprocess.
type CommandTransport struct {
	cmd   *exec.Cmd
	line  *LineTransport
	stdin io.Closer
	done  chan error
}

// NewCommandTransport starts cmd, which must have Stdin, Stdout, and
// Stderr unset so the transport can wire them. It returns a transport
// speaking to the child.
func NewCommandTransport(cmd *exec.Cmd, stderr io.Writer) (*CommandTransport, error) {
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil {
		return nil, errors.New("acp: cmd Stdin, Stdout and Stderr must be unset")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if stderr != nil {
		go func() { _, _ = io.Copy(stderr, errPipe) }()
	} else {
		go func() { _, _ = io.Copy(io.Discard, errPipe) }()
	}
	t := &CommandTransport{cmd: cmd, stdin: stdin, done: make(chan error, 1)}
	t.line = NewLineTransport(stdout, stdin, nil)
	go func() { t.done <- cmd.Wait() }()
	return t, nil
}

// Command returns a transport for the named child process. The command
// path comes from the caller, which is the intended use of this
// transport: launching a user-configured agent binary.
func Command(ctx context.Context, name string, args ...string) (*CommandTransport, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- launching a user-configured agent is the purpose of this API
	return NewCommandTransport(cmd, nil)
}

// Read reads the next message from the child.
func (t *CommandTransport) Read(ctx context.Context) (json.RawMessage, error) {
	return t.line.Read(ctx)
}

// Write sends a message to the child.
func (t *CommandTransport) Write(ctx context.Context, msg json.RawMessage) error {
	return t.line.Write(ctx, msg)
}

// Process returns the running child process.
func (t *CommandTransport) Process() *os.Process { return t.cmd.Process }

// Wait blocks until the child exits and returns its exit error, if any.
func (t *CommandTransport) Wait() error { return <-t.done }

// Close closes the child stdin, then kills the process if it has not
// exited.
func (t *CommandTransport) Close() error {
	_ = t.stdin.Close()
	select {
	case err := <-t.done:
		return err
	default:
	}
	_ = t.cmd.Process.Kill()
	return <-t.done
}
