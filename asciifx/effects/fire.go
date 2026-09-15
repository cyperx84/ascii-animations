package effects

import (
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "fire",
		Title:       "Doom Fire",
		Description: "The PSX Doom fire algorithm at double vertical resolution using half-block cells, each carrying two colours.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "classic"},
		Glyphs:      []string{"halfblock"},
		FPS:         30,
		MinW:        4,
		MinH:        2,
		Params: []fx.Param{
			fx.PaletteParam("palette", "fire", "Heat colours, coldest first."),
			fx.FloatParam("cooling", 1, 0.2, 4, "How fast flames die out. Lower gives taller flames."),
			fx.FloatParam("wind", 0, -1, 1, "Sideways drift."),
		},
		Example: "asciifx play fire -p palette=synthwave -p cooling=0.7",
		New:     newFire,
	})
}

const fireLevels = 37

type fire struct {
	w, h    int // pixel grid: h is twice the cell rows
	heat    []uint8
	colors  []tint.Color
	cooling float64
	wind    float64
	rng     *rand.Rand
}

func newFire(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	f := &fire{
		w: w, h: h * 2,
		colors:  p.Palette("palette").Steps(fireLevels),
		cooling: p.Float("cooling"),
		wind:    p.Float("wind"),
		rng:     rng,
	}
	f.heat = make([]uint8, f.w*f.h)
	for x := 0; x < f.w; x++ {
		f.heat[(f.h-1)*f.w+x] = fireLevels - 1
	}
	return f, nil
}

func (f *fire) Step(fr *fx.Frame) {
	// Scale cooling to the height so flames reach about 60% of the screen
	// whatever the terminal size.
	meanDecay := float64(fireLevels) / (float64(f.h) * 0.6) * f.cooling
	for y := 0; y < f.h-1; y++ {
		for x := 0; x < f.w; x++ {
			below := f.heat[(y+1)*f.w+x]
			r := f.rng.Float64()
			decay := int(r*2*meanDecay + 0.5)
			dx := x - int(r*3) + 1 + int(f.wind*f.rng.Float64()*2)
			dx = (dx%f.w + f.w) % f.w
			v := int(below) - decay
			f.heat[y*f.w+dx] = uint8(max(v, 0))
		}
	}
	for y := 0; y < fr.Buf.H; y++ {
		for x := 0; x < fr.Buf.W; x++ {
			top, bot := f.heat[y*2*f.w+x], f.heat[(y*2+1)*f.w+x]
			fr.Buf.Set(x, y, HalfBlock(f.color(top), f.color(bot)))
		}
	}
}

func (f *fire) color(v uint8) tint.Color {
	if v == 0 {
		return tint.None
	}
	return f.colors[v]
}
