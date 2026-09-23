// SPDX-License-Identifier: 0BSD

package acp

// CreateTerminalRequest asks the client to run a command in a new
// terminal. Requires the terminal capability.
type CreateTerminalRequest struct {
	SessionID       SessionID     `json:"sessionId"`
	Command         string        `json:"command"`
	Args            []string      `json:"args,omitempty"`
	Env             []EnvVariable `json:"env,omitempty"`
	CWD             string        `json:"cwd,omitempty"`
	OutputByteLimit *int64        `json:"outputByteLimit,omitempty"`
	Meta            Meta          `json:"_meta,omitempty"`
}

// CreateTerminalResponse returns the new terminal ID.
type CreateTerminalResponse struct {
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// TerminalOutputRequest asks for a terminal's current output and exit
// status.
type TerminalOutputRequest struct {
	SessionID  SessionID  `json:"sessionId"`
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// TerminalExitStatus reports how a terminal command ended.
type TerminalExitStatus struct {
	ExitCode *int64 `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}

// TerminalOutputResponse returns terminal output and exit status.
type TerminalOutputResponse struct {
	Output     string              `json:"output"`
	Truncated  bool                `json:"truncated"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
	Meta       Meta                `json:"_meta,omitempty"`
}

// WaitForTerminalExitRequest blocks until the command exits.
type WaitForTerminalExitRequest struct {
	SessionID  SessionID  `json:"sessionId"`
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// WaitForTerminalExitResponse returns the exit status.
type WaitForTerminalExitResponse struct {
	ExitCode *int64 `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}

// KillTerminalRequest kills the command without releasing the terminal.
type KillTerminalRequest struct {
	SessionID  SessionID  `json:"sessionId"`
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// KillTerminalResponse acknowledges the kill.
type KillTerminalResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}

// ReleaseTerminalRequest releases a terminal and its resources.
type ReleaseTerminalRequest struct {
	SessionID  SessionID  `json:"sessionId"`
	TerminalID TerminalID `json:"terminalId"`
	Meta       Meta       `json:"_meta,omitempty"`
}

// ReleaseTerminalResponse acknowledges the release.
type ReleaseTerminalResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}
