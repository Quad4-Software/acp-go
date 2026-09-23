// SPDX-License-Identifier: 0BSD

package acp

// Method names defined by the protocol. Extension methods must start
// with an underscore.
const (
	MethodInitialize          = "initialize"
	MethodAuthenticate        = "authenticate"
	MethodLogout              = "logout"
	MethodSessionNew          = "session/new"
	MethodSessionLoad         = "session/load"
	MethodSessionList         = "session/list"
	MethodSessionDelete       = "session/delete"
	MethodSessionResume       = "session/resume"
	MethodSessionClose        = "session/close"
	MethodSessionSetMode      = "session/set_mode"
	MethodSessionSetConfig    = "session/set_config_option"
	MethodSessionPrompt       = "session/prompt"
	MethodSessionCancel       = "session/cancel"
	MethodSessionUpdate       = "session/update"
	MethodRequestPermission   = "session/request_permission"
	MethodFSReadTextFile      = "fs/read_text_file"
	MethodFSWriteTextFile     = "fs/write_text_file"
	MethodTerminalCreate      = "terminal/create"
	MethodTerminalOutput      = "terminal/output"
	MethodTerminalRelease     = "terminal/release"
	MethodTerminalWaitForExit = "terminal/wait_for_exit"
	MethodTerminalKill        = "terminal/kill"
	MethodElicitationCreate   = "elicitation/create"
	MethodElicitationComplete = "elicitation/complete"
)
