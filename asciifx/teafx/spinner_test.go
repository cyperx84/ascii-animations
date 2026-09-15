package teafx

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
)

// The drop-in claim, checked by the compiler: the same call shapes a
// bubbles/spinner user already writes.
func TestSpinnerHasTheBubblesShape(t *testing.T) {
	var s Spinner
	var _ func(tea.Msg) (Spinner, tea.Cmd) = s.Update
	var _ func() string = s.View
	var _ tea.Cmd = s.Tick()
}

func TestNewSpinnerRejectsAnUnknownStyle(t *testing.T) {
	_, err := NewSpinner("dotz")
	if err == nil {
		t.Fatal("a typo'd style must fail, not silently spin something else")
	}
	if !strings.Contains(err.Error(), "dotz") {
		t.Fatalf("error should name the bad style: %v", err)
	}
	if !strings.Contains(err.Error(), "dots") {
		t.Fatalf("error should list the valid styles: %v", err)
	}
}

func TestSpinnerStylesMatchTheEffect(t *testing.T) {
	styles := SpinnerStyles()
	if len(styles) < 14 {
		t.Fatalf("only %d styles: %v", len(styles), styles)
	}
	for _, s := range styles {
		if _, err := NewSpinner(s); err != nil {
			t.Errorf("style %q from SpinnerStyles does not build: %v", s, err)
		}
	}
}

// TestSpinnerViewIsExactlyItsContent is the layout-safety property: a widget in
// a larger view must not emit padding the layout did not ask for.
func TestSpinnerViewIsExactlyItsContent(t *testing.T) {
	for _, label := range []string{"", "Compiling", "a much longer label that overflows a default width"} {
		s, err := NewSpinner("dots", WithLabel(label))
		if err != nil {
			t.Fatal(err)
		}
		plain := stripANSI(s.View())
		if strings.TrimRight(plain, " ") != plain {
			t.Errorf("label %q: view has trailing padding: %q", label, plain)
		}
		want := 1
		if label != "" {
			want = len([]rune(label)) + 2
		}
		if got := s.Width(); got != want {
			t.Errorf("label %q: Width() = %d, want %d", label, got, want)
		}
		if got := len([]rune(plain)); got != want {
			t.Errorf("label %q: view is %d columns, want %d (%q)", label, got, want, plain)
		}
	}
}

func TestSpinnerAdvancesOnItsOwnTick(t *testing.T) {
	s, err := NewSpinner("dots", WithLabel("Working"))
	if err != nil {
		t.Fatal(err)
	}
	first := stripANSI(s.View())
	glyphs := map[string]bool{firstGlyph(first): true}
	for i := 0; i < 12; i++ {
		var cmd tea.Cmd
		s, cmd = s.Update(TickMsg{ID: s.m.id})
		if cmd == nil {
			t.Fatalf("tick %d did not schedule the next one; the spinner would stop", i)
		}
		glyphs[firstGlyph(stripANSI(s.View()))] = true
	}
	if len(glyphs) < 2 {
		t.Fatalf("twelve ticks showed one glyph (%v); the animation is stuck", glyphs)
	}
	if got := stripANSI(s.View()); !strings.HasSuffix(got, "Working") {
		t.Fatalf("the label was lost while spinning: %q", got)
	}
}

func TestSpinnerIgnoresAnotherInstancesTick(t *testing.T) {
	a, err := NewSpinner("dots", WithLabel("A"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSpinner("dots", WithLabel("B"))
	if err != nil {
		t.Fatal(err)
	}
	if a.m.id == b.m.id {
		t.Fatal("two spinners share an id, so they would answer each other's ticks")
	}
	// A tick belonging to b must not advance a.
	before := a.View()
	a, _ = a.Update(TickMsg{ID: b.m.id})
	if a.View() != before {
		t.Fatal("a spinner advanced on another instance's tick")
	}
}

// TestSetLabelKeepsSpinning is the bug this API is most likely to have: a
// rebuild that issues a new instance id would drop the tick already in flight
// and freeze the spinner.
func TestSetLabelKeepsSpinning(t *testing.T) {
	s, err := NewSpinner("dots", WithLabel("step 1"))
	if err != nil {
		t.Fatal(err)
	}
	s, _ = s.Update(TickMsg{ID: s.m.id})
	before := s.m.run.Tick()
	id := s.m.id

	s = s.SetLabel("step 2")
	if s.m.id != id {
		t.Fatalf("SetLabel changed the instance id (%d -> %d), which orphans in-flight ticks", id, s.m.id)
	}
	if got := s.m.run.Tick(); got != before {
		t.Fatalf("SetLabel moved the frame from %d to %d", before, got)
	}
	if !strings.Contains(stripANSI(s.View()), "step 2") {
		t.Fatalf("label not applied: %q", stripANSI(s.View()))
	}
	// The tick in flight from before the label change must still advance it.
	s, cmd := s.Update(TickMsg{ID: id})
	if cmd == nil {
		t.Fatal("the spinner froze after SetLabel")
	}
	if s.m.run.Tick() <= before {
		t.Fatal("the spinner did not advance after SetLabel")
	}
}

func TestSetLabelKeepsEveryOtherOption(t *testing.T) {
	s, err := NewSpinner("pipe", WithLabel("a"), WithPalette("nord"), WithSpeed(2), WithShimmer(0.5), WithFPS(20))
	if err != nil {
		t.Fatal(err)
	}
	s = s.SetLabel("b")
	// Read the params back off the built run, which is what the effect sees.
	got := s.m.run.Values.Map()
	for _, want := range []struct{ key, val string }{
		{"style", "pipe"}, {"label", "b"}, {"palette", "nord"}, {"speed", "2"}, {"shimmer", "0.5"},
	} {
		if got[want.key] != want.val {
			t.Errorf("after SetLabel, %s = %q, want %q", want.key, got[want.key], want.val)
		}
	}
	if s.m.run.FPS() != 20 {
		t.Errorf("after SetLabel, fps = %d, want 20", s.m.run.FPS())
	}
	if s.Style() != "pipe" {
		t.Errorf("SetLabel changed the style to %q", s.Style())
	}
}

func TestSetStyleKeepsTheFrame(t *testing.T) {
	s, err := NewSpinner("dots", WithLabel("x"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		s, _ = s.Update(TickMsg{ID: s.m.id})
	}
	tick := s.m.run.Tick()
	s = s.SetStyle("line")
	if s.Style() != "line" {
		t.Fatalf("style is %q", s.Style())
	}
	if s.m.run.Tick() != tick {
		t.Fatalf("SetStyle moved the frame from %d to %d", tick, s.m.run.Tick())
	}
	// An unknown style leaves the spinner alone rather than blanking it.
	if got := s.SetStyle("nope"); got.Style() != "line" {
		t.Fatalf("an unknown style changed the spinner to %q", got.Style())
	}
}

func TestSpinnerWithNoLabelIsJustTheGlyph(t *testing.T) {
	s, err := NewSpinner("line", WithShimmer(0))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Width(); got != 1 {
		t.Fatalf("Width() = %d, want 1", got)
	}
	if got := stripANSI(s.View()); !strings.ContainsAny(got, `|/\-`) {
		t.Fatalf("view %q is not a line spinner frame", got)
	}
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func firstGlyph(s string) string {
	for _, r := range s {
		if r != ' ' {
			return string(r)
		}
	}
	return ""
}
