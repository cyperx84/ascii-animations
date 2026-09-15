// This file is the drop-in claim, checked by the compiler.
//
// It is an external test package on purpose: it may only touch the exported
// surface, so if the compatibility claim needed an unexported helper, or a
// different call shape, this file would not build. Every snippet below is
// written the way charm.land/bubbles/v2/spinner code is written, so a reader can
// diff it against their own program.
package spinner_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cyperx84/ascii-animations/asciifx/spinner"
)

// The shapes upstream's README and examples use, as compile-time assertions.
var (
	_ func(...spinner.Option) spinner.Model  = spinner.New
	_ func(spinner.Spinner) spinner.Option   = spinner.WithSpinner
	_ func(lipgloss.Style) spinner.Option    = spinner.WithStyle
	_ func(tea.Msg) (spinner.Model, tea.Cmd) = spinner.Model{}.Update
	_ func() string                          = spinner.Model{}.View
	_ func() tea.Msg                         = spinner.Model{}.Tick
	_ func() int                             = spinner.Model{}.ID
	_ tea.Msg                                = spinner.TickMsg{}
	_                                        = []spinner.Spinner{
		spinner.Line, spinner.Dot, spinner.MiniDot, spinner.Jump, spinner.Pulse,
		spinner.Points, spinner.Globe, spinner.Moon, spinner.Monkey, spinner.Meter,
		spinner.Hamburger, spinner.Ellipsis,
	}
)

// TestUpstreamProgramShape is the migration in miniature: the parts of a real
// program that would have to change, left unchanged.
func TestUpstreamProgramShape(t *testing.T) {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))),
	)

	// Init returns the method value, not a call: Tick is a Msg-returning
	// method, so m.spinner.Tick is itself a tea.Cmd.
	var cmd tea.Cmd = s.Tick
	if cmd == nil {
		t.Fatal("Tick as a method value is not usable as a command")
	}

	// The message routes back through Update, and the frame advances.
	first := s.View()
	s, next := s.Update(cmd())
	if next == nil {
		t.Fatal("an accepted tick must schedule the next one")
	}
	if s.View() == first {
		t.Fatal("the frame did not advance")
	}

	// Users assign the struct fields directly.
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle()
	if got := s.View(); strings.Contains(got, "⣾") {
		t.Fatalf("assigning Spinner did not take effect: %q", got)
	}

	// Users type-switch on TickMsg and compare IDs.
	if msg, ok := cmd().(spinner.TickMsg); ok {
		if msg.ID != s.ID() {
			t.Fatalf("a tick carries ID %d, the model is %d", msg.ID, s.ID())
		}
		if msg.Time.IsZero() {
			t.Error("TickMsg.Time is not populated")
		}
	} else {
		t.Fatal("Tick did not produce a TickMsg")
	}
}

// TestPredefinedSpinnersMatchUpstream pins the twelve frame sets and intervals.
// A replacement that renders different glyphs, or at a different rate, would
// change the output of a program that never asked to change.
func TestPredefinedSpinnersMatchUpstream(t *testing.T) {
	cases := []struct {
		name   string
		got    spinner.Spinner
		frames []string
		fps    time.Duration
	}{
		{"Line", spinner.Line, []string{"|", "/", "-", "\\"}, time.Second / 10},
		{"Dot", spinner.Dot, []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}, time.Second / 10},
		{"MiniDot", spinner.MiniDot, []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, time.Second / 12},
		{"Jump", spinner.Jump, []string{"⢄", "⢂", "⢁", "⡁", "⡈", "⡐", "⡠"}, time.Second / 10},
		{"Pulse", spinner.Pulse, []string{"█", "▓", "▒", "░"}, time.Second / 8},
		{"Points", spinner.Points, []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"}, time.Second / 7},
		{"Globe", spinner.Globe, []string{"🌍", "🌎", "🌏"}, time.Second / 4},
		{"Moon", spinner.Moon, []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}, time.Second / 8},
		{"Monkey", spinner.Monkey, []string{"🙈", "🙉", "🙊"}, time.Second / 3},
		{"Meter", spinner.Meter, []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▰▰▱", "▰▱▱", "▱▱▱"}, time.Second / 7},
		{"Hamburger", spinner.Hamburger, []string{"☱", "☲", "☴", "☲"}, time.Second / 3},
		{"Ellipsis", spinner.Ellipsis, []string{"", ".", "..", "..."}, time.Second / 3},
	}
	for _, c := range cases {
		if c.got.FPS != c.fps {
			t.Errorf("%s: FPS = %v, want %v", c.name, c.got.FPS, c.fps)
		}
		if len(c.got.Frames) != len(c.frames) {
			t.Errorf("%s: %d frames, want %d", c.name, len(c.got.Frames), len(c.frames))
			continue
		}
		for i, f := range c.frames {
			if c.got.Frames[i] != f {
				t.Errorf("%s frame %d = %q, want %q", c.name, i, c.got.Frames[i], f)
			}
		}
	}
}

