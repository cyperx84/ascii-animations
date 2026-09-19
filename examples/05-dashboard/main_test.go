package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/cyperx84/ascii-animations/asciifx/lint"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func animated() term.Caps { return term.Caps{Profile: term.TrueColor, Animate: true} }

// A panel that leaves stale cells behind is worse than no panel, so the two
// things this program draws into fixed rectangles are linted with the
// project's own linter: equal-width rows, single-width glyphs.
//
// The whole canvas is deliberately not linted. lipgloss trims the trailing
// blanks off each rendered line, so a sparse layout is ragged by
// construction — that is the renderer's business, not the layout's.
func TestPanelsAreTerminalSafe(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	for _, is := range lint.String(stripANSI(m.tableString())) {
		t.Errorf("node table %d:%d: %s", is.Line, is.Col, is.Message)
	}
	run := m.ambient.Run()
	for _, tick := range []int{0, 30, 60} {
		b, err := run.Seek(tick)
		if err != nil {
			t.Fatal(err)
		}
		for _, is := range lint.String(b.Plain()) {
			t.Errorf("ambient tick %d %d:%d: %s", tick, is.Line, is.Col, is.Message)
		}
	}
}

// The table is the panel a transition snapshots, so every row has to be the
// full panel width: a short row would hand the effect blanks.
func TestTableFillsThePanel(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(stripANSI(m.tableString()), "\n") {
		if got := len([]rune(line)); got != tableW {
			t.Errorf("row %d is %d columns, panel is %d", i, got, tableW)
		}
	}
}

// The refresh path is the interesting one: the table is drawn, snapshotted
// and handed to a transition, so the frame it resolves to has to be the table
// itself.
func TestRefreshResolvesToTheNewTable(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	if cmd := m.refresh(); cmd == nil {
		t.Fatal("refresh started no transition")
	}
	if m.reveal == nil {
		t.Fatal("refresh left no transition to draw")
	}
	run := m.reveal.Run()
	last, err := run.Seek(run.Frames() - 1)
	if err != nil {
		t.Fatal(err)
	}
	want := stripANSI(m.tableString())
	for _, line := range strings.Split(want, "\n") {
		if line = strings.TrimRight(line, " "); line != "" && !strings.Contains(last.Plain(), line) {
			t.Fatalf("the finished transition is missing a table row\n row: %q\nframe:\n%s", line, last.Plain())
		}
	}
}

// Fake data is derived from fx.Hash01, so two runs of the program show the
// same fleet — which is what makes a recording reproducible.
func TestFleetIsDeterministic(t *testing.T) {
	a, b := fakeNodes(3), fakeNodes(3)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("node %d differs between runs: %+v vs %+v", i, a[i], b[i])
		}
	}
	if fakeNodes(4)[0] == a[0] {
		t.Fatal("a new generation produced identical data")
	}
}

func TestResizeMovesTheEffectsWithTheLayout(t *testing.T) {
	m, err := newModel(animated())
	if err != nil {
		t.Fatal(err)
	}
	wide, _ := m.Update(windowSize(120, 40))
	got := wide.(model)
	area := got.ambientRect()
	if w, h := got.ambient.Run().Size(); w != area.Dx() || h != area.Dy() {
		t.Fatalf("ambient is %dx%d, panel is %dx%d", w, h, area.Dx(), area.Dy())
	}
	// Below the minimum the layout clamps rather than collapsing.
	small, _ := got.Update(windowSize(20, 5))
	if s := small.(model); s.w < minW || s.h < minH {
		t.Fatalf("layout collapsed to %dx%d", s.w, s.h)
	}
}

func windowSize(w, h int) tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: w, Height: h} }

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
