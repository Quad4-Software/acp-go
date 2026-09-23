// SPDX-License-Identifier: 0BSD

package acp

// ReadTextFileRequest asks the client to read a file. Path must be
// absolute. Line and Limit select a 1-based line range; both absent
// reads the whole file. Requires the readTextFile capability.
type ReadTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Line      *int64    `json:"line,omitempty"`
	Limit     *int64    `json:"limit,omitempty"`
	Meta      Meta      `json:"_meta,omitempty"`
}

// ReadTextFileResponse returns the file contents.
type ReadTextFileResponse struct {
	Content string `json:"content"`
	Meta    Meta   `json:"_meta,omitempty"`
}

// WriteTextFileRequest asks the client to write a file. Path must be
// absolute. Requires the writeTextFile capability.
type WriteTextFileRequest struct {
	SessionID SessionID `json:"sessionId"`
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Meta      Meta      `json:"_meta,omitempty"`
}

// WriteTextFileResponse acknowledges the write.
type WriteTextFileResponse struct {
	Meta Meta `json:"_meta,omitempty"`
}
