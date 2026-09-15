package fx

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// marker writes one rune for its whole duration, so a frame says which link of
// a chain was active.
type marker struct {
	r rune
	d float64
}

func (m marker) Duration() float64 { return m.d }

func (m marker) Step(f *Frame) { f.Buf.WriteString(0, 0, string(m.r), tint.None) }

// forever is a marker with no end, standing in for an ambient simulation.
type forever struct{ r rune }

func (e forever) Step(f *Frame) { f.Buf.WriteString(0, 0, string(e.r), tint.None) }

// registerMarker adds a test effect under a unique name. dur 0 makes it loop.
func registerMarker(t *testing.T, name string, r rune, dur float64) *Spec {
	t.Helper()
	s := &Spec{
		Name: name, Title: name, Kind: Ambient,
		Duration: dur, FPS: 30, MinW: 1, MinH: 1, DefW: 6, DefH: 1,
		New: func(Values, int, int, *rand.Rand) (Effect, error) {
			if dur == 0 {
				return forever{r}, nil
			}
			return marker{r, dur}, nil
		},
	}
	Register(*s)
	return s
}

// firstRune returns the first rune of frame tick.
func firstRune(t *testing.T, r *Run, tick int) rune {
	t.Helper()
	b, err := r.Seek(tick)
	if err != nil {
		t.Fatalf("Seek(%d): %v", tick, err)
	}
	return []rune(b.Plain())[0]
}

