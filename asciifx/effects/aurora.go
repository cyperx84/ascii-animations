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
		Name:        "aurora",
		Title:       "Aurora",
		Description: "Northern lights: slow curtains of light fold and ripple across a starlit sky, bright at their hem and dissolving upward through the palette.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "calm", "sky"},
		Glyphs:      []string{"halfblock", "ascii"},
		FPS:         24,
		MinW:        4,
		MinH:        3,
		Params: []fx.Param{
			fx.PaletteParam("palette", "aurora", "Sky-to-glow colours; the hem of each curtain takes the brightest."),
			fx.FloatParam("speed", 1, 0, 5, "Drift speed multiplier."),
			fx.IntParam("curtains", 3, 1, 5, "Layered curtains."),
			fx.BoolParam("stars", true, "Faint twinkling stars behind the curtains."),
		},
		Example: "asciifx play aurora -p palette=synthwave -p curtains=4",
		New:     newAurora,
	})
}

type curtain struct {
	base, f1, f2, f3, p1, p2, p3, drift float64
}

type aurora struct {
	pal      tint.Gradient
	speed    float64
	stars    bool
	curtains []curtain
	seed     uint64
	light    []float64 // per pixel intensity
	tone     []float64 // per pixel palette position
}

func newAurora(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	a := &aurora{
		pal:      p.Palette("palette"),
		speed:    p.Float("speed"),
		stars:    p.Bool("stars"),
		curtains: make([]curtain, p.Int("curtains")),
		seed:     rng.Uint64(),
		light:    make([]float64, w*h*2),
		tone:     make([]float64, w*h*2),
	}
	for i := range a.curtains {
		c := &a.curtains[i]
		n := float64(len(a.curtains))
		c.base = 0.38 + 0.3*(float64(i)+0.5)/n + (rng.Float64()-0.5)*0.08
		c.f1 = 1.2 + rng.Float64()*1.5
		c.f2 = 3 + rng.Float64()*3
		c.f3 = 9 + rng.Float64()*8
		c.p1, c.p2, c.p3 = rng.Float64()*6.3, rng.Float64()*6.3, rng.Float64()*6.3
		c.drift = (0.15 + rng.Float64()*0.2) * float64(1-2*(i%2))
	}
	return a, nil
}

func (a *aurora) Step(fr *fx.Frame) {
	b := fr.Buf
	w, ph := b.W, b.H*2
	t := fr.T() * a.speed
	for i := range a.light {
		a.light[i], a.tone[i] = 0, 0
	}
	for ci, c := range a.curtains {
		depth := 1 - 0.25*float64(ci)/math.Max(1, float64(len(a.curtains)-1))
		for x := 0; x < w; x++ {
			u := float64(x) / float64(w)
			// Hem height folds on three scales, each drifting on its own.
			hem := c.base +
				0.12*math.Sin(u*c.f1*math.Pi+t*c.drift+c.p1) +
				0.05*math.Sin(u*c.f2*math.Pi-t*c.drift*1.7+c.p2) +
				0.015*math.Sin(u*c.f3*math.Pi+t*0.9+c.p3)
			hemY := hem * float64(ph)
			// Brightness along the curtain: broad bands with fine rays.
			band := 0.5 + 0.5*math.Sin(u*c.f2*2.1+t*c.drift*2.3+c.p1)
			// Fine vertical rays shimmer along the curtain.
			ray := 0.5 + 0.5*math.Sin(u*float64(w)*0.55+2*math.Sin(t*0.6+u*7+c.p3))
			ray2 := 0.5 + 0.5*math.Sin(u*float64(w)*1.3-t*0.8+c.p2)
			rays := 0.35 + 0.65*ray*ray*(0.6+0.4*ray2)
			glow := ambSmooth(band*1.7-0.55) * rays * depth
			if glow <= 0.01 {
				continue
			}
			tall := float64(ph) * (0.12 + 0.22*band*ray)
			for y := 0; y < ph; y++ {
				dy := hemY - float64(y)
				var v float64
				if dy >= 0 { // above the hem: rays fade upward
					v = math.Exp(-dy / tall * 2.6)
				} else { // below: a sharp edge
					v = math.Exp(-dy * dy / 3)
				}
				v *= glow
				i := y*w + x
				if v > a.light[i] {
					// Tone rises from hem colour to upper colour.
					a.tone[i] = ambClamp(dy/tall) * 0.5
				}
				a.light[i] = math.Min(1, a.light[i]+v*(1-a.light[i]*0.5))
			}
		}
	}
	pixel := func(x, y int) tint.Color {
		i := y*w + x
		v := a.light[i]
		if v < 0.06 {
			return tint.None
		}
		// Brightness picks the band of the palette, height nudges it cooler.
		pos := 0.15 + 0.85*math.Pow(ambClamp(v), 0.85) - 0.3*a.tone[i]
		return a.pal.At(ambClamp(pos))
	}
	for y := 0; y < b.H; y++ {
		for x := 0; x < w; x++ {
			top, bot := pixel(x, y*2), pixel(x, y*2+1)
			if a.stars && !top.Valid && !bot.Valid && y < b.H*2/3 {
				s := fx.Hash01(x, y, a.seed)
				if s > 0.985 {
					tw := 0.5 + 0.5*math.Sin(t*2+s*400)
					b.Set(x, y, cell.Cell{Rune: '.', FG: a.pal.At(0.35 + 0.5*tw)})
					continue
				}
			}
			b.Set(x, y, HalfBlock(top, bot))
		}
	}
}
