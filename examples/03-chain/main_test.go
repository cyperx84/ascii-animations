package main

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/lint"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func animated() term.Caps { return term.Caps{Profile: term.TrueColor, Animate: true} }

// The claim the filter makes: fire runs, and the banner survives it. Checked
// on the frame in the middle of the fire step, where unfiltered fire would
// have eaten the letters.
func TestBannerSurvivesTheFireStep(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	run := m.chain.Run()
	fireTick := run.Frames() * 6 / 10
	b, err := run.Seek(fireTick)
	if err != nil {
		t.Fatal(err)
	}
	frame := b.Plain()
	// The block font draws with █; fire draws with halfblocks. If the filter
	// were ignored, the banner's rows would be overwritten by flames.
	if !strings.Contains(frame, "█") {
		t.Fatalf("no banner glyphs left at tick %d:\n%s", fireTick, frame)
	}
}

func TestEveryFrameIsTerminalSafe(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	run := m.chain.Run()
	frames := run.Frames()
	for tick := 0; tick < frames; tick += max(frames/12, 1) {
		b, err := run.Seek(tick)
		if err != nil {
			t.Fatal(err)
		}
		for _, is := range lint.String(b.Plain()) {
			t.Errorf("tick %d: %d:%d: %s", tick, is.Line, is.Col, is.Message)
		}
	}
}

// A chain is one run, so it has one length: the sum of its steps.
func TestChainRunsForTheSumOfItsSteps(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	spec := m.chain.Run().Spec
	if got, want := spec.Duration, 1.6+1.4+0.9; got != want {
		t.Fatalf("chain is %.2fs, want %.2fs (reveal + fire + shine)", got, want)
	}
	if got := spec.Name; got != "reveal+fire+shine" {
		t.Fatalf("composed spec is named %q", got)
	}
}