// TestNewDefaultsToLine matches upstream's default, which is the cheap path:
// ten frames a second, unstyled, no label.
func TestNewDefaultsToLine(t *testing.T) {
	m := spinner.New()
	if len(m.Spinner.Frames) != len(spinner.Line.Frames) || m.Spinner.FPS != spinner.Line.FPS {
		t.Fatalf("New defaulted to %v at %v, want Line at %v", m.Spinner.Frames, m.Spinner.FPS, spinner.Line.FPS)
	}
	if m.ID() == 0 {
		t.Error("New gave an id of 0, which is upstream's wildcard value")
	}
	// Unstyled means unstyled: the zero Style renders the frame with no escape
	// sequence at all. (GetForeground returns lipgloss.NoColor, not nil, so it
	// is not the thing to assert on.)
	if got, want := m.View(), "|"; got != want {
		t.Errorf("View() = %q, want %q for an unstyled model", got, want)
	}
}

func TestIDsAreUniqueAndGrow(t *testing.T) {
	a, b := spinner.New(), spinner.New()
	if a.ID() == b.ID() {
		t.Fatalf("two spinners share id %d, so they would answer each other's ticks", a.ID())
	}
	if b.ID() <= a.ID() {
		t.Errorf("ids are not increasing: %d then %d", a.ID(), b.ID())
	}
}

// TestViewIsStyledAndErrorsLikeUpstream pins the two output behaviours a golden
// test would notice.
func TestViewIsStyledAndErrorsLikeUpstream(t *testing.T) {
	m := spinner.New(spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))))
	if got := m.View(); !strings.Contains(got, "38;2;255;0;0") {
		t.Errorf("View did not apply the style: %q", got)
	}
	// Upstream returns "(error)" when the frame is out of range, and an empty
	// frame list is out of range at frame 0.
	if got := spinner.New(spinner.WithSpinner(spinner.Spinner{})).View(); got != "(error)" {
		t.Errorf("empty frame set rendered %q, want \"(error)\"", got)
	}
	// Ellipsis has an empty first frame, which legitimately renders nothing.
	if got := spinner.New(spinner.WithSpinner(spinner.Ellipsis)).View(); got != "" {
		t.Errorf("Ellipsis frame 0 rendered %q, want an empty string", got)
	}
}

// TestDuplicatedTickDoesNotDoubleTheRate is the property the tag exists for, seen
// from outside the package.
func TestDuplicatedTickDoesNotDoubleTheRate(t *testing.T) {
	m := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	// Two chains started from the same state, which is what a parent does when
	// it returns Tick from Init and again from somewhere else.
	msg1 := m.Tick().(spinner.TickMsg)
	msg2 := m.Tick().(spinner.TickMsg)
	var (
		cmdA tea.Cmd
		cmdB tea.Cmd
	)
	m, cmdA = m.Update(msg1)
	m, cmdB = m.Update(msg2)
	if cmdA == nil || cmdB == nil {
		t.Fatal("an accepted tick must schedule a follow-up")
	}
	// Both first messages carried the same tag, so both were accepted. From
	// here the follow-ups carry different tags and the older chain is dead:
	// one chain survives, not two.
	if _, cmd := m.Update(cmdA()); cmd != nil {
		t.Error("the superseded chain is still alive, so the rate stays doubled")
	}
	if _, cmd := m.Update(cmdB()); cmd == nil {
		t.Error("the live chain stopped ticking")
	}
}

// TestIDFilteringAndWildcard covers the routing rules a program relies on when
// several spinners share one message loop.
func TestIDFilteringAndWildcard(t *testing.T) {
	a, b := spinner.New(), spinner.New()
	before := a.View()

	if got, cmd := a.Update(spinner.TickMsg{ID: b.ID()}); cmd != nil || got.View() != before {
		t.Error("a foreign ID advanced the spinner")
	}
	// A zero ID is a wildcard, which is how a hand-built message works.
	got, cmd := a.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("a zero-ID message was dropped")
	}
	if got.View() == before {
		t.Error("a zero-ID message did not advance the spinner")
	}
}

func TestExtrasAreOffByDefault(t *testing.T) {
	m := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	for i := 0; i < len(spinner.MiniDot.Frames); i++ {
		v := m.View()
		if strings.Contains(v, "\x1b[") {
			t.Fatalf("frame %d is styled with no palette or shimmer asked for: %q", i, v)
		}
		if strings.ContainsAny(v, "abcdefghijklmnopqrstuvwxyz") {
			t.Fatalf("frame %d carries a label that was never set: %q", i, v)
		}
		var cmd tea.Cmd
		m, cmd = m.Update(m.Tick())
		if cmd == nil {
			t.Fatal("the spinner stopped")
		}
	}
}

