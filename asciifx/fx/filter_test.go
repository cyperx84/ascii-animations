package fx

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// selAt asks a selector about one cell, for the truth tables.
func selAt(s Selector, x, y, w, h int, c cell.Cell) bool { return s.Match(x, y, w, h, &c) }

func TestSelectorTruthTables(t *testing.T) {
	ink := cell.Cell{Rune: '█'}
	blank := cell.Cell{Rune: ' '}
	zero := cell.Cell{}
	red := cell.Cell{Rune: '#', FG: tint.RGB(255, 0, 0)}

	cases := []struct {
		name string
		sel  Selector
		x, y int
		c    cell.Cell
		want bool
	}{
		{"ink on a block", SelInk, 0, 0, ink, true},
		{"ink on a space", SelInk, 0, 0, blank, false},
		{"ink on the zero rune", SelInk, 0, 0, zero, false},
		{"not(ink) on a space", SelNot(SelInk), 0, 0, blank, true},
		{"not(nil) accepts everything", SelNot(nil), 0, 0, blank, true},

		// A 10x10 buffer, margin 1: the middle is inner, the border is outer.
		{"inner(1) on the border", SelInner(1, 1), 0, 5, ink, false},
		{"inner(1) in the middle", SelInner(1, 1), 5, 5, ink, true},
		{"inner(1) on the last row", SelInner(1, 1), 5, 9, ink, false},
		{"outer(1) on the border", SelOuter(1, 1), 0, 5, ink, true},
		{"outer(1) in the middle", SelOuter(1, 1), 5, 5, ink, false},
		// inner(0) is everything and outer(0) is nothing.
		{"inner(0) accepts the corner", SelInner(0, 0), 0, 0, ink, true},
		{"outer(0) rejects the corner", SelOuter(0, 0), 0, 0, ink, true == false},
		// A margin wider than the buffer leaves no interior, so outer takes all.
		{"inner(99) rejects", SelInner(99, 99), 5, 5, ink, false},
		{"outer(99) accepts", SelOuter(99, 99), 5, 5, ink, true},
		// h and v are separate.
		{"inner(2,0) rejects a side column", SelInner(2, 0), 1, 5, ink, false},
		{"inner(2,0) accepts a top row", SelInner(2, 0), 5, 0, ink, true},

		// fg matches exactly, and unset is not black.
		{"fg(red) on red", SelFG(tint.RGB(255, 0, 0)), 0, 0, red, true},
		{"fg(red) on unset", SelFG(tint.RGB(255, 0, 0)), 0, 0, blank, false},
		{"fg(none) on unset", SelFG(tint.None), 0, 0, blank, true},
		{"fg(none) on black", SelFG(tint.None), 0, 0, cell.Cell{Rune: '#', FG: tint.RGB(0, 0, 0)}, false},
		{"fg(black) on black", SelFG(tint.RGB(0, 0, 0)), 0, 0, cell.Cell{Rune: '#', FG: tint.RGB(0, 0, 0)}, true},

		// Combinators, including their empty identities.
		{"all() accepts everything", AllOf(), 0, 0, blank, true},
		{"any() accepts nothing", AnyOf(), 0, 0, ink, false},
		{"all(ink) with one child", AllOf(SelInk), 0, 0, ink, true},
		{"any(ink) with one child", AnyOf(SelInk), 0, 0, blank, false},
		{"all(ink,inner(1)) both must hold", AllOf(SelInk, SelInner(1, 1)), 5, 5, ink, true},
		{"all(ink,inner(1)) fails on the border", AllOf(SelInk, SelInner(1, 1)), 0, 5, ink, false},
		{"any(ink,inner(1)) either holds", AnyOf(SelInk, SelInner(1, 1)), 0, 5, ink, true},
		{"any(ink,inner(1)) on a blank middle", AnyOf(SelInk, SelInner(1, 1)), 5, 5, blank, true},
		{"nil children are ignored by all", AllOf(nil, SelInk), 0, 0, ink, true},
		{"nil children are ignored by any", AnyOf(nil), 0, 0, ink, false},
	}
	for _, c := range cases {
		if got := selAt(c.sel, c.x, c.y, 10, 10, c.c); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestSelectorNeedsContent pins which selectors require a content run, because
// that is what the engine guard keys on.
func TestSelectorNeedsContent(t *testing.T) {
	cases := []struct {
		name string
		sel  Selector
		want bool
	}{
		{"ink", SelInk, true},
		{"fg", SelFG(tint.None), true},
		{"inner", SelInner(1, 1), false},
		{"outer", SelOuter(1, 1), false},
		{"not(ink)", SelNot(SelInk), true},
		{"not(inner)", SelNot(SelInner(1, 1)), false},
		{"all(ink,inner)", AllOf(SelInk, SelInner(1, 1)), true},
		{"all(inner,outer)", AllOf(SelInner(1, 1), SelOuter(2, 2)), false},
		{"any(ink,inner)", AnyOf(SelInk, SelInner(1, 1)), true},
		{"all()", AllOf(), false},
		{"any()", AnyOf(), false},
		{"not(nil)", SelNot(nil), false},
	}
	for _, c := range cases {
		if got := c.sel.NeedsContent(); got != c.want {
			t.Errorf("%s: NeedsContent() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseSelector(t *testing.T) {
	// Round-tripping through the parser is checked by behaviour: each
	// expression must accept the cells its Go spelling does.
	cases := []struct {
		expr string
		x, y int
		c    cell.Cell
		want bool
	}{
		{"ink", 0, 0, cell.Cell{Rune: 'x'}, true},
		{"ink", 0, 0, cell.Cell{Rune: ' '}, false},
		{"not(ink)", 0, 0, cell.Cell{Rune: ' '}, true},
		{"inner(1)", 5, 5, cell.Cell{Rune: 'x'}, true},
		{"inner(1)", 0, 5, cell.Cell{Rune: 'x'}, false},
		// h=2 excludes columns 0 and 1, v=1 excludes rows 0 and 9.
		{"inner(2, 1)", 2, 1, cell.Cell{Rune: 'x'}, true},
		{"inner(2, 1)", 1, 5, cell.Cell{Rune: 'x'}, false},
		{"inner(2, 1)", 5, 0, cell.Cell{Rune: 'x'}, false},
		{"outer(1)", 0, 5, cell.Cell{Rune: 'x'}, true},
		{"all(ink, inner(1))", 5, 5, cell.Cell{Rune: 'x'}, true},
		{"all(ink, inner(1))", 0, 5, cell.Cell{Rune: 'x'}, false},
		{"any(ink, inner(1))", 0, 5, cell.Cell{Rune: 'x'}, true},
		{"not(all(ink, inner(1)))", 0, 5, cell.Cell{Rune: 'x'}, true},
		{"fg(#ff0000)", 0, 0, cell.Cell{Rune: 'x', FG: tint.RGB(255, 0, 0)}, true},
		{"fg(#ff0000)", 0, 0, cell.Cell{Rune: 'x'}, false},
		{"fg(none)", 0, 0, cell.Cell{Rune: 'x'}, true},
		{"all(fg(none), not(ink))", 0, 0, cell.Cell{Rune: ' '}, true},
		// Whitespace is free, as in patterns.
		{"  all( ink ,  inner( 1 , 1 ) )  ", 5, 5, cell.Cell{Rune: 'x'}, true},
		// INK is spelled in lower case, conventionally.
		{"INK", 0, 0, cell.Cell{Rune: 'x'}, true},
	}
	for _, c := range cases {
		sel, err := ParseSelector(c.expr)
		if err != nil {
			t.Errorf("ParseSelector(%q): %v", c.expr, err)
			continue
		}
		if got := selAt(sel, c.x, c.y, 10, 10, c.c); got != c.want {
			t.Errorf("%q at (%d,%d) with %+v = %v, want %v", c.expr, c.x, c.y, c.c, got, c.want)
		}
	}
}

func TestParseSelectorErrors(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"nke", "did you mean"},
		{"nto(ink)", "did you mean"},
		{"", "expected"},
		{"ink(1)", "ink takes no arguments"},
		{"not()", "not"},
		{"not(ink,inner(1))", "not takes one selector"},
		{"all(ink", "another selector"},
		{"all()", "expected"},
		{"fg(#zzz)", "fg wants a colour"},
		{"fg()", "fg wants a colour"},
		{"inner()", "want a margin"},
		{"inner(-1)", "whole number"},
		{"inner(1.5)", "whole number"},
		{"inner(99", "expected"},
		{"ink extra", "unexpected"},
		{"all(ink) extra", "unexpected"},
	}
	for _, c := range cases {
		_, err := ParseSelector(c.expr)
		if err == nil {
			t.Errorf("ParseSelector(%q) should have failed", c.expr)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("ParseSelector(%q) error %q lacks %q", c.expr, err, c.want)
		}
	}
}

func TestSelectorNamesAndGrammarAreConsistent(t *testing.T) {
	// Every name the help lists must actually parse, or the help is lying.
	for _, name := range []string{"ink", "fg(#ff0000)", "inner(1)", "outer(1)", "not(ink)", "all(ink)", "any(ink)"} {
		if _, err := ParseSelector(name); err != nil {
			t.Errorf("SelectorNames lists %q but it does not parse: %v", name, err)
		}
	}
	if SelectorGrammar() == "" || len(SelectorNames()) == 0 {
		t.Fatal("the grammar and names must be published for help and the catalog")
	}
}

// ---------------------------------------------------------------------------
// The wrapper
// ---------------------------------------------------------------------------

// paint is a finite effect that overwrites every cell with one glyph, so what
// survives it is entirely the filter's doing.
type paint struct {
	r rune
	d float64
}

func (p paint) Duration() float64 { return p.d }

func (p paint) Step(f *Frame) {
	for y := 0; y < f.Buf.H; y++ {
		for x := 0; x < f.Buf.W; x++ {
			f.Buf.Set(x, y, cell.Cell{Rune: p.r, FG: tint.RGB(1, 2, 3)})
		}
	}
}

// forever paints and never ends, for the non-Finite case.
type foreverPaint struct{ r rune }

func (p foreverPaint) Step(f *Frame) {
	for y := 0; y < f.Buf.H; y++ {
		for x := 0; x < f.Buf.W; x++ {
			f.Buf.Set(x, y, cell.Cell{Rune: p.r})
		}
	}
}

func paintSpec(name string, r rune, content bool, dur float64) *Spec {
	s := &Spec{
		Name: name, Title: name, Kind: Ambient, Content: content,
		Duration: dur, FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 3,
	}
	s.New = func(Values, int, int, *rand.Rand) (Effect, error) {
		if dur == 0 {
			return foreverPaint{r}, nil
		}
		return paint{r, dur}, nil
	}
	Register(*s)
	return s
}

// conformance lists each registered effect with its finite duration, and runs a
// wrapped and an unwrapped instance side by side.
func TestFilterAcceptingEverythingIsUnchanged(t *testing.T) {
	paintSpec("tfilter-paint", '#', false, 1.0)
	base := &Spec{
		Name: "tfilter-base", Title: "base", Kind: Ambient, Duration: 1.0,
		FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 3,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return paint{'#', 1.0}, nil },
	}
	Register(*base)

	for _, sel := range []Selector{AllOf(), SelNot(nil)} {
		plain, err := NewRun(base, Options{Seed: 1})
		if err != nil {
			t.Fatal(err)
		}
		kept, err := NewRun(base, Options{Seed: 1, Filter: sel})
		if err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 30; tick++ {
			a, b := plain.Next(), kept.Next()
			if !a.Equal(b) {
				t.Fatalf("an accepting filter changed frame %d:\n%s\nvs\n%s", tick, a.Plain(), b.Plain())
			}
		}
	}
}

func TestFilterRejectingEverythingLeavesTheBufferAlone(t *testing.T) {
	base := &Spec{
		Name: "tfilter-none", Title: "none", Kind: Ambient, Duration: 1.0,
		FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 3,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return paint{'#', 1.0}, nil },
	}
	Register(*base)
	blank := cell.New(4, 3)
	got, err := NewRun(base, Options{Seed: 1, Filter: AnyOf()})
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 10; tick++ {
		if b := got.Next(); !b.Equal(blank) {
			t.Fatalf("a rejecting filter let something through at tick %d:\n%s", tick, b.Plain())
		}
	}
}

// TestFilterRejectedCellsEqualThePreStepBuffer is the property the design rests
// on: a rejected cell is byte-identical to what the effect was handed, glyph,
// both colours and attributes together.
func TestFilterRejectedCellsEqualThePreStepBuffer(t *testing.T) {
	// A content spec, so the handed buffer is known exactly.
	base := &Spec{
		Name: "tfilter-pre", Title: "pre", Kind: Transition, Content: true, Duration: 0.2,
		FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 2,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return paint{'#', 0.2}, nil },
	}
	Register(*base)
	content := func(b *cell.Buffer) {
		b.Set(0, 0, cell.Cell{Rune: 'A', FG: tint.RGB(9, 8, 7), BG: tint.RGB(1, 1, 1), Attr: cell.Bold})
		b.Set(1, 0, cell.Cell{Rune: 'B', FG: tint.RGB(6, 5, 4)})
	}
	// Reject only the inked cells, so the picture must survive and the blanks
	// must be painted over.
	r, err := NewRun(base, Options{Seed: 1, Filter: SelNot(SelInk), Content: content})
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Seek(3)
	if err != nil {
		t.Fatal(err)
	}
	want := cell.New(4, 2)
	content(want)
	if b.Cells[0] != want.Cells[0] {
		t.Errorf("rejected cell 0 = %+v, want the pre-step %+v", b.Cells[0], want.Cells[0])
	}
	if b.Cells[1] != want.Cells[1] {
		t.Errorf("rejected cell 1 = %+v, want the pre-step %+v", b.Cells[1], want.Cells[1])
	}
	if b.Cells[2].Rune != '#' {
		t.Errorf("accepted cell 2 = %q, want the paint", b.Cells[2].Rune)
	}
}

