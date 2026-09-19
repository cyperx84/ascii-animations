package main

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/lint"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func animated() term.Caps { return term.Caps{Profile: term.TrueColor, Animate: true} }

func TestIntroIsStaticWhenMotionIsUnwanted(t *testing.T) {
	// The zero Caps is what CI, a pipe and ASCIIFX_REDUCED_MOTION resolve to.
	m, err := newModel(term.Caps{Profile: term.TrueColor})
	if err != nil {
		t.Fatal(err)
	}
	if cmd := m.Init(); cmd != nil {
		t.Fatal("a reduced-motion intro should start no tick chain")
	}
	// The static frame is the finished banner, not a blank first frame.
	if strings.TrimSpace(stripANSI(m.intro.View())) == "" {
		t.Fatal("static frame is blank")
	}
}

func TestIntroFramesAreTerminalSafe(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	run := m.intro.Run()
	for _, tick := range []int{0, run.Frames() / 2, run.Frames() - 1} {
		b, err := run.Seek(tick)
		if err != nil {
			t.Fatal(err)
		}
		for _, is := range lint.String(b.Plain()) {
			t.Errorf("tick %d: %d:%d: %s", tick, is.Line, is.Col, is.Message)
		}
	}
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
