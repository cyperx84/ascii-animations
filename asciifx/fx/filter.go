package fx

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// Selector decides which cells an effect is allowed to write.
//
// Match is asked about every cell, and is handed the cell as it was *before*
// the effect ran — the same pre-write check tachyonfx's filtered cell iterator
// makes. That ordering is what makes a selector compose with an effect rather
// than fight it: a selector that rejects a cell leaves that cell exactly as the
// effect found it, and the effect has no idea it was filtered.
//
// x, y, w and h are the whole buffer's geometry. asciifx has no sub-areas, so
// geometry selectors are relative to the buffer.
type Selector interface {
	Match(x, y, w, h int, c *cell.Cell) bool
	// NeedsContent reports whether Match reads the cell. Geometry selectors do
	// not; ink and fg do. A run without content has nothing for a
	// content-reading selector to read — it would see the previous frame, and
	// a blank one at tick zero — so NewRun and Compose reject that combination
	// rather than render something confusing.
	NeedsContent() bool
}

// selFunc adapts a function to a Selector, carrying the one piece of metadata
// the interface needs.
type selFunc struct {
	needsContent bool
	fn           func(x, y, w, h int, c *cell.Cell) bool
}

func (s selFunc) Match(x, y, w, h int, c *cell.Cell) bool { return s.fn(x, y, w, h, c) }

func (s selFunc) NeedsContent() bool { return s.needsContent }

// SelFunc adapts a function to a Selector. needsContent says whether fn reads
// the cell; pass false only for a function that depends on position alone, or
// the run it is used on will be required to have content it does not need.
func SelFunc(needsContent bool, fn func(x, y, w, h int, c *cell.Cell) bool) Selector {
	return selFunc{needsContent: needsContent, fn: fn}
}

// SelInk selects cells the effect can draw on: anything with a visible glyph.
// It is fx.Ink as a selector, and matches tachyonfx's NonEmpty.
//
// Not to be confused with tachyonfx's Text, which matches alphabetic, numeric
// and punctuation characters and spaces but *not* box drawing or block glyphs —
// that would select almost all of a banner's blank field and none of its
// blocks, which is the opposite of what this is for.
var SelInk Selector = selFunc{needsContent: true, fn: func(_, _, _, _ int, c *cell.Cell) bool {
	return c != nil && c.Rune != ' ' && c.Rune != 0
}}

// SelNot inverts a selector. A nil selector matches nothing, so not(nil)
// matches everything, which is the same as no filter at all.
func SelNot(s Selector) Selector {
	return selFunc{needsContent: s != nil && s.NeedsContent(), fn: func(x, y, w, h int, c *cell.Cell) bool {
		return s == nil || !s.Match(x, y, w, h, c)
	}}
}

// AllOf selects a cell only when every selector accepts it. With no children it
// accepts everything, which is what makes it the identity for "and".
func AllOf(ss ...Selector) Selector {
	kids := nonNil(ss)
	needs := anyNeedsContent(kids)
	return selFunc{needsContent: needs, fn: func(x, y, w, h int, c *cell.Cell) bool {
		for _, s := range kids {
			if !s.Match(x, y, w, h, c) {
				return false
			}
		}
		return true
	}}
}

// AnyOf selects a cell when at least one selector accepts it. With no children
// it accepts nothing, which is what makes it the identity for "or".
func AnyOf(ss ...Selector) Selector {
	kids := nonNil(ss)
	needs := anyNeedsContent(kids)
	return selFunc{needsContent: needs, fn: func(x, y, w, h int, c *cell.Cell) bool {
		for _, s := range kids {
			if s.Match(x, y, w, h, c) {
				return true
			}
		}
		return false
	}}
}

// SelInner selects cells inside a margin: h columns are excluded on the left and
// right, v rows on the top and bottom. Inner(0, 0) is the whole buffer.
//
// A margin of a single value is easier to write as SelInner(n, n); the parser
// accepts inner(n) for that.
func SelInner(h, v int) Selector {
	h, v = max(h, 0), max(v, 0)
	return selFunc{fn: func(x, y, w, hh int, _ *cell.Cell) bool {
		return x >= h && x < w-h && y >= v && y < hh-v
	}}
}

// SelOuter selects cells outside a margin — the complement of SelInner, and the
// selector for "draw the frame but leave the picture alone". Outer(0, 0) selects
// nothing.
func SelOuter(h, v int) Selector {
	return SelNot(SelInner(h, v))
}

// SelFG selects cells whose foreground is exactly c, so SelFG(tint.None)
// selects cells with no colour of their own and SelFG(tint.RGB(0, 0, 0))
// selects black ones. The two are different, which is the point: an unset
// colour means the terminal's own, not black.
func SelFG(c tint.Color) Selector {
	return selFunc{needsContent: true, fn: func(_, _, _, _ int, got *cell.Cell) bool {
		return got != nil && got.FG == c
	}}
}

// nonNil drops nil children, so a nil selector contributes nothing to a
// combinator rather than silently meaning "reject everything".
func nonNil(ss []Selector) []Selector {
	out := make([]Selector, 0, len(ss))
	for _, s := range ss {
		if s != nil {
			out = append(out, s)
		}
	}
	return out
}

