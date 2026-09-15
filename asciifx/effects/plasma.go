package effects

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "plasma",
		Title:       "Plasma",
		Description: "Demoscene sine plasma: interfering waves and a wandering radial ripple, rendered in half-blocks and cycling through the palette in OKLab.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "classic", "demoscene"},
		Glyphs:      []string{"halfblock"},
		FPS:         30,
		MinW:        4,
		MinH:        2,
		Params: []fx.Param{
			fx.PaletteParam("palette", "synthwave", "Colours the field cycles through."),
			fx.FloatParam("speed", 1, 0, 5, "Animation speed multiplier."),
			fx.FloatParam("scale", 1, 0.2, 5, "Wave frequency. Higher gives smaller blobs."),
		},
		Example: "asciifx play plasma -p palette=aurora -p scale=0.6",
		New:     newPlasma,
	})
}

type plasma struct {
	pal   tint.Gradient
	speed float64
	scale float64
	phase float64 // random start so seeds differ
}

func newPlasma(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &plasma{
		pal:   p.Palette("palette"),
		speed: p.Float("speed"),
		scale: p.Float("scale"),
		phase: rng.Float64() * 100,
	}, nil
}

func (e *plasma) Step(fr *fx.Frame) {
	b := fr.Buf
	t := fr.T()*e.speed + e.phase
	k := e.scale * 0.18
	// Pixel grid is W × 2H: half-block pixels are roughly square.
	pw, ph := float64(b.W)*k, float64(b.H*2)*k
	cx := pw * (0.5 + 0.4*math.Sin(t*0.31))
	cy := ph * (0.5 + 0.4*math.Cos(t*0.23))
	sample := func(x, y int) tint.Color {
		u, v := float64(x)*k, float64(y)*k
		v1 := math.Sin(u + t*0.9)
		v2 := math.Sin(u*math.Sin(t*0.21)+v*math.Cos(t*0.17)+t*0.7) * 0.9
		dx, dy := u-cx, v-cy
		v3 := math.Sin(math.Sqrt(dx*dx+dy*dy+1)*1.3 - t*1.1)
		v4 := math.Sin(v*0.7 - t*0.5)
		val := (v1 + v2 + v3 + v4) / 4 // [-1,1]
		return e.pal.Cyclic(val*0.6 + t*0.02)
	}
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			b.Set(x, y, HalfBlock(sample(x, y*2), sample(x, y*2+1)))
		}
	}
}

// ambClamp clamps v to [0,1].
func ambClamp(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// ambSmooth is the smoothstep ease on [0,1].
func ambSmooth(v float64) float64 {
	v = ambClamp(v)
	return v * v * (3 - 2*v)
}
