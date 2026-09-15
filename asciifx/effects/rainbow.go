package effects

import (
	"math"
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:  "rainbow",
		Title: "Rainbow",
		Description: "lolcat, alive: a seamless perceptual colour cycle flows through the content across space and time, " +
			"stripes drifting steadily in the chosen direction.",
		Kind:    fx.Transition,
		Content: true,
		Tags:    []string{"loop", "lolcat", "colour", "banner"},
		Glyphs:  []string{"ascii"},
		FPS:     30,
		MinW:    1,
		MinH:    1,
		Params: []fx.Param{
			fx.PaletteParam("palette", "rainbow", "Colours cycled through; the ends join seamlessly."),
			fx.FloatParam("speed", 0.4, -10, 10, "Full colour cycles per second; negative flows backwards."),
			fx.FloatParam("spread", 40, 1, 500, "Columns per full colour cycle: smaller gives tighter stripes."),
			fx.EnumParam("direction", "diagonal", tint.Directions, "Which way stripes lie and flow."),
			fx.FloatParam("brightness", 1, 0.2, 1.5, "Lightness multiplier in OKLab."),
		},
		Example: `asciifx play rainbow --banner "PARTY" -p speed=0.8 -p spread=24`,
		New:     newRainbow,
	})
}

type rainbow struct {
	speed, spread, bright float64
	pal                   tint.Gradient
	dir                   tint.Direction
}

func newRainbow(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &rainbow{
		speed:  p.Float("speed"),
		spread: p.Float("spread"),
		bright: p.Float("brightness"),
		pal:    p.Palette("palette"),
		dir:    tint.Direction(p.String("direction")),
	}, nil
}

func (r *rainbow) Step(f *fx.Frame) {
	b := f.Buf
	bx, by, w, h := fx.Bounds(b)
	shift := f.T() * r.speed
	for y := by; y < by+h; y++ {
		for x := bx; x < bx+w; x++ {
			c := b.At(x, y)
			if !fx.Ink(c) {
				continue
			}
			lx, ly := float64(x-bx), float64(y-by)
			var cells float64
			switch r.dir {
			case tint.Vertical:
				cells = ly * fxRowAspect
			case tint.Diagonal:
				cells = (lx + ly*fxRowAspect) / math.Sqrt2
			case tint.Radial:
				cells = math.Hypot(lx-float64(w-1)/2, (ly-float64(h-1)/2)*fxRowAspect)
			default:
				cells = lx
			}
			col := r.pal.Cyclic(cells/r.spread - shift)
			if r.bright != 1 {
				col = tint.Scale(col, r.bright)
			}
			c.FG = col
		}
	}
}
