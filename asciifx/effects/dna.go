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
		Name:        "dna",
		Title:       "Double Helix",
		Description: "A rotating DNA double helix drawn in braille: the front strand blazes, the back strand sinks into shadow, and base-pair rungs twist between them.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "science", "3d"},
		Glyphs:      []string{"braille"},
		FPS:         30,
		MinW:        8,
		MinH:        4,
		Params: []fx.Param{
			fx.PaletteParam("palette", "aurora", "Strand colours, back (dark) to front (bright)."),
			fx.FloatParam("speed", 1, 0, 5, "Rotation speed multiplier."),
			fx.FloatParam("twist", 2.5, 0.5, 8, "Full turns across the width."),
			fx.IntParam("rungs", 5, 2, 12, "Cells between base-pair rungs."),
		},
		Example: "asciifx play dna -p palette=synthwave -p twist=3",
		New:     newDNA,
	})
}

// dots is a braille canvas at 2×4 sub-cell resolution. Each cell keeps the
// colour of its highest-priority dot, since braille carries one foreground.
type dots struct {
	w, h int // cells
	bits []uint8
	col  []tint.Color
	pri  []float64
}

var brailleBit = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func newDots(w, h int) *dots {
	return &dots{w: w, h: h, bits: make([]uint8, w*h), col: make([]tint.Color, w*h), pri: make([]float64, w*h)}
}

func (d *dots) clear() {
	for i := range d.bits {
		d.bits[i], d.col[i], d.pri[i] = 0, tint.None, math.Inf(-1)
	}
}

// plot lights sub-pixel (px,py); pri decides which colour a shared cell wears.
func (d *dots) plot(px, py int, c tint.Color, pri float64) {
	if px < 0 || py < 0 || px >= d.w*2 || py >= d.h*4 {
		return
	}
	i := (py/4)*d.w + px/2
	d.bits[i] |= brailleBit[py%4][px%2]
	if pri >= d.pri[i] {
		d.pri[i], d.col[i] = pri, c
	}
}

// line plots from (x0,y0) to (x1,y1) in sub-pixels, colour by position t.
func (d *dots) line(x0, y0, x1, y1 float64, c func(t float64) (tint.Color, float64)) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for s := 0; s <= n; s++ {
		t := float64(s) / float64(n)
		col, pri := c(t)
		d.plot(int(math.Floor(x0+(x1-x0)*t)), int(math.Floor(y0+(y1-y0)*t)), col, pri)
	}
}

func (d *dots) draw(b *cell.Buffer) {
	for y := 0; y < d.h; y++ {
		for x := 0; x < d.w; x++ {
			i := y*d.w + x
			if d.bits[i] == 0 {
				b.Set(x, y, cell.Blank)
				continue
			}
			b.Set(x, y, cell.Cell{Rune: 0x2800 + rune(d.bits[i]), FG: d.col[i]})
		}
	}
}

type dna struct {
	pal   tint.Gradient
	speed float64
	twist float64
	rungs int
	phase float64
	d     *dots
}

func newDNA(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &dna{
		pal:   p.Palette("palette"),
		speed: p.Float("speed"),
		twist: p.Float("twist"),
		rungs: p.Int("rungs"),
		phase: rng.Float64() * 2 * math.Pi,
		d:     newDots(w, h),
	}, nil
}

func (e *dna) Step(fr *fx.Frame) {
	d := e.d
	d.clear()
	t := fr.T() * e.speed
	W, H := float64(d.w), float64(d.h)
	k := e.twist * 2 * math.Pi / (W * 2) // radians per sub-pixel column
	rot := t*1.6 + e.phase
	amp := H * 4 * 0.36
	axis := func(px float64) float64 {
		return H*2 + H*0.35*math.Sin(px/(W*2)*math.Pi*1.3+t*0.4)
	}
	shade := func(z, px float64) tint.Color {
		zn := (z + 1) / 2 // 0 back, 1 front
		c := e.pal.At(0.12 + 0.88*math.Pow(zn, 1.3))
		// A slow shimmer travels along the strand.
		return tint.Scale(c, 0.9+0.12*math.Sin(px*0.08-t*2.5))
	}

	// Rungs sit at fixed helix angles so they turn with it.
	step := float64(e.rungs * 2)
	off := math.Mod(rot/k, step)
	for rx := off; rx < W*2; rx += step {
		th := rx*k - rot
		ya, yb := axis(rx)+amp*math.Sin(th), axis(rx)-amp*math.Sin(th)
		za := math.Cos(th)
		face := 0.35 + 0.65*math.Abs(math.Sin(th))
		ca := tint.Scale(e.pal.At(0.35+0.45*(za+1)/2), face)
		cb := tint.Scale(e.pal.Cyclic(0.55+0.45*(1-za)/2), face)
		// Dotted rungs keep the strands dominant.
		lo, hi := math.Min(ya, yb)+2, math.Max(ya, yb)-2
		for y := lo; y <= hi; y += 2 {
			c := ca
			if (y-lo)/(hi-lo+1e-9) >= 0.5 == (ya < yb) {
				c = cb
			}
			d.plot(int(rx), int(y), c, -2)
		}
	}
	for px := 0; px < d.w*2; px++ {
		x := float64(px)
		th := x*k - rot
		for s, sign := range [2]float64{1, -1} {
			z := math.Cos(th) * sign
			y := axis(x) + amp*math.Sin(th)*sign
			c := shade(z, x+float64(s)*40)
			d.plot(px, int(y), c, z)
			if z > -0.2 { // nearer strand reads thicker
				d.plot(px, int(y)+1, c, z)
			}
		}
	}
	d.draw(fr.Buf)
}