func anyNeedsContent(ss []Selector) bool {
	for _, s := range ss {
		if s.NeedsContent() {
			return true
		}
	}
	return false
}

// ErrSelectorNeedsContent reports a selector that reads cell contents used on a
// run with no content. Callers that present usage errors can recognise it, so
// the CLI can explain the fix instead of printing a bare failure.
var ErrSelectorNeedsContent = errors.New("this selector reads cell contents, but the run has no content for it to read: it would select against the previous frame, which is blank on the first tick. Add a transition step such as reveal, or use a geometry selector such as inner or outer")

// checkNeedsContent rejects a selector that reads cells on a run that has none
// for it to read.
//
// The failure it prevents is quiet rather than loud: a content-reading selector
// on a run with no content is handed whatever the previous frame left there,
// which is blanks on the first tick, so the effect would appear to do nothing
// and then select against stale output. Both are worse than refusing to run.
func checkNeedsContent(s Selector, hasContent bool) error {
	if s == nil || !s.NeedsContent() || hasContent {
		return nil
	}
	return ErrSelectorNeedsContent
}

// Filter restricts which cells e may change. After e runs, every cell the
// selector rejected is restored to what it was before.
//
// The wrapper knows nothing about the effect, so it works on any of them and no
// effect can forget to honour it. It costs one buffer copy per frame plus one
// predicate call per cell, and it draws no random numbers, so a filtered run is
// as reproducible as an unfiltered one. Filter(e, nil) returns e untouched.
//
// Two properties follow from restricting writes rather than inputs, and both
// are worth knowing:
//
//   - The effect still computes over the whole buffer. A filtered reveal sweeps
//     across the full extent and is then clipped, so its timing does not change.
//   - An ambient effect keeps its own state. Filtering fire hides flames but
//     does not freeze them: the heat buffer advances every tick regardless.
func Filter(e Effect, s Selector) Effect {
	if s == nil {
		return e
	}
	f := filterEffect{e: e, sel: s}
	if fin, ok := e.(Finite); ok {
		return &filteredFinite{filterEffect: f, fin: fin}
	}
	return &filtered{filterEffect: f}
}

// filterEffect is the shared state of both wrappers: the child, the selector,
// and one scratch buffer so filtering a frame does not allocate.
type filterEffect struct {
	e   Effect
	sel Selector
	pre *cell.Buffer
	// w, h remembered so the scratch buffer is only reallocated on a resize.
	w, h int
}

func (f *filterEffect) Step(fr *Frame) {
	if f.pre == nil || f.w != fr.Buf.W || f.h != fr.Buf.H {
		f.pre = cell.New(fr.Buf.W, fr.Buf.H)
		f.w, f.h = fr.Buf.W, fr.Buf.H
	}
	f.pre.CopyFrom(fr.Buf)
	f.e.Step(fr)
	restore(fr.Buf, f.pre, f.sel)
}

// restore puts back every cell the selector rejected. Cells are compared whole,
// so a rejected cell keeps its glyph, both colours and its attributes.
func restore(b, pre *cell.Buffer, s Selector) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			i := y*b.W + x
			if !s.Match(x, y, b.W, b.H, &pre.Cells[i]) {
				b.Cells[i] = pre.Cells[i]
			}
		}
	}
}

// filtered is the wrapper for an effect with no natural end.
type filtered struct{ filterEffect }

func (f *filtered) Step(fr *Frame) { f.filterEffect.Step(fr) }

// filteredFinite keeps Duration, so Done keeps working. Wrapping must never turn
// a finite effect into a looping one or the reverse: Sequence, Compose and
// Done all type-assert Finite, and getting it wrong makes an effect that ends
// report that it never does.
type filteredFinite struct {
	filterEffect
	fin Finite
}

func (f *filteredFinite) Step(fr *Frame) { f.filterEffect.Step(fr) }

func (f *filteredFinite) Duration() float64 { return f.fin.Duration() }

// SelectorNames lists the built-in selector spellings, sorted, for help and for
// the catalog.
func SelectorNames() []string {
	out := []string{"ink", "fg(...)", "inner(...)", "outer(...)", "not(...)", "all(...)", "any(...)"}
	sort.Strings(out)
	return out
}

// SelectorGrammar describes the expressions ParseSelector accepts.
func SelectorGrammar() string {
	return "ink, fg(#rrggbb|none), inner(H[,V]), outer(H[,V]), not(A), all(A,B,..) or any(A,B,..)"
}

// ParseSelector resolves a selector expression: a name such as "ink", or a
// combinator over expressions such as "all(ink, inner(1))" or "not(ink)".
//
// The grammar mirrors ParsePattern's, deliberately: the same identifier reading,
// the same whitespace tolerance, the same did-you-mean on a typo. A flag that
// takes an expression should not feel different from the other flag that does.
func ParseSelector(expr string) (Selector, error) {
	s, rest, err := parseSelectorExpr(expr)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rest) != "" {
		return nil, fmt.Errorf("unexpected %q after the selector: %s", strings.TrimSpace(rest), SelectorGrammar())
	}
	return s, nil
}