func TestWithLabel(t *testing.T) {
	m := spinner.New(spinner.WithSpinner(spinner.Line), spinner.WithLabel("Compiling"))
	got := m.View()
	if !strings.HasSuffix(got, " Compiling") {
		t.Fatalf("View() = %q, want it to end with the label", got)
	}
	// An empty label added on purpose renders nothing extra, unlike no label.
	if got := spinner.New(spinner.WithSpinner(spinner.Line), spinner.WithLabel("")).View(); got != "|" {
		t.Errorf("an empty label rendered %q", got)
	}
}

func TestWithPaletteCyclesTheGlyph(t *testing.T) {
	m := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithPalette("nord"))
	seen := map[string]bool{}
	for i := 0; i < len(spinner.MiniDot.Frames); i++ {
		seen[m.View()] = true
		m, _ = m.Update(m.Tick())
	}
	// The frames differ anyway, so the palette is only proven by the escape
	// sequences being present and not constant.
	if len(seen) < 2 {
		t.Fatalf("the palette produced %d distinct frames", len(seen))
	}
	for v := range seen {
		if !strings.Contains(v, "\x1b[") {
			t.Fatalf("a frame carries no colour: %q", v)
		}
	}
}

func TestWithShimmerHighlightsTheLabel(t *testing.T) {
	unshimmered := spinner.New(spinner.WithSpinner(spinner.Line), spinner.WithLabel("Compiling assets"))
	shimmered := spinner.New(
		spinner.WithSpinner(spinner.Line),
		spinner.WithLabel("Compiling assets"),
		spinner.WithShimmer(1.4),
	)
	if got := unshimmered.View(); strings.Contains(got, "\x1b[") {
		t.Fatalf("a label with no shimmer should carry no colour: %q", got)
	}
	// The highlight moves, so at least two frames differ.
	seen := map[string]bool{}
	colours := map[string]bool{}
	for i := 0; i < 12; i++ {
		v := shimmered.View()
		seen[v] = true
		colours[colourRuns(v)] = true
		// The highlight is per rune, so it interleaves escape sequences with
		// the text; what must survive is the visible label behind the glyph,
		// which advances one frame per tick.
		got := plain(v)
		if !strings.HasSuffix(got, " Compiling assets") {
			t.Fatalf("the shimmer mangled the label: %q", got)
		}
		if glyph := strings.TrimSuffix(got, " Compiling assets"); !isLineFrame(glyph) {
			t.Fatalf("the glyph became %q", glyph)
		}
		shimmered, _ = shimmered.Update(shimmered.Tick())
	}
	if len(seen) < 2 {
		t.Fatalf("the shimmer produced %d distinct frames", len(seen))
	}
	if len(colours) < 2 {
		t.Fatalf("the shimmer moved but the colours did not: %d distinct runs", len(colours))
	}
}

func TestNamedReachesThisProjectsFrameSets(t *testing.T) {
	for _, name := range spinner.StyleNames() {
		s, ok := spinner.Named(name)
		if !ok {
			t.Errorf("StyleNames lists %q but Named does not accept it", name)
			continue
		}
		if len(s.Frames) == 0 || s.FPS <= 0 {
			t.Errorf("%q has %d frames at %v", name, len(s.Frames), s.FPS)
		}
		// A spinner is shown in a TUI, so its frames must not be wide.
		for _, f := range s.Frames {
			if w := lipgloss.Width(f); w > 1 {
				t.Errorf("%q frame %q is %d columns wide", name, f, w)
			}
		}
	}
	if _, ok := spinner.Named("nope"); ok {
		t.Error("Named accepted a name that does not exist")
	}
	// The frame sets that clash with upstream names are still reachable.
	for _, name := range []string{"line", "pulse"} {
		if _, ok := spinner.Named(name); !ok {
			t.Errorf("Named(%q) should reach this project's frame set", name)
		}
	}
}

func TestZeroFPSDoesNotHotLoop(t *testing.T) {
	// A Spinner with no FPS is what a hand-written frame set looks like, and a
	// zero interval would make tea.Tick fire as fast as the scheduler allows.
	m := spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: []string{"a", "b"}}))
	if _, cmd := m.Update(m.Tick()); cmd == nil {
		t.Fatal("a tick was dropped")
	}
}

var sgrRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// plain strips styling, which is what a reader of the label sees.
func plain(s string) string { return sgrRe.ReplaceAllString(s, "") }

// isLineFrame reports whether s is one of the Line spinner's frames, so the
// glyph can be asserted without pinning which frame a tick landed on.
func isLineFrame(s string) bool {
	for _, f := range spinner.Line.Frames {
		if s == f {
			return true
		}
	}
	return false
}

// colourRuns keeps only the styling, so a frame that looks identical but is
// coloured differently still counts as a change.
func colourRuns(s string) string { return strings.Join(sgrRe.FindAllString(s, -1), "|") }
