// SPDX-License-Identifier: 0BSD

// Package transport adapts byte streams into channels of whole
// JSON-RPC messages: newline-delimited lines, process stdio, and any
// custom reader and writer pair.
package transport

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

// Line frames JSON-RPC messages as newline-delimited JSON over
// an io.Reader and io.Writer, as required by the ACP stdio transport.
// Messages must not contain embedded newlines.
type Line struct {
	w   io.Writer
	wmu sync.Mutex
	c   io.Closer

	lines   chan readResult
	done    chan struct{}
	doneOne sync.Once
	lastErr error
}

// readResult is one line or the terminal read error from serve.
type readResult struct {
	line []byte
	err  error
}

// NewLine returns a transport over r and w. If closer is
// non-nil it is called by Close.
func NewLine(r io.Reader, w io.Writer, closer io.Closer) *Line {
	t := &Line{
		w:     w,
		c:     closer,
		lines: make(chan readResult, 32),
		done:  make(chan struct{}),
	}
	go t.serve(bufio.NewReaderSize(r, 1<<20))
	return t
}

// Stdio returns the transport an agent uses: newline-delimited
// JSON on stdin and stdout. Nothing else may be written to stdout. Use
// stderr for logging.
func Stdio() *Line {
	return NewLine(os.Stdin, os.Stdout, nil)
}

// serve is the single reader goroutine. It delivers each line and the
// terminal error over lines, then closes the channel. lastErr is set
// before close so drained readers see the same terminal error.
func (t *Line) serve(r *bufio.Reader) {
	for {
		line, err := r.ReadBytes('\n')
		if err == nil && len(line) > MaxMessageSize {
			err = fmt.Errorf("transport: message exceeds %d bytes", MaxMessageSize)
		}
		res := readResult{err: err}
		if err == nil {
			res.line = trimLine(line)
		}
		select {
		case t.lines <- res:
		case <-t.done:
		}
		if err != nil {
			t.lastErr = err
			close(t.lines)
			return
		}
		select {
		case <-t.done:
			t.lastErr = io.EOF
			close(t.lines)
			return
		default:
		}
	}
}

// trimLine removes a trailing \n and optional \r.
func trimLine(line []byte) []byte {
	if n := len(line); n > 0 && line[n-1] == '\n' {
		line = line[:n-1]
		if n > 1 && line[n-2] == '\r' {
			line = line[:n-2]
		}
	}
	return line
}

// Read reads one newline-delimited message. The message is not
// validated; malformed JSON is reported to the peer by the Conn layer.
// A canceled Read never loses or corrupts a partially read line: the
// next Read continues from the same stream position.
func (t *Line) Read(ctx context.Context) (json.RawMessage, error) {
	select {
	case res, ok := <-t.lines:
		if !ok {
			if t.lastErr != nil {
				return nil, t.lastErr
			}
			return nil, io.EOF
		}
		if res.err != nil {
			return nil, res.err
		}
		return json.RawMessage(res.line), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.done:
		return nil, io.EOF
	}
}

// Write writes one message followed by a newline.
func (t *Line) Write(ctx context.Context, msg json.RawMessage) error {
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

// Close releases transport resources and unblocks pending reads. If the
// underlying reader is a stream that cannot be closed from here (for
// example a pipe owned by the caller), the reader goroutine stays
// parked on it until input arrives or it reaches EOF.
func (t *Line) Close() error {
	t.doneOne.Do(func() { close(t.done) })
	if t.c != nil {
		return t.c.Close()
	}
	return nil
}

// CommandTransport runs a child process and speaks newline-delimited
// JSON over its stdin and stdout. The child's stderr is forwarded to
// the provided writer, or discarded when nil. This is the transport a
// client uses to talk to a local agent subprocess.
type CommandTransport struct {
	cmd     *exec.Cmd
	line    *Line
	stdin   io.Closer
	exited  chan struct{}
	waitErr error
}

// NewCommandTransport starts cmd, which must have Stdin, Stdout, and
// Stderr unset so the transport can wire them. It returns a transport
// speaking to the child.
func NewCommandTransport(cmd *exec.Cmd, stderr io.Writer) (*CommandTransport, error) {
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil {
		return nil, errors.New("transport: cmd Stdin, Stdout and Stderr must be unset")
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
	t := &CommandTransport{cmd: cmd, stdin: stdin, exited: make(chan struct{})}
	t.line = NewLine(stdout, stdin, nil)
	go func() {
		t.waitErr = cmd.Wait()
		close(t.exited)
	}()
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
// It is safe to call Wait and Close together; both report the same
// exit error.
func (t *CommandTransport) Wait() error {
	<-t.exited
	return t.waitErr
}

// Close closes the child stdin, then kills the process if it has not
// exited.
func (t *CommandTransport) Close() error {
	_ = t.stdin.Close()
	select {
	case <-t.exited:
	default:
		_ = t.cmd.Process.Kill()
	}
	return t.Wait()
}
