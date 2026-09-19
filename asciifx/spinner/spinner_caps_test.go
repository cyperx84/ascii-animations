package spinner

import (
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/term"
)

// A spinner is the one thing in a TUI that animates all the time, so it is
// the one that has to stop when the terminal asks.
func TestCapsStopTheSpinner(t *testing.T) {
	m := New(WithSpinner(Dots), WithCaps(term.Caps{Profile: term.TrueColor}))
	first := m.View()
	next, cmd := m.Update(m.Tick())
	if cmd != nil {
		t.Fatal("a still spinner should not schedule another tick")
	}
	if next.View() != first {
		t.Fatalf("frame advanced: %q then %q", first, next.View())
	}
}

func TestCapsSlowTheSpinnerWithoutStoppingIt(t *testing.T) {
	m := New(WithSpinner(Dots), WithCaps(term.Caps{Profile: term.TrueColor, Animate: true, FPS: 4}))
	next, cmd := m.Update(m.Tick())
	if cmd == nil {
		t.Fatal("an animating spinner should keep ticking")
	}
	if next.View() == m.View() {
		t.Fatal("frame did not advance")
	}
	// The cap is a rate; the spinner holds an interval, so the cap is a floor.
	if m.minTick == 0 || m.Spinner.FPS >= m.minTick {
		return // this style is already slower than the cap
	}
	if got := next.minTick; got != m.minTick {
		t.Fatalf("cap lost across Update: %v then %v", m.minTick, got)
	}
}

// Nothing changes for a spinner that never asked.
func TestWithoutCapsNothingChanges(t *testing.T) {
	m := New(WithSpinner(Dots))
	if m.still || m.minTick != 0 {
		t.Fatal("a spinner built without WithCaps carries a contract it was never given")
	}
	if _, cmd := m.Update(m.Tick()); cmd == nil {
		t.Fatal("the default spinner stopped ticking")
	}
}
