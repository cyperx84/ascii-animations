package effects

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/ease"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:  "glitch",
		Title: "Glitch",
		Description: "A corrupted signal locking on: horizontal slices tear sideways, red and cyan channels split apart, " +
			"rows drop out and block noise sparks, all easing down until the content snaps clean.",
		Kind:     fx.Transition,
		Content:  true,
		Tags:     []string{"intro", "cyberpunk", "distortion", "banner"},
		Glyphs:   []string{"ascii", "block"},
		FPS:      30,
		Duration: 1.6,
		MinW:     1,
		MinH:     1,
		Params: []fx.Param{
			fx.FloatParam("duration", 1.6, 0.1, 30, "Seconds until the signal is clean."),
			fx.FloatParam("intensity", 1, 0, 2, "Strength of the corruption at the start."),
			fx.IntParam("shift", 8, 0, 60, "Largest sideways slice displacement, in columns."),
			fx.IntParam("slice", 2, 1, 10, "Height of torn slices, in rows."),
			fx.IntParam("hold", 2, 1, 10, "Ticks each corruption pattern holds before it jumps."),
			fx.FloatParam("noise_rate", 0.08, 0, 0.5, "Share of cells sparking noise glyphs at full intensity."),
			fx.ColorParamOf("split_a", "#ff2a6d", "Colour of the leading split channel."),
			fx.ColorParamOf("split_b", "#05d9e8", "Colour of the trailing split channel."),
			fx.PaletteParam("palette", "synthwave", "Resting gradient and noise colours."),
			fx.EnumParam("final", "gradient", []string{"content", "gradient"}, "Resting colour: the content's own colour, or the palette laid across it."),
			fx.EnumParam("direction", "horizontal", tint.Directions, "Direction of the final gradient."),
			fx.EasingParam("easing", "in-out-quad", "How fast the corruption dies away."),
			fx.StringParam("noise", "░▒▓█▚▞▀▄", "Noise glyphs. Single-width only."),
		},
		Example: `asciifx play glitch --banner "SIGNAL" -p shift=12`,
		New:     newGlitch,
	})
}

type glitch struct {
	dur, intensity, noiseRate float64
	shift, slice, hold        int
	splitA, splitB            tint.Color
	pal                       tint.Gradient
	useGradient               bool
	dir                       tint.Direction
	ease                      ease.Func
	noise                     []rune
	seed                      uint64
	src                       *cell.Buffer
}

func newGlitch(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	g := &glitch{
		dur:         p.Float("duration"),
		intensity:   p.Float("intensity"),
		noiseRate:   p.Float("noise_rate"),
		shift:       p.Int("shift"),
		slice:       p.Int("slice"),
		hold:        p.Int("hold"),
		splitA:      p.Color("split_a"),
		splitB:      p.Color("split_b"),
		pal:         p.Palette("palette"),
		useGradient: p.String("final") == "gradient",
		dir:         tint.Direction(p.String("direction")),
		ease:        p.Easing("easing"),
		noise:       safeRunes(p.String("noise"), "░▒▓█"),
		seed:        rng.Uint64(),
		src:         cell.New(w, h),
	}
	if !g.splitA.Valid {
		g.splitA = tint.RGB(255, 42, 109)
	}
	if !g.splitB.Valid {
		g.splitB = tint.RGB(5, 217, 232)
	}
	return g, nil
}

func (g *glitch) Duration() float64 { return g.dur }

func (g *glitch) Step(f *fx.Frame) {
	b := f.Buf
	p := tickProgress(f, g.dur)
	if p >= 1 {
		settle(b, g.pal, g.useGradient, g.dir)
		return
	}
	bx, by, w, h := fx.Bounds(b)
	// Rest colours go on first, so every copy below carries them. src is
	// scratch space only: it is fully rewritten from the content each tick.
	settle(b, g.pal, g.useGradient, g.dir)
	g.src.CopyFrom(b)

	bucket := f.Tick / g.hold
	// The signal stutters: some held patterns are much calmer than others.
	stutter := 0.55 + 0.45*fx.Hash01(0, bucket, g.seed^0x11)
	in := (1 - g.ease(p)) * g.intensity * stutter
	phase := int(fx.Hash01(1, bucket, g.seed^0x22) * float64(g.slice))
	x0, x1 := max(bx-g.shift-3, 0), min(bx+w+g.shift+3, b.W)

	for y := by; y < by+h; y++ {
		sl := (y - by + phase) / g.slice
		rowHash := fx.Hash01(sl, bucket, g.seed^0x33)
		off := 0
		if rowHash < math.Min(in, 1)*0.7 {
			v := fx.Hash01(sl, bucket, g.seed^0x44)*2 - 1
			off = int(math.Round(v * float64(g.shift) * math.Min(in, 1)))
		}
		if fx.Hash01(sl, bucket, g.seed^0x55) < in*in*0.25 {
			for x := x0; x < x1; x++ {
				b.Set(x, y, cell.Blank)
			}
			continue
		}
		// Channel split distance, wider in torn slices.
		split := 0
		if off != 0 || fx.Hash01(sl, bucket, g.seed^0x88) < in*0.4 {
			split = 1 + int(math.Min(in, 1)*1.5)
		}
		tintTo := g.splitA
		if off < 0 {
			tintTo = g.splitB
		}
		for x := x0; x < x1; x++ {
			out := cell.Blank
			if s := g.src.At(x-off, y); s != nil && fx.Ink(s) {
				out = *s
				if off != 0 {
					out.FG = tint.Lerp(out.FG, tintTo, 0.65*math.Min(in, 1))
				}
			} else if split > 0 {
				a, c := g.src.At(x-off-split, y), g.src.At(x-off+split, y)
				switch {
				case a != nil && fx.Ink(a):
					out = cell.Cell{Rune: ghostRune(a.Rune), FG: tint.Scale(g.splitA, (0.55+0.35*math.Min(in, 1))*math.Min(in*3, 1))}
				case c != nil && fx.Ink(c):
					out = cell.Cell{Rune: ghostRune(c.Rune), FG: tint.Scale(g.splitB, (0.55+0.35*math.Min(in, 1))*math.Min(in*3, 1))}
				}
			}
			if fx.Hash01(x, y^(bucket*7919), g.seed^0x66) < g.noiseRate*in && x >= bx && x < bx+w {
				n := fx.Hash01(x^bucket, y, g.seed^0x77)
				out = cell.Cell{Rune: g.noise[min(int(n*float64(len(g.noise))), len(g.noise)-1)], FG: g.pal.At(0.4 + 0.6*n)}
			}
			b.Set(x, y, out)
		}
	}
}

// ghostRune is how a split channel draws a glyph: solid blocks thin to a
// light shade so the fringe reads as colour bleed rather than more text.
func ghostRune(r rune) rune {
	if r >= 0x2580 && r <= 0x259F {
		return '░'
	}
	return r
}
