package fx

import (
	"fmt"
	"math"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// Content draws target content for a transition into a blank buffer.
type Content func(b *cell.Buffer)

// Options configure a Run.
type Options struct {
	Params map[string]string
	W, H   int
	Seed   uint64
	// FPS overrides the spec's recommended rate; 0 keeps it.
	FPS int
	// Content is the target for transitions. Ignored by ambient effects.
	Content Content
}

// Run drives one effect deterministically. It is the single place that
// decides tick order, so the CLI, the terminal player and framework adapters
// all produce identical frames.
type Run struct {
	Spec   *Spec
	Values Values
	opts   Options
	effect Effect
	frame  Frame
	target *cell.Buffer
}

// NewRun validates params and builds the effect. Nothing is drawn until Next.
func NewRun(spec *Spec, o Options) (*Run, error) {
	if o.W <= 0 || o.H <= 0 {
		o.W, o.H = spec.DefW, spec.DefH
	}
	if err := spec.checkSize(o.W, o.H); err != nil {
		return nil, err
	}
	if o.FPS <= 0 {
		o.FPS = spec.FPS
	}
	vals, err := spec.Resolve(o.Params)
	if err != nil {
		return nil, err
	}
	r := &Run{Spec: spec, Values: vals, opts: o}
	if err := r.build(o.W, o.H); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Spec) checkSize(w, h int) error {
	if w < max(s.MinW, 1) || h < max(s.MinH, 1) {
		return fmt.Errorf("effect %s needs at least %dx%d, got %dx%d", s.Name, max(s.MinW, 1), max(s.MinH, 1), w, h)
	}
	return nil
}

// build replaces the effect state for size w×h. On error r is unchanged.
func (r *Run) build(w, h int) error {
	rng := NewRand(r.opts.Seed)
	e, err := r.Spec.New(r.Values, w, h, rng)
	if err != nil {
		return fmt.Errorf("effect %s: %w", r.Spec.Name, err)
	}
	target := cell.New(w, h)
	if r.opts.Content != nil {
		r.opts.Content(target)
	}
	r.opts.W, r.opts.H = w, h
	r.effect = e
	r.frame = Frame{Buf: cell.New(w, h), Tick: -1, Dt: 1 / float64(r.opts.FPS), Rand: rng}
	r.target = target
	return nil
}

// Next advances one tick and returns the buffer, which is reused between
// calls: copy it if you need to keep a frame.
func (r *Run) Next() *cell.Buffer {
	r.frame.Tick++
	if r.Spec.Content {
		r.frame.Buf.CopyFrom(r.target)
	}
	r.effect.Step(&r.frame)
	return r.frame.Buf
}

// Seek replays from the start to tick n and returns that frame. Because runs
// are deterministic this is exact, and cheap for the short effects agents
// usually inspect.
func (r *Run) Seek(n int) (*cell.Buffer, error) {
	if n < r.frame.Tick {
		if err := r.build(r.opts.W, r.opts.H); err != nil {
			return nil, err
		}
	}
	for r.frame.Tick < n {
		r.Next()
	}
	return r.frame.Buf, nil
}

// Current returns the most recent frame without advancing. Before the first
// Next it is a blank buffer.
func (r *Run) Current() *cell.Buffer { return r.frame.Buf }

// Tick is the index of the last frame produced, or -1 before the first.
func (r *Run) Tick() int { return r.frame.Tick }

// FPS is the tick rate in use.
func (r *Run) FPS() int { return r.opts.FPS }

// Duration is the effect's length in seconds, 0 when it loops forever.
func (r *Run) Duration() float64 {
	if fin, ok := r.effect.(Finite); ok {
		return fin.Duration()
	}
	return 0
}

// Frames is the number of frames in a finite run, or 0 when it loops.
func (r *Run) Frames() int {
	d := r.Duration()
	if d == 0 {
		return 0
	}
	// Round up so the last tick lands at or after the end: effects that
	// compute progress from time then always reach 1 on the final frame.
	return int(math.Ceil(d*float64(r.opts.FPS)-1e-9)) + 1
}

// Done reports whether a finite run has shown its last frame.
func (r *Run) Done() bool {
	f := r.Frames()
	return f > 0 && r.frame.Tick >= f-1
}

// Size returns the buffer size.
func (r *Run) Size() (w, h int) { return r.opts.W, r.opts.H }

// Resize rebuilds the effect for a new size. Transitions keep their tick so
// they do not restart; ambient simulations start fresh at the new size. If
// the size is too small or the effect fails to build, the run is left exactly
// as it was and the error is returned.
func (r *Run) Resize(w, h int) error {
	if w == r.opts.W && h == r.opts.H {
		return nil
	}
	if err := r.Spec.checkSize(w, h); err != nil {
		return err
	}
	tick := r.frame.Tick
	if err := r.build(w, h); err != nil {
		return err
	}
	if r.Spec.Content && tick >= 0 {
		// Transitions are pure in tick; jump straight there.
		r.frame.Tick = tick - 1
		r.Next()
	}
	return nil
}

// Render is the one-shot form: build the effect and return frame tick.
func Render(name string, o Options, tick int) (*cell.Buffer, *Run, error) {
	spec, err := Lookup(name)
	if err != nil {
		return nil, nil, err
	}
	r, err := NewRun(spec, o)
	if err != nil {
		return nil, nil, err
	}
	b, err := r.Seek(tick)
	return b, r, err
}