func parseSelectorExpr(s string) (Selector, string, error) {
	s = strings.TrimLeft(s, " \t")
	name, rest := patternIdent(s)
	if name == "" {
		return nil, "", fmt.Errorf("expected %s", SelectorGrammar())
	}
	lower := strings.ToLower(name)
	if !strings.HasPrefix(rest, "(") {
		switch lower {
		case "ink":
			return SelInk, rest, nil
		case "not", "all", "any", "fg", "inner", "outer":
			return nil, "", fmt.Errorf("selector %q takes arguments: write %s(...)", name, lower)
		}
		return nil, "", unknownSelector(name)
	}
	switch lower {
	case "ink":
		return nil, "", fmt.Errorf("ink takes no arguments")
	case "not":
		a, inner, err := parseSelectorExpr(rest[1:])
		if err != nil {
			return nil, "", fmt.Errorf("not: %w", err)
		}
		inner, err = patternExpect(inner, ')')
		if err != nil {
			return nil, "", fmt.Errorf("not takes one selector: %w", err)
		}
		return SelNot(a), inner, nil
	case "all", "any":
		kids, inner, err := parseSelectorList(rest[1:])
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", lower, err)
		}
		if lower == "all" {
			return AllOf(kids...), inner, nil
		}
		return AnyOf(kids...), inner, nil
	case "fg":
		arg, inner := selectorToken(rest[1:])
		inner, err := patternExpect(inner, ')')
		if err != nil {
			return nil, "", fmt.Errorf("fg takes one colour: %w", err)
		}
		c, err := parseSelectorColour(arg)
		if err != nil {
			return nil, "", err
		}
		return SelFG(c), inner, nil
	case "inner", "outer":
		n, inner, err := parseSelectorMargins(rest[1:])
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", lower, err)
		}
		inner, err = patternExpect(inner, ')')
		if err != nil {
			return nil, "", fmt.Errorf("%s takes a margin: %w", lower, err)
		}
		if lower == "inner" {
			return SelInner(n[0], n[1]), inner, nil
		}
		return SelOuter(n[0], n[1]), inner, nil
	}
	return nil, "", unknownSelector(name)
}

// parseSelectorList reads one or more comma-separated selectors up to the
// closing parenthesis.
func parseSelectorList(s string) ([]Selector, string, error) {
	var kids []Selector
	for {
		a, rest, err := parseSelectorExpr(s)
		if err != nil {
			return nil, "", err
		}
		kids = append(kids, a)
		s = strings.TrimLeft(rest, " \t")
		if strings.HasPrefix(s, ",") {
			s = s[1:]
			continue
		}
		rest, err = patternExpect(s, ')')
		if err != nil {
			return nil, "", fmt.Errorf("expected another selector or `)`, got %q", truncate(s))
		}
		return kids, rest, nil
	}
}

// selectorToken reads a raw argument up to the next comma or closing
// parenthesis, which is what a colour spelling needs.
func selectorToken(s string) (tok, rest string) {
	i := 0
	for i < len(s) && s[i] != ',' && s[i] != ')' {
		i++
	}
	return strings.TrimSpace(s[:i]), s[i:]
}

// parseSelectorMargins reads h or h,v and returns both, defaulting v to h.
func parseSelectorMargins(s string) ([2]int, string, error) {
	first, rest, err := patternFloat(s)
	if err != nil {
		return [2]int{}, s, fmt.Errorf("want a margin, %w", err)
	}
	n := [2]int{int(first), int(first)}
	if first != math.Trunc(first) || first < 0 {
		return [2]int{}, s, fmt.Errorf("a margin must be a whole number of cells, got %g", first)
	}
	rest = strings.TrimLeft(rest, " \t")
	if strings.HasPrefix(rest, ",") {
		second, rest2, err := patternFloat(rest[1:])
		if err != nil {
			return [2]int{}, rest, fmt.Errorf("want a vertical margin, %w", err)
		}
		if second != math.Trunc(second) || second < 0 {
			return [2]int{}, rest, fmt.Errorf("a margin must be a whole number of cells, got %g", second)
		}
		n[1] = int(second)
		rest = rest2
	}
	return n, rest, nil
}

// parseSelectorColour reads "none" or a hex colour.
func parseSelectorColour(arg string) (tint.Color, error) {
	arg = strings.TrimSpace(arg)
	if arg == "none" {
		return tint.None, nil
	}
	if arg == "" {
		return tint.None, fmt.Errorf("fg wants a colour like #rrggbb or none, got nothing")
	}
	c, err := tint.Hex(arg)
	if err != nil {
		return tint.None, fmt.Errorf("fg wants a colour like #rrggbb or none, got %q", arg)
	}
	return c, nil
}

func unknownSelector(name string) error {
	base := []string{"ink", "not", "all", "any", "fg", "inner", "outer"}
	if s := closest(name, base); s != "" {
		return fmt.Errorf("unknown selector %q: did you mean %q? %s", name, s, SelectorGrammar())
	}
	return fmt.Errorf("unknown selector %q: use %s", name, SelectorGrammar())
}
