package main

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func animated() term.Caps { return term.Caps{Profile: term.TrueColor, Animate: true} }

// The claim this example makes is that the two doors paint the same pixels.
// Checked on the glyphs, because the two renderers are free to emit
// different escape sequences for the same colours.
func TestBothPathsPaintTheSameGlyphs(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	// The flag stays put across both calls: it only picks which label the
	// chrome shows, and a different label is not a different layout.
	m.drawables = true
	strs := glyphs(m.stringPath())
	canvas := glyphs(m.canvasPath())
	if strs != canvas {
		t.Fatalf("paths differ\nstrings:\n%s\ncanvas:\n%s", strs, canvas)
	}
}

func glyphs(s string) string {
	lines := strings.Split(stripANSI(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
