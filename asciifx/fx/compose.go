package fx

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
)

// Sequence plays finite effects one after another, each seeing its own tick
// count from zero. Transitions each receive the same target content.
func Sequence(effects ...Finite) Finite { return &sequence{effects: effects} }

type sequence struct{ effects []Finite }

func (s *sequence) Duration() float64 {
	var d float64
	for _, e := range s.effects {
		d += e.Duration()
	}
	return d
}

func (s *sequence) Step(f *Frame) {
	start := 0.0
	for i, e := range s.effects {
		end := start + e.Duration()
		if f.T() < end || i == len(s.effects)-1 {
			local := *f
			local.Tick = f.Tick - int(start/f.Dt+0.5)
			e.Step(&local)
			return
		}
		start = end
	}
}

// Parallel steps every effect on the same buffer in order, so later effects
// see earlier effects' output. Its duration is the longest child's.
func Parallel(effects ...Effect) Effect { return &parallel{effects: effects} }

type parallel struct{ effects []Effect }

func (p *parallel) Step(f *Frame) {
	for _, e := range p.effects {
		e.Step(f)
	}
}

func (p *parallel) Duration() float64 {
	var d float64
	for _, e := range p.effects {
		fin, ok := e.(Finite)
		if !ok {
			return 0
		}
		d = max(d, fin.Duration())
	}
	return d
}

// Delay holds the target content untouched for seconds, then runs e.
func Delay(seconds float64, e Finite) Finite { return &delay{d: seconds, e: e} }

type delay struct {
	d float64
	e Finite
}

func (d *delay) Duration() float64 { return d.d + d.e.Duration() }

func (d *delay) Step(f *Frame) {
	if f.T() < d.d {
		return
	}
	local := *f
	local.Tick = f.Tick - int(d.d/f.Dt+0.5)
	d.e.Step(&local)
}

// Hold keeps e's final frame on screen for an extra number of seconds.
func Hold(e Finite, seconds float64) Finite { return &hold{e: e, d: seconds} }

type hold struct {
	e Finite
	d float64
}

func (h *hold) Duration() float64 { return h.e.Duration() + h.d }

func (h *hold) Step(f *Frame) {
	local := *f
	if last := int(h.e.Duration()/f.Dt + 0.5); local.Tick > last {
		local.Tick = last
	}
	h.e.Step(&local)
}

// Timed gives any effect a fixed duration, so a looping effect (an ambient
// simulation, a spinner) can join a Sequence. A Sequence steps each child
// exactly once per tick while it is the active child, so a stateful
// simulation advances normally. Combinators that revisit an earlier tick or
// step a child twice at one tick — Hold, and anything that replays — need
// children whose output depends only on the tick, which a running simulation
// is not.
func Timed(e Effect, seconds float64) Finite { return &timed{e: e, d: max(0, seconds)} }

type timed struct {
	e Effect
	d float64
}

func (t *timed) Duration() float64 { return t.d }

func (t *timed) Step(f *Frame) { t.e.Step(f) }

// Step is one link in a Composition: a registered effect, the raw params to
// resolve against it, and an optional duration for effects that loop.
type Step struct {
	Name   string
	Params map[string]string
	// For gives a looping effect a duration. It is required for any effect
	// whose spec has no natural end, because a sequence can only advance past
	// an effect that finishes. It is ignored for finite effects.
	For float64
	// Filter restricts which cells this step may change. It applies to this
	// step alone: a chain does not propagate one step's filter to the others,
	// because each step is asked for its own.
	Filter Selector
}

// Compose builds a single finite effect that plays steps in order, each
// starting from a tick of zero. It returns a synthetic Spec, so every Run,
// renderer and player works on a composition exactly as it does on a
// registered effect.
//
// Content is shared: a composition whose steps include any transition gets
// the same target buffer copied in before every tick, so each transition
// transforms the same content. Per-step content is not expressible.
//
// Each step gets a generator derived from the run's seed in step order, so
// appending a step never changes the frames an earlier step produces.
//
// A composition that contains a looping step cannot be fast-forwarded: a
// simulation's state depends on how many times it has been stepped, so
// resizing such a run restarts the active step rather than resuming it. Pure
// transitions resume exactly, because their output depends only on the tick.
func Compose(steps ...Step) (*Spec, error) {
	if len(steps) == 0 {
		return nil, errors.New("a composition needs at least one step")
	}
	kids := make([]kid, 0, len(steps))
	spec := &Spec{
		Name:  "chain",
		Title: "Chain",
		Kind:  Ambient,
		// A composition is a sequence: it always finishes.
		Glyphs: []string{},
	}
	glyphs := map[string]bool{}
	names := make([]string, 0, len(steps))
	for i, st := range steps {
		child, err := Lookup(st.Name)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		vals, err := child.Resolve(st.Params)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, child.Name, err)
		}
		if child.Duration == 0 && st.For <= 0 {
			return nil, fmt.Errorf("step %d (%s) loops forever: give it a duration so the composition can move on", i+1, child.Name)
		}
		dur := child.Duration
		if st.For > 0 {
			dur = st.For
		}
		kids = append(kids, kid{spec: child, vals: vals, forDur: st.For, sel: st.Filter})
		names = append(names, child.Name)
		for _, g := range child.Glyphs {
			glyphs[g] = true
		}
		spec.Duration += dur
		spec.MinW = max(spec.MinW, child.MinW)
		spec.MinH = max(spec.MinH, child.MinH)
		spec.DefW = max(spec.DefW, child.DefW)
		spec.DefH = max(spec.DefH, child.DefH)
		spec.FPS = max(spec.FPS, child.FPS)
		spec.Content = spec.Content || child.Content
		for _, t := range child.Tags {
			if !slices.Contains(spec.Tags, t) {
				spec.Tags = append(spec.Tags, t)
			}
		}
	}
	sort.Strings(spec.Tags)
	for g := range glyphs {
		spec.Glyphs = append(spec.Glyphs, g)
	}
	sort.Strings(spec.Glyphs)
	spec.Name = strings.Join(names, "+")
	spec.Description = "Plays " + strings.Join(names, ", then ") + " in order."
	spec.Example = "asciifx play " + strings.Join(names, " --then ")
	if spec.Content {
		spec.Kind = Transition
	}
	// Checked after the loop, because the answer depends on the whole chain:
	// `reveal --then fire --for 1s --filter not(ink)` is legitimate precisely
	// because the reveal gives the run content for fire's selector to read.
	for i, k := range kids {
		if err := checkNeedsContent(k.sel, spec.Content); err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, k.spec.Name, err)
		}
	}
	spec.New = func(_ Values, w, h int, rng *rand.Rand) (Effect, error) {
		fin := make([]Finite, 0, len(kids))
		for i, k := range kids {
			e, err := k.spec.New(k.vals, w, h, NewRand(rng.Uint64()))
			if err != nil {
				return nil, fmt.Errorf("step %d (%s): %w", i+1, k.spec.Name, err)
			}
			// Wrapped before the Finite check, so a filtered step is still
			// recognised as finite and still gets its duration.
			e = Filter(e, k.sel)
			if f, ok := e.(Finite); ok && k.forDur <= 0 {
				fin = append(fin, f)
				continue
			}
			// A looping effect with an explicit duration, or one whose
			// natural duration is overridden by it.
			fin = append(fin, Timed(e, k.forDur))
		}
		return Sequence(fin...), nil
	}
	return spec, nil
}

// kid is one resolved step of a composition.
type kid struct {
	spec   *Spec
	vals   Values
	forDur float64
	sel    Selector
}
