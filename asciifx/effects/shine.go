package effects

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:  "shine",
		Title: "Shine",
		Description: "A polished-metal glint: a soft diagonal band of light glides across the content every few seconds " +
			"over a calm palette gradient, like a logo catching the light.",
		Kind:    fx.Transition,
		Content: true,
		Tags:    []string{"loop", "logo", "highlight", "banner"},
		Glyphs:  []string{"ascii"},
		FPS:     30,
		MinW:    1,
		MinH:    1,
		Params: []fx.Param{
			fx.FloatParam("period", 2.5, 0.3, 30, "Seconds between glints."),
			fx.FloatParam("sweep", 0.55, 0.1, 1, "Fraction of the period the band spends crossing; the rest is rest."),
			fx.FloatParam("width", 0.14, 0.02, 1, "Band width as a fraction of the content's diagonal."),
			fx.FloatParam("strength", 0.9, 0, 1, "How close the band's core gets to the highlight colour."),
			fx.FloatParam("slant", 1, -3, 3, "Band angle: 0 is vertical, 1 a 45° diagonal, negative leans the other way."),
			fx.PaletteParam("palette", "catppuccin", "Base gradient laid across the content."),
			fx.EnumParam("base", "gradient", []string{"content", "gradient"}, "Base colour: the content's own colour, or the palette."),
			fx.EnumParam("direction", "horizontal", tint.Directions, "Direction of the base gradient."),
			fx.ColorParamOf("highlight", "#ffffff", "Colour at the band's core."),
		},
		Example: `asciifx play shine --banner "PREMIUM" -p palette=gruvbox -p period=3`,
		New:     newShine,
	})
}

type shine struct {
	period, sweep, width, strength, slant float64
	pal                                   tint.Gradient
	useGradient                           bool
	dir                                   tint.Direction
	highlight                             tint.Color
}

func newShine(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	s := &shine{
		period:      p.Float("period"),
		sweep:       p.Float("sweep"),
		width:       p.Float("width"),
		strength:    p.Float("strength"),
		slant:       p.Float("slant"),
		pal:         p.Palette("palette"),
		useGradient: p.String("base") == "gradient",
		dir:         tint.Direction(p.String("direction")),
		highlight:   p.Color("highlight"),
	}
	if !s.highlight.Valid {
		s.highlight = tint.RGB(255, 255, 255)
	}
	return s, nil
}

func (s *shine) Step(f *fx.Frame) {
	b := f.Buf
	bx, by, w, h := fx.Bounds(b)
	// Distance along the band's normal, normalised so the band always
	// enters fully off one side and leaves fully off the other.
	tall := float64(h-1) * fxRowAspect * math.Abs(s.slant)
	span := float64(w-1) + tall
	cycle := math.Mod(f.T()/s.period, 1)
	pos := cycle / s.sweep // 0..1 while crossing, >1 while resting
	center := -s.width + pos*(1+2*s.width)
	sigma := s.width / 2
	for y := by; y < by+h; y++ {
		for x := bx; x < bx+w; x++ {
			c := b.At(x, y)
			if !fx.Ink(c) {
				continue
			}
			lx, ly := x-bx, y-by
			base := c.FG
			if s.useGradient || !base.Valid {
				base = s.pal.At(0.35 + 0.6*tint.Position(s.dir, lx, ly, w, h))
			}
			along := float64(lx)
			if s.slant >= 0 {
				along += float64(ly) * fxRowAspect * s.slant
			} else {
				along += float64(h-1-ly) * fxRowAspect * -s.slant
			}
			d := 0.0
			if span > 0 {
				d = along / span
			}
			k := 0.0
			if pos <= 1 {
				k = math.Exp(-(d - center) * (d - center) / (2 * sigma * sigma))
			}
			c.FG = tint.Lerp(base, s.highlight, k*s.strength)
		}
	}
}

// fxRowAspect is how many columns look as tall as one row.
const fxRowAspect = 2.0
