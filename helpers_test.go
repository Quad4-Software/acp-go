// SPDX-License-Identifier: 0BSD

package acp

import (
	"io"

	"github.com/Quad4-Software/acp-go/transport"
)

// pipePair returns two transports connected back to back.
func pipePair() (*transport.Line, *transport.Line) {
	aR, aW := io.Pipe()
	bR, bW := io.Pipe()
	return transport.NewLine(aR, bW, nil), transport.NewLine(bR, aW, nil)
}
