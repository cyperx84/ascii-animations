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
		Name:  "spinner",
		Title: "Spinner",
		Description: "A one-line activity indicator: a classic spinner glyph cycling through palette colours, " +
			"with an optional label that a soft highlight shimmers across.",
		Kind:   fx.Spinner,
		Tags:   []string{"loading", "progress", "cli", "inline"},
		Glyphs: []string{"ascii", "box", "block", "braille"},
		FPS:    30,
		MinW:   1,
		MinH:   1,
		DefW:   30,
		DefH:   1,
		Params: []fx.Param{
			fx.EnumParam("style", "dots", spinnerStyleNames(), "Spinner frame set."),
			fx.StringParam("label", "Loading", "Text after the spinner; empty for the glyph alone."),
			fx.FloatParam("speed", 1, 0.1, 10, "Frame rate multiplier."),
			fx.PaletteParam("palette", "catppuccin", "Colours the glyph cycles through and the label shimmers with."),
			fx.FloatParam("shimmer", 1.4, 0, 10, "Seconds for the label highlight to cross; 0 turns it off."),
			fx.ColorParamOf("label_color", "#a6adc8", "Resting label colour; none uses the terminal default."),
		},
		Example: `asciifx play spinner -p style=arc -p label="Compiling assets"`,
		New:     newSpinner,
	})
}

type spinner struct {
	style      spinnerStyle
	label      []rune
	speed      float64
	shimmer    float64
	pal        tint.Gradient
	labelColor tint.Color
}

func newSpinner(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	var label []rune
	if l := p.String("label"); l != "" {
		label = safeRunes(l, "?")
	}
	return &spinner{
		style:      spinnerStyles[p.String("style")],
		label:      label,
		speed:      p.Float("speed"),
		shimmer:    p.Float("shimmer"),
		pal:        p.Palette("palette"),
		labelColor: p.Color("label_color"),
	}, nil
}

func (s *spinner) Step(f *fx.Frame) {
	b := f.Buf
	// Spinners do not get content copied in, so start clean every tick.
	b.Clear()
	y := (b.H - 1) / 2
	t := f.T() * s.speed
	frames := s.style.frames
	frame := []rune(frames[int(t/s.style.interval)%len(frames)])
	// The glyph drifts through the palette's bright half.
	glyphColor := s.pal.Cyclic(f.T() * 0.25)
	glyphColor = tint.Lerp(glyphColor, s.pal.At(1), 0.35)
	x := 0
	for _, r := range frame {
		b.Set(x, y, cell.Cell{Rune: r, FG: glyphColor})
		x++
	}
	if len(s.label) == 0 {
		return
	}
	x++
	n := len(s.label)
	// The shimmer band crosses the label, then rests for half a crossing.
	var center float64
	if s.shimmer > 0 {
		cycle := math.Mod(f.T()/(s.shimmer*1.5), 1) * 1.5
		center = -3 + cycle*float64(n+6)
	}
	for i, r := range s.label {
		base := s.labelColor
		if !base.Valid {
			base = s.pal.At(0.6)
		}
		fg := base
		if s.shimmer > 0 && r != ' ' {
			d := float64(i) - center
			k := math.Exp(-d * d / 4.5)
			hi := tint.Lerp(s.pal.Cyclic(float64(i)/float64(max(n, 1))*0.5+f.T()*0.1), tint.RGB(255, 255, 255), 0.45)
			fg = tint.Lerp(base, hi, k)
		}
		if b.In(x, y) {
			b.Set(x, y, cell.Cell{Rune: r, FG: fg})
		}
		x++
	}
}
