package effects

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "donut",
		Title:       "Spinning Donut",
		Description: "a1k0n's donut.c: a lit torus tumbling in 3D with a z-buffer, shaded by the classic luminance ramp and coloured by light through the palette.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "classic", "3d"},
		Glyphs:      []string{"ascii"},
		FPS:         30,
		MinW:        10,
		MinH:        5,
		Params: []fx.Param{
			fx.PaletteParam("palette", "sunset", "Shading colours, shadow to highlight."),
			fx.FloatParam("speed", 1, 0, 5, "Rotation speed multiplier."),
			fx.FloatParam("size", 1, 0.3, 1, "Torus size relative to the screen."),
		},
		Example: "asciifx play donut -p palette=synthwave -p speed=1.5",
		New:     newDonut,
	})
}

type donut struct {
	pal    tint.Gradient
	speed  float64
	size   float64
	a0, b0 float64
	zbuf   []float64
	lum    []float64
	colors []tint.Color
}

func newDonut(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &donut{
		pal:    p.Palette("palette"),
		speed:  p.Float("speed"),
		size:   p.Float("size"),
		a0:     rng.Float64() * 2 * math.Pi,
		b0:     rng.Float64() * 2 * math.Pi,
		zbuf:   make([]float64, w*h),
		lum:    make([]float64, w*h),
		colors: p.Palette("palette").Steps(48),
	}, nil
}

func (e *donut) Step(fr *fx.Frame) {
	const r1, r2, k2 = 1.0, 2.0, 5.0
	b := fr.Buf
	w, h := b.W, b.H
	t := fr.T() * e.speed
	A := e.a0 + t*1.1
	B := e.b0 + t*0.55
	// Worst-case projected half-extent of the torus is 0.75*K1; fit it to
	// the width, and to twice the height since cells are twice as tall.
	k1 := math.Min(0.48*float64(w), 0.96*float64(h)) / 0.75 * e.size
	for i := range e.zbuf {
		e.zbuf[i] = 0
		e.lum[i] = -1
	}
	cA, sA, cB, sB := math.Cos(A), math.Sin(A), math.Cos(B), math.Sin(B)
	// Sample density scales with size so big screens stay solid.
	thStep := math.Min(0.07, 1.2/k1)
	phStep := math.Min(0.02, 0.6/k1)
	for th := 0.0; th < 2*math.Pi; th += thStep {
		ct, st := math.Cos(th), math.Sin(th)
		for ph := 0.0; ph < 2*math.Pi; ph += phStep {
			cp, sp := math.Cos(ph), math.Sin(ph)
			cx := r2 + r1*ct
			cy := r1 * st
			x := cx*(cB*cp+sA*sB*sp) - cy*cA*sB
			y := cx*(sB*cp-sA*cB*sp) + cy*cA*cB
			z := k2 + cA*cx*sp + cy*sA
			ooz := 1 / z
			xp := int(float64(w)/2 + k1*ooz*x)
			yp := int(float64(h)/2 - k1*ooz*y*0.5)
			if xp < 0 || yp < 0 || xp >= w || yp >= h {
				continue
			}
			i := yp*w + xp
			if ooz <= e.zbuf[i] {
				continue
			}
			e.zbuf[i] = ooz
			e.lum[i] = cp*ct*sB - cA*ct*sp - sA*st + cB*(cA*st-ct*sA*sp)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if e.zbuf[i] == 0 {
				b.Set(x, y, cell.Blank)
				continue
			}
			l := ambClamp(e.lum[i] / math.Sqrt2) // 0 for faces turned from the light
			v := 0.06 + 0.94*l
			c := e.colors[int(ambClamp(0.2+0.8*math.Pow(l, 0.8))*float64(len(e.colors)-1))]
			b.Set(x, y, cell.Cell{Rune: RampRune(v), FG: c})
		}
	}
}