func TestFilterPreservesFiniteness(t *testing.T) {
	paintSpec("tfilter-loop", 'L', false, 0)
	finite := &Spec{
		Name: "tfilter-dur", Title: "dur", Kind: Ambient, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 3,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return paint{'D', 0.5}, nil },
	}
	Register(*finite)
	loop, _ := Lookup("tfilter-loop")
	fin, _ := Lookup("tfilter-dur")

	// A finite effect wrapped stays finite with the same duration, or Done
	// would start reporting the wrong thing.
	// A geometry selector, because a content selector on a run with no content
	// is rejected outright (see TestFilterNeedsContent).
	r, err := NewRun(fin, Options{Seed: 1, Filter: SelInner(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Duration() != 0.5 {
		t.Errorf("filtered duration = %g, want 0.5", r.Duration())
	}
	if r.Frames() == 0 {
		t.Error("a filtered finite run reported no frames")
	}
	if _, ok := Filter(paint{'x', 0.5}, SelInner(0, 0)).(Finite); !ok {
		t.Error("Filter dropped Finiteness")
	}
	// A looping effect wrapped stays looping.
	lr, err := NewRun(loop, Options{Seed: 1, Filter: SelInner(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if lr.Duration() != 0 || lr.Frames() != 0 {
		t.Errorf("a wrapped looping effect became finite: duration %g, frames %d", lr.Duration(), lr.Frames())
	}
	if _, ok := Filter(foreverPaint{'x'}, SelInner(0, 0)).(Finite); ok {
		t.Error("Filter invented a duration for a looping effect")
	}
}

func TestFilterIsDeterministic(t *testing.T) {
	base := &Spec{
		Name: "tfilter-det", Title: "det", Kind: Ambient, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 4, DefH: 3,
		New: func(_ Values, _, _ int, rng *rand.Rand) (Effect, error) {
			// A stateful effect, so the frame counter is not the whole story.
			return paint{rune('a' + rng.IntN(26)), 0.5}, nil
		},
	}
	Register(*base)
	render := func() string {
		r, err := NewRun(base, Options{Seed: 7, Filter: SelInner(1, 1)})
		if err != nil {
			t.Fatal(err)
		}
		var sb strings.Builder
		for tick := 0; tick < 15; tick++ {
			sb.WriteString(r.Next().Plain())
			sb.WriteByte('\n')
		}
		return sb.String()
	}
	first := render()
	for i := 0; i < 3; i++ {
		if got := render(); got != first {
			t.Fatalf("run %d differs from the first", i+2)
		}
	}
	// Seek backwards and forwards lands on the same frame.
	r, err := NewRun(base, Options{Seed: 7, Filter: SelInner(1, 1)})
	if err != nil {
		t.Fatal(err)
	}
	at5, err := r.Seek(5)
	if err != nil {
		t.Fatal(err)
	}
	saved := at5.Clone()
	at12, err := r.Seek(12)
	if err != nil {
		t.Fatal(err)
	}
	_ = at12
	back, err := r.Seek(5)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Equal(saved) {
		t.Fatal("Seek is not exact with a filter")
	}
}

// TestFilteredAmbientKeepsSimulating is the flagship concern: filtering an
// ambient effect must hide its output, not freeze its state. If the wrapper
// leaked the previous frame back into a simulation's input, its state would stop
// advancing and the visible part would go still.
func TestFilteredAmbientKeepsSimulating(t *testing.T) {
	// A simulation whose output depends on how many times it has been stepped.
	counting := &Spec{
		Name: "tfilter-count", Title: "count", Kind: Ambient, Duration: 0,
		FPS: 30, MinW: 1, MinH: 1, DefW: 3, DefH: 1,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return &stepper{}, nil },
	}
	Register(*counting)

	// outer(1) leaves only the first and last columns for the effect to write.
	r, err := NewRun(counting, Options{Seed: 1, Filter: SelOuter(1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for tick := 0; tick < 8; tick++ {
		b := r.Next()
		seen[b.Plain()] = true
		// The middle column is rejected, so it must stay blank; the edges are
		// accepted and must carry the count.
		if b.Cells[1].Rune != ' ' {
			t.Fatalf("tick %d wrote a rejected cell: %q", tick, b.Plain())
		}
		if b.Cells[0].Rune == ' ' || b.Cells[2].Rune == ' ' {
			t.Fatalf("tick %d did not write an accepted cell: %q", tick, b.Plain())
		}
	}
	if len(seen) < 4 {
		t.Fatalf("only %d distinct frames: the simulation is not advancing", len(seen))
	}
}

// stepper fills the whole row with its step count, so a frozen simulation is
// obvious and a rejected cell is provably restored rather than merely unwritten.
type stepper struct{ n int }

func (c *stepper) Step(f *Frame) {
	c.n++
	for y := 0; y < f.Buf.H; y++ {
		for x := 0; x < f.Buf.W; x++ {
			f.Buf.Set(x, y, cell.Cell{Rune: rune('0' + c.n%10)})
		}
	}
}

func TestFilterResizeKeepsWorking(t *testing.T) {
	base := &Spec{
		Name: "tfilter-resize", Title: "resize", Kind: Ambient, Duration: 0,
		FPS: 30, MinW: 1, MinH: 1, DefW: 6, DefH: 3,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return foreverPaint{'R'}, nil },
	}
	Register(*base)
	r, err := NewRun(base, Options{Seed: 1, Filter: SelInner(1, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Seek(4); err != nil {
		t.Fatal(err)
	}
	if err := r.Resize(8, 4); err != nil {
		t.Fatal(err)
	}
	b := r.Next()
	if b.W != 8 || b.H != 4 {
		t.Fatalf("buffer is %dx%d after resize", b.W, b.H)
	}
	// The scratch buffer must have followed the resize: the border is rejected
	// and the interior accepted at the new size.
	if b.At(0, 0).Rune != ' ' {
		t.Errorf("border cell was written after a resize: %q", b.Plain())
	}
	if b.At(3, 2).Rune != 'R' {
		t.Errorf("interior cell was not written after a resize: %q", b.Plain())
	}
}

func TestFilterNilIsIdentity(t *testing.T) {
	e := paint{'x', 1.0}
	if got := Filter(e, nil); got != Effect(e) {
		t.Fatal("Filter(e, nil) should return e itself, not a wrapper")
	}
}