func TestComposePlaysStepsInOrder(t *testing.T) {
	registerMarker(t, "tchain-a", 'A', 1.0)
	registerMarker(t, "tchain-b", 'B', 0.5)
	spec, err := Compose(Step{Name: "tchain-a"}, Step{Name: "tchain-b"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != Ambient || spec.Duration != 1.5 {
		t.Fatalf("kind=%v duration=%g, want ambient 1.5", spec.Kind, spec.Duration)
	}
	if spec.Name != "tchain-a+tchain-b" {
		t.Fatalf("name %q", spec.Name)
	}
	if spec.FPS != 30 || spec.DefW != 6 || spec.MinW != 1 {
		t.Fatalf("derived size/fps wrong: fps=%d def=%dx%d min=%dx%d", spec.FPS, spec.DefW, spec.DefH, spec.MinW, spec.MinH)
	}
	r, err := NewRun(spec, Options{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	// 1.5s at 30fps rounds up to 46 frames: ticks 0..45.
	if got := r.Frames(); got != 46 {
		t.Fatalf("Frames() = %d, want 46", got)
	}
	for _, c := range []struct {
		tick int
		want rune
	}{{0, 'A'}, {29, 'A'}, {30, 'B'}, {44, 'B'}, {45, 'B'}} {
		if got := firstRune(t, r, c.tick); got != c.want {
			t.Errorf("tick %d = %q, want %q", c.tick, got, c.want)
		}
	}
}

func TestComposeNeedsADurationForLoopingSteps(t *testing.T) {
	registerMarker(t, "tchain-loop", 'L', 0)
	registerMarker(t, "tchain-end", 'E', 0.5)
	if _, err := Compose(Step{Name: "tchain-loop"}, Step{Name: "tchain-end"}); err == nil {
		t.Fatal("a looping step with no --for must be rejected")
	} else if !strings.Contains(err.Error(), "tchain-loop") {
		t.Fatalf("error should name the step: %v", err)
	}
	// Giving it a duration makes it legal, and the duration drives the length.
	spec, err := Compose(Step{Name: "tchain-loop", For: 2}, Step{Name: "tchain-end"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Duration != 2.5 {
		t.Fatalf("duration %g, want 2.5", spec.Duration)
	}
	r, err := NewRun(spec, Options{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := firstRune(t, r, 0); got != 'L' {
		t.Fatalf("tick 0 = %q, want L", got)
	}
	if got := firstRune(t, r, 59); got != 'L' {
		t.Fatalf("tick 59 (1.97s) = %q, want L", got)
	}
	if got := firstRune(t, r, 60); got != 'E' {
		t.Fatalf("tick 60 (2s) = %q, want E", got)
	}
}

func TestComposeForOverridesAFiniteDuration(t *testing.T) {
	registerMarker(t, "tchain-short", 'S', 5.0)
	registerMarker(t, "tchain-tail", 'T', 0.5)
	spec, err := Compose(Step{Name: "tchain-short", For: 0.5}, Step{Name: "tchain-tail"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Duration != 1.0 {
		t.Fatalf("duration %g, want 1.0", spec.Duration)
	}
}

func TestComposeContentIsShared(t *testing.T) {
	plain := &Spec{
		Name: "tchain-plain", Title: "plain", Kind: Ambient, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 6, DefH: 1,
		New: func(Values, int, int, *rand.Rand) (Effect, error) {
			// Scribbles over the target content, the way a transition that has
			// not finished draws noise.
			return marker{'P', 0.5}, nil
		},
	}
	Register(*plain)
	// probe reports whether the target content was in the buffer when its tick
	// began, by writing Y or N into column 1.
	probe := &Spec{
		Name: "tchain-probe", Title: "probe", Kind: Transition, Content: true, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 2, DefW: 6, DefH: 2,
		New: func(Values, int, int, *rand.Rand) (Effect, error) {
			return StepFunc(func(f *Frame) {
				seen := 'N'
				if c := f.Buf.At(0, 0); c != nil && c.Rune == 'Z' {
					seen = 'Y'
				}
				f.Buf.WriteString(1, 0, string(seen), tint.None)
			}), nil
		},
	}
	Register(*probe)

	spec, err := Compose(Step{Name: "tchain-plain"}, Step{Name: "tchain-probe"})
	if err != nil {
		t.Fatal(err)
	}
	if !spec.Content {
		t.Fatal("one content step must make the whole composition content-driven")
	}
	if spec.Kind != Transition {
		t.Fatalf("kind %v, want transition", spec.Kind)
	}
	if spec.MinH != 2 {
		t.Fatalf("MinH %d, want the maximum over the steps (2)", spec.MinH)
	}
	r, err := NewRun(spec, Options{Seed: 1, Content: func(b *cell.Buffer) {
		b.WriteString(0, 0, "Z", tint.None)
	}})
	if err != nil {
		t.Fatal(err)
	}
	// While the plain step runs it overwrites the content.
	if b, _ := r.Seek(0); []rune(b.Plain())[0] != 'P' {
		t.Fatalf("tick 0 = %q, want the plain step's P", b.Plain())
	}
	// A later content step must see the target restored behind it, even though
	// an earlier step had scribbled over it.
	b, err := r.Seek(20)
	if err != nil {
		t.Fatal(err)
	}
	if got := []rune(b.Plain())[1]; got != 'Y' {
		t.Fatalf("content was not copied in before the active step: %q", b.Plain())
	}
}

// TestComposeAppendingAStepKeepsEarlierFrames is the property the per-step
// seed derivation exists for: an author can add a link without invalidating
// everything already tuned.
func TestComposeAppendingAStepKeepsEarlierFrames(t *testing.T) {
	seeded := &Spec{
		Name: "tchain-seeded", Title: "seeded", Kind: Ambient, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 6, DefH: 1,
		New: func(_ Values, _, _ int, rng *rand.Rand) (Effect, error) {
			// A per-effect random quantity, like the seed a real effect keeps.
			return marker{rune('a' + rng.IntN(26)), 0.5}, nil
		},
	}
	Register(*seeded)
	registerMarker(t, "tchain-after", 'X', 0.5)

	short, err := Compose(Step{Name: "tchain-seeded"})
	if err != nil {
		t.Fatal(err)
	}
	long, err := Compose(Step{Name: "tchain-seeded"}, Step{Name: "tchain-after"})
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewRun(short, Options{Seed: 9})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRun(long, Options{Seed: 9})
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 15; tick++ {
		ba, _ := a.Seek(tick)
		bb, _ := b.Seek(tick)
		if ba.Plain() != bb.Plain() {
			t.Fatalf("tick %d changed after appending a step: %q vs %q", tick, ba.Plain(), bb.Plain())
		}
	}
}

func TestComposeIsDeterministic(t *testing.T) {
	registerMarker(t, "tchain-det-a", 'A', 0.3)
	registerMarker(t, "tchain-det-b", 'B', 0.3)
	build := func() string {
		spec, err := Compose(Step{Name: "tchain-det-a"}, Step{Name: "tchain-det-b"})
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewRun(spec, Options{Seed: 12})
		if err != nil {
			t.Fatal(err)
		}
		var sb strings.Builder
		for tick := 0; tick < 18; tick++ {
			b, err := r.Seek(tick)
			if err != nil {
				t.Fatal(err)
			}
			sb.WriteString(b.Plain())
			sb.WriteByte('\n')
		}
		return sb.String()
	}
	first := build()
	for i := 0; i < 3; i++ {
		if again := build(); again != first {
			t.Fatalf("run %d differs from the first", i+2)
		}
	}
}

func TestTimedEndsAnEndlessEffect(t *testing.T) {
	e := Timed(forever{'F'}, 0.25)
	if e.Duration() != 0.25 {
		t.Fatalf("Duration() = %g, want 0.25", e.Duration())
	}
	f := &Frame{Buf: cell.New(4, 1), Dt: 1, Rand: NewRand(1)}
	e.Step(f)
	if got := rune(f.Buf.Plain()[0]); got != 'F' {
		t.Fatalf("Timed drew %q, want F", got)
	}
	if Timed(forever{'F'}, -1).Duration() != 0 {
		t.Fatal("a negative duration must clamp to zero")
	}
}

func TestComposeErrors(t *testing.T) {
	registerMarker(t, "tchain-ok", 'O', 0.5)
	if _, err := Compose(); err == nil {
		t.Fatal("an empty composition must be rejected")
	}
	if _, err := Compose(Step{Name: "no-such-effect"}); err == nil {
		t.Fatal("an unknown step must be rejected")
	}
	if _, err := Compose(Step{Name: "tchain-ok", Params: map[string]string{"nope": "1"}}); err == nil {
		t.Fatal("an unknown param must be rejected")
	}
}

// TestComposeFailsOnABadChildBuild checks the build path, not just the spec
// path: a child that refuses to construct names its own step.
func TestComposeFailsOnABadChildBuild(t *testing.T) {
	bad := &Spec{
		Name: "tchain-bad", Title: "bad", Kind: Ambient, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 6, DefH: 1,
		New: func(Values, int, int, *rand.Rand) (Effect, error) {
			return nil, errTestBuild
		},
	}
	Register(*bad)
	spec, err := Compose(Step{Name: "tchain-bad"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRun(spec, Options{Seed: 1}); err == nil {
		t.Fatal("a failing child build must fail the composition")
	} else if !strings.Contains(err.Error(), "step 1") || !strings.Contains(err.Error(), errTestBuild.Error()) {
		t.Fatalf("error should name the step and the cause: %v", err)
	}
}

var errTestBuild = errTest("the child effect refused to build")

type errTest string

func (e errTest) Error() string { return string(e) }

// TestComposeResizeKeepsPlaying covers the play --fit path, where a fullscreen
// run is rebuilt on every terminal resize. A chain of pure transitions must
// resume at its tick rather than restart, and a chain containing a simulation
// must at least keep producing frames instead of failing.
func TestComposeResizeKeepsPlaying(t *testing.T) {
	registerMarker(t, "tchain-resize-a", 'A', 0.5)
	registerMarker(t, "tchain-resize-loop", 'L', 0)
	// A transition is a function of its tick, so a content chain resumes
	// exactly; a chain holding a simulation can only restart that step.
	transition := &Spec{
		Name: "tchain-resize-c", Title: "content", Kind: Transition, Content: true, Duration: 0.5,
		FPS: 30, MinW: 1, MinH: 1, DefW: 8, DefH: 2,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return marker{'c', 0.5}, nil },
	}
	Register(*transition)
	loopSpec, err := Compose(Step{Name: "tchain-resize-a"}, Step{Name: "tchain-resize-loop", For: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		spec   *Spec
		resume bool
	}{
		{"pure chain resumes", mustCompose(t, Step{Name: "tchain-resize-c"}), true},
		{"chain with a simulation stays alive", loopSpec, false},
	} {
		r, err := NewRun(tc.spec, Options{Seed: 4, W: 8, H: 2})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if _, err := r.Seek(10); err != nil {
			t.Fatalf("%s: seek: %v", tc.name, err)
		}
		if err := r.Resize(12, 3); err != nil {
			t.Fatalf("%s: resize: %v", tc.name, err)
		}
		if w, h := r.Size(); w != 12 || h != 3 {
			t.Fatalf("%s: size %dx%d after resize", tc.name, w, h)
		}
		if b := r.Current(); b.W != 12 || b.H != 3 {
			t.Fatalf("%s: buffer %dx%d after resize", tc.name, b.W, b.H)
		}
		if tc.resume && r.Tick() != 10 {
			t.Fatalf("%s: tick %d after resize, want 10", tc.name, r.Tick())
		}
		if r.Frames() == 0 {
			t.Fatalf("%s: a composition must stay finite, got 0 frames", tc.name)
		}
	}
}

// TestComposeResizeRejectsAnImpossibleSize keeps the failure mode honest: a
// chain is only as small as its largest step's minimum.
func TestComposeResizeRejectsAnImpossibleSize(t *testing.T) {
	big := &Spec{
		Name: "tchain-big", Title: "big", Kind: Ambient, Duration: 1,
		FPS: 30, MinW: 20, MinH: 8, DefW: 40, DefH: 16,
		New: func(Values, int, int, *rand.Rand) (Effect, error) { return marker{'B', 1}, nil },
	}
	Register(*big)
	spec := mustCompose(t, Step{Name: "tchain-big"})
	if spec.MinW != 20 || spec.MinH != 8 || spec.DefW != 40 || spec.DefH != 16 {
		t.Fatalf("derived size wrong: min=%dx%d def=%dx%d", spec.MinW, spec.MinH, spec.DefW, spec.DefH)
	}
	r, err := NewRun(spec, Options{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Resize(10, 4); err == nil {
		t.Fatal("a resize below the largest step's minimum must be refused")
	}
	if w, h := r.Size(); w != 40 || h != 16 {
		t.Fatalf("a refused resize must leave the run alone, got %dx%d", w, h)
	}
}

func mustCompose(t *testing.T, steps ...Step) *Spec {
	t.Helper()
	spec, err := Compose(steps...)
	if err != nil {
		t.Fatal(err)
	}
	return spec
}
