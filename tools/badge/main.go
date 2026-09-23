// SPDX-License-Identifier: 0BSD

// Command badge renders a small SVG shield in the quad4 void style:
// dark surface, violet accent, monospace text. It is generic so it can
// render any label/value pair, not just coverage.
//
//	badge -label coverage -value 78.6% -out docs/coverage.svg
package main

import (
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Void palette, shared with quad4.io assets.
const (
	colorSurface  = "#12121a"
	colorRaised   = "#1b1b26"
	colorEdge     = "#2d2d3d"
	colorText     = "#c8c8d4"
	colorMuted    = "#8a8a9e"
	colorViolet   = "#8b5cf6"
	colorEmerald  = "#34d399"
	colorAmber    = "#fbbf24"
	colorRose     = "#fb7185"
	colorDisabled = "#52525e"
)

var accents = map[string]string{
	"violet":  colorViolet,
	"emerald": colorEmerald,
	"amber":   colorAmber,
	"rose":    colorRose,
	"gray":    colorDisabled,
}

func main() {
	var (
		label  = flag.String("label", "coverage", "left segment text")
		value  = flag.String("value", "", "right segment text; defaults to coverage percent from -coverprofile")
		cover  = flag.String("coverprofile", "", "go coverage profile to derive -value from when -value is empty")
		accent = flag.String("accent", "", "accent name (violet|emerald|amber|rose|gray); auto from percent when empty")
		out    = flag.String("out", "", "output SVG path (required)")
	)
	flag.Parse()

	val := *value
	var pct float64
	var hasPct bool
	if *cover != "" {
		p, err := coveragePercent(*cover)
		if err != nil {
			fatal("coverage: %v", err)
		}
		pct, hasPct = p, true
		if val == "" {
			val = fmt.Sprintf("%.1f%%", p)
		}
	}
	if val == "" {
		fatal("-value or -coverprofile is required")
	}
	if !hasPct {
		if p, ok := parsePercent(val); ok {
			pct, hasPct = p, true
		}
	}

	ac := *accent
	if ac == "" {
		ac = accentFor(pct, hasPct)
	}
	acColor, ok := accents[ac]
	if !ok {
		fatal("unknown accent %q", ac)
	}
	if *out == "" {
		fatal("-out is required")
	}

	svg := render(*label, val, acColor)
	if dir := filepath.Dir(*out); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			fatal("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(*out, []byte(svg), 0o600); err != nil {
		fatal("write: %v", err)
	}
	fmt.Println(*out)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "badge: "+format+"\n", args...)
	os.Exit(1)
}

// coveragePercent totals statement coverage from a go coverage profile.
func coveragePercent(path string) (float64, error) {
	data, err := os.ReadFile(path) // #nosec G304 - the coverage profile path is an explicit CLI flag
	if err != nil {
		return 0, err
	}
	var stmts, covered int
	for i, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 {
			return 0, fmt.Errorf("line %d: malformed", i+1)
		}
		n, err := strconv.Atoi(f[1])
		if err != nil {
			return 0, fmt.Errorf("line %d: %w", i+1, err)
		}
		cnt, err := strconv.Atoi(f[2])
		if err != nil {
			return 0, fmt.Errorf("line %d: %w", i+1, err)
		}
		stmts += n
		if cnt > 0 {
			covered += n
		}
	}
	if stmts == 0 {
		return 0, fmt.Errorf("no statements in %s", path)
	}
	return float64(covered) * 100 / float64(stmts), nil
}

func parsePercent(s string) (float64, bool) {
	p, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return p, err == nil
}

func accentFor(pct float64, ok bool) string {
	if !ok {
		return "violet"
	}
	switch {
	case pct >= 80:
		return "violet"
	case pct >= 60:
		return "emerald"
	case pct >= 40:
		return "amber"
	default:
		return "rose"
	}
}

// textWidth approximates rendered width for a monospace font at 11px.
func textWidth(s string) int {
	return len(s)*7 + 8
}

func render(label, value, accent string) string {
	lw, vw := textWidth(label), textWidth(value)
	h, y := 20, 13
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img" aria-label="%s: %s">
<rect width="%d" height="%d" rx="4" fill="%s"/>
<rect x="%d" width="%d" height="%d" rx="4" fill="%s"/>
<rect x="%d" width="4" height="%d" fill="%s"/>
<rect x="0.5" y="0.5" width="%d" height="%d" rx="3.5" fill="none" stroke="%s"/>
<text x="%d" y="%d" fill="%s" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="11" text-anchor="middle">%s</text>
<text x="%d" y="%d" fill="%s" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="11" text-anchor="middle" font-weight="bold">%s</text>
</svg>
`,
		lw+vw, h, esc(label), esc(value),
		lw+vw, h, colorSurface,
		lw, vw, h, accent,
		lw, h, accent,
		lw+vw-1, h-1, colorEdge,
		lw/2, y, colorMuted, esc(label),
		lw+vw/2, y, colorSurface, esc(value),
	)
}

func esc(s string) string {
	return html.EscapeString(s)
}
