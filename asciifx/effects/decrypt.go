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
		Name:  "decrypt",
		Title: "Decrypt",
		Description: "The No More Secrets movie hack: every character churns through dim cipher noise, " +
			"then locks into place along a pattern, each one flashing white-hot before cooling to its colour.",
		Kind:     fx.Transition,
		Content:  true,
		Tags:     []string{"intro", "hacker", "splash", "banner"},
		Glyphs:   []string{"ascii"},
		FPS:      30,
		Duration: 2.4,
		MinW:     1,
		MinH:     1,
		Params: []fx.Param{
			fx.FloatParam("duration", 2.4, 0.2, 30, "Seconds for the whole decrypt."),
			fx.PatternParamOf("pattern", "left", "Order characters resolve in."),
			fx.FloatParam("scramble", 0.3, 0, 0.9, "Fraction of the run spent fully scrambled before anything resolves."),
			fx.FloatParam("softness", 0.2, 0.01, 1, "How many characters are mid-resolve at once."),
			fx.FloatParam("jitter", 0.25, 0, 1, "Randomness mixed into the resolve order, so the edge is ragged."),
			fx.PaletteParam("palette", "ocean", "Cipher colours, dim to bright; the brightest stop is the flash."),
			fx.EnumParam("final", "gradient", []string{"content", "gradient"}, "Resting colour: the content's own colour, or the palette laid across it."),
			fx.EnumParam("direction", "horizontal", tint.Directions, "Direction of the final gradient."),
			fx.EasingParam("easing", "in-out-sine", "Progress curve of the resolve phase."),
			fx.StringParam("noise", `!@#$%^&*<>?/\|{}[]=+-~;:0123456789abcdefABCDEF`, "Cipher glyphs characters churn through. Single-width only."),
		},
		Example: `asciifx play decrypt --text "ACCESS GRANTED"`,
		New:     newDecrypt,
	})
}

type decrypt struct {
	dur, scramble, soft, jitter float64
	pattern                     fx.Pattern
	pal                         tint.Gradient
	useGradient                 bool
	dir                         tint.Direction
	ease                        ease.Func
	noise                       []rune
	seed                        uint64
}

func newDecrypt(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	seed := rng.Uint64()
	return &decrypt{
		dur:         p.Float("duration"),
		scramble:    p.Float("scramble"),
		soft:        p.Float("softness"),
		jitter:      p.Float("jitter"),
		pattern:     p.Pattern("pattern", seed),
		pal:         p.Palette("palette"),
		useGradient: p.String("final") == "gradient",
		dir:         tint.Direction(p.String("direction")),
		ease:        p.Easing("easing"),
		noise:       safeRunes(p.String("noise"), "#%&*+=<>?"),
		seed:        seed,
	}, nil
}

func (d *decrypt) Duration() float64 { return d.dur }

func (d *decrypt) Step(f *fx.Frame) {
	b := f.Buf
	p := tickProgress(f, d.dur)
	bx, by, w, h := fx.Bounds(b)
	if p >= 1 {
		settle(b, d.pal, d.useGradient, d.dir)
		return
	}
	q := 0.0
	if p > d.scramble {
		q = d.ease((p - d.scramble) / (1 - d.scramble))
	}
	flash := tint.Lerp(d.pal.At(1), tint.RGB(255, 255, 255), 0.6)
	for y := by; y < by+h; y++ {
		for x := bx; x < bx+w; x++ {
			c := b.At(x, y)
			if !fx.Ink(c) {
				continue
			}
			lx, ly := x-bx, y-by
			final := restColor(c.FG, d.pal, d.useGradient, d.dir, lx, ly, w, h)
			order := d.pattern.Order(lx, ly, w, h)*(1-d.jitter) + fx.Hash01(x, y, d.seed)*d.jitter
			local := fx.Local(q, order, d.soft)
			if local > 0 {
				c.FG = tint.Lerp(flash, final, ease.OutCubic(local))
				continue
			}
			// Characters blink into existence over the first moments.
			if p < fx.Hash01(x, y, d.seed^0xA5)*0.1 {
				*c = cell.Blank
				continue
			}
			// Each cell churns on its own clock so the noise never flips in
			// lockstep, and churns faster as its turn approaches.
			near := math.Max(0, 1-(order-q*(1+d.soft))/0.25)
			period := 3 - int(near*2)
			slot := (f.Tick + int(fx.Hash01(x, y, d.seed^0x5A)*7)) / max(period, 1)
			n := fx.Hash01(x, y^(slot*7919), d.seed)
			c.Rune = d.noise[min(int(n*float64(len(d.noise))), len(d.noise)-1)]
			dim := d.pal.At(0.15 + 0.3*fx.Hash01(x^slot, y, d.seed^0x77))
			c.FG = tint.Lerp(dim, d.pal.At(0.75), near*near*0.6)
		}
	}
}

// tickProgress is elapsed/duration measured in whole ticks, so the last
// frame a Run shows (tick round(duration·fps)) is always exactly 1 even when
// duration·fps is not an integer.
func tickProgress(f *fx.Frame, duration float64) float64 {
	total := int(duration/f.Dt + 0.5)
	if total <= 0 {
		return 1
	}
	return math.Max(0, math.Min(1, float64(f.Tick)/float64(total)))
}

// restColor is where a cell settles: its own colour, or the palette's upper
// range laid across the content bounds.
func restColor(own tint.Color, pal tint.Gradient, gradient bool, dir tint.Direction, lx, ly, w, h int) tint.Color {
	if gradient || !own.Valid {
		return pal.At(0.35 + 0.65*tint.Position(dir, lx, ly, w, h))
	}
	return own
}

// settle draws the finished frame: the content, recoloured to rest.
func settle(b *cell.Buffer, pal tint.Gradient, gradient bool, dir tint.Direction) {
	bx, by, w, h := fx.Bounds(b)
	for y := by; y < by+h; y++ {
		for x := bx; x < bx+w; x++ {
			if c := b.At(x, y); fx.Ink(c) {
				c.FG = restColor(c.FG, pal, gradient, dir, x-bx, y-by, w, h)
			}
		}
	}
}
