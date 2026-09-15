package effects

import (
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/ease"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "reveal",
		Title:       "Reveal",
		Description: "Content sweeps in along a spatial pattern. Each cell scrambles through noise glyphs, flashes bright, then cools through the palette to its final colour.",
		Kind:        fx.Transition,
		Content:     true,
		Tags:        []string{"intro", "splash", "banner"},
		Glyphs:      []string{"ascii", "block"},
		FPS:         30,
		Duration:    1.6,
		MinW:        1,
		MinH:        1,
		Params: []fx.Param{
			fx.FloatParam("duration", 1.6, 0.1, 30, "Seconds for the whole reveal."),
			fx.PatternParamOf("pattern", "diagonal", "Order cells appear in."),
			fx.FloatParam("softness", 0.35, 0.01, 1, "Width of the moving edge: 0.05 is a hard wipe, 1 animates everything at once."),
			fx.FloatParam("cell_time", 0.45, 0.05, 1, "Fraction of the edge each cell spends scrambling before its flash settles."),
			fx.PaletteParam("palette", "aurora", "Colours cells cool through, dark to bright."),
			fx.EnumParam("final", "gradient", []string{"content", "gradient"}, "Resting colour: the content's own colour, or the palette laid across it."),
			fx.EnumParam("direction", "diagonal", tint.Directions, "Direction of the final gradient."),
			fx.EasingParam("easing", "out-cubic", "Global progress curve."),
			fx.StringParam("noise", "░▒▓█", "Glyphs cells scramble through. Single-width only."),
			fx.BoolParam("out", false, "Play in reverse: content dissolves away."),
		},
		Example: `asciifx play reveal --text "HELLO" -p pattern=center -p palette=synthwave`,
		New:     newReveal,
	})
}

type reveal struct {
	dur, soft, cellTime float64
	pattern             fx.Pattern
	pal                 tint.Gradient
	useGradient, out    bool
	dir                 tint.Direction
	ease                ease.Func
	noise               []rune
	seed                uint64
}

func newReveal(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	seed := rng.Uint64()
	return &reveal{
		dur:         p.Float("duration"),
		soft:        p.Float("softness"),
		cellTime:    p.Float("cell_time"),
		pattern:     p.Pattern("pattern", seed),
		pal:         p.Palette("palette"),
		useGradient: p.String("final") == "gradient",
		out:         p.Bool("out"),
		dir:         tint.Direction(p.String("direction")),
		ease:        p.Easing("easing"),
		noise:       safeRunes(p.String("noise"), "░▒▓█"),
		seed:        seed,
	}, nil
}

func (r *reveal) Duration() float64 { return r.dur }

func (r *reveal) Step(f *fx.Frame) {
	b := f.Buf
	p := r.ease(fx.Progress(f, r.dur))
	if r.out {
		p = 1 - p
	}
	bright := r.pal.At(1)
	// Lay the pattern and gradient over the content, not the whole buffer,
	// so small text gets the full sweep.
	bx, by, w, h := fx.Bounds(b)
	for y := by; y < by+h; y++ {
		for x := bx; x < bx+w; x++ {
			c := b.At(x, y)
			if !fx.Ink(c) {
				continue
			}
			lx, ly := x-bx, y-by
			final := c.FG
			if r.useGradient || !final.Valid {
				final = r.pal.At(0.35 + 0.65*tint.Position(r.dir, lx, ly, w, h))
			}
			local := fx.Local(p, r.pattern.Order(lx, ly, w, h), r.soft)
			switch {
			case local <= 0:
				*c = cell.Blank
			case local >= 1:
				c.FG = final
			default:
				// Scramble phase, then a flash that cools to the resting colour.
				phase := local / r.cellTime
				if phase < 1 {
					n := int(fx.Hash01(x+f.Tick, y, r.seed) * float64(len(r.noise)))
					c.Rune = r.noise[min(n, len(r.noise)-1)]
					c.FG = r.pal.At(phase * 0.8)
				} else {
					settle := (local - r.cellTime) / (1 - r.cellTime + 1e-9)
					c.FG = tint.Lerp(bright, final, ease.OutQuad(settle))
				}
			}
		}
	}
}

// safeRunes keeps only single-width runes, falling back when none survive.
func safeRunes(s, fallback string) []rune {
	var out []rune
	for _, r := range s {
		if cell.Safe(r) {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return []rune(fallback)
	}
	return out
}
