// Package fx is the asciifx animation engine.
//
// An Effect draws into a cell.Buffer one tick at a time. Runs are
// deterministic: given the same spec, params, size and seed, tick N always
// produces the same buffer. That is what lets tools render "frame 40 as text"
// for an agent, and what makes golden-frame tests possible.
//
// There are two kinds of effect. Ambient effects (fire, matrix) own the whole
// buffer. Transitions (reveal, decrypt) animate existing content: before each
// Step the runner fills the buffer with the target content, and the effect
// transforms it.
package fx

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// Frame is the per-tick context passed to Step.
type Frame struct {
	// Buf is the buffer to draw into. For transitions it arrives holding the
	// target content.
	Buf *cell.Buffer
	// Tick counts from 0.
	Tick int
	// Dt is the fixed tick length in seconds.
	Dt float64
	// Rand is seeded once per run and shared across ticks. Effects must use
	// it (never math/rand globals or wall time) so runs stay reproducible.
	Rand *rand.Rand
}

// T is the elapsed time in seconds at the start of this tick.
func (f *Frame) T() float64 { return float64(f.Tick) * f.Dt }

// Effect advances one tick and draws into f.Buf.
type Effect interface {
	Step(f *Frame)
}

// Finite effects end. Duration is in seconds.
//
// A Finite effect's output must depend only on f.Tick and the buffer contents
// it is handed, never on earlier Step calls. The combinators rely on this to
// replay, hold and re-time children freely.
type Finite interface {
	Effect
	Duration() float64
}

// StepFunc adapts a function to Effect.
type StepFunc func(f *Frame)

func (s StepFunc) Step(f *Frame) { s(f) }

// Progress returns elapsed/duration clamped to [0,1]. A zero duration is
// already complete.
func Progress(f *Frame, duration float64) float64 {
	if duration <= 0 {
		return 1
	}
	return math.Max(0, math.Min(1, f.T()/duration))
}

// Done reports whether a finite effect has finished at this frame's tick.
func Done(e Effect, f *Frame) bool {
	fin, ok := e.(Finite)
	return ok && f.T() >= fin.Duration()
}

// NewRand returns the deterministic generator a run uses for seed.
func NewRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0x9E3779B97F4A7C15^seed))
}

// Hash01 is a stateless per-cell random value in [0,1), for patterns and
// effects that need spatial noise without advancing Rand.
func Hash01(x, y int, seed uint64) float64 {
	h := uint64(x)*0x9E3779B185EBCA87 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ seed*0x165667B19E3779F9
	h ^= h >> 33
	h *= 0xFF51AFD7ED558CCD
	h ^= h >> 33
	h *= 0xC4CEB9FE1A85EC53
	h ^= h >> 33
	return float64(h>>11) / float64(1<<53)
}
