package effects

import (
	"math"
	"math/rand/v2"
	"sort"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "snow",
		Title:       "Snowfall",
		Description: "Layered snowfall: flakes at three depths sway on their own sine drift and glint as they fall, slowly building soft drifts along the ground.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "weather", "calm", "winter"},
		Glyphs:      []string{"ascii", "braille", "block"},
		FPS:         30,
		MinW:        3,
		MinH:        4,
		Params: []fx.Param{
			fx.PaletteParam("palette", "ice", "Flake colours, far (dim) to near (bright)."),
			fx.FloatParam("density", 1, 0.1, 4, "Flake count multiplier."),
			fx.FloatParam("speed", 1, 0.1, 4, "Fall speed multiplier."),
			fx.FloatParam("wind", 0, -1, 1, "Steady sideways drift."),
			fx.BoolParam("accumulate", true, "Let snow pile up along the bottom."),
		},
		Example: "asciifx play snow -p wind=0.3 -p density=2",
		New:     newSnow,
	})
}

type flake struct {
	x, y, depth, phase, freq, amp float64
}

type snow struct {
	w, h       int
	pal        tint.Gradient
	speed      float64
	wind       float64
	accumulate bool
	flakes     []flake
	pile       []float64 // snow depth per column, in cells
	cap        float64
	seed       uint64
	rng        *rand.Rand
}

func newSnow(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	n := int(p.Float("density") * float64(w*h) / 12)
	s := &snow{
		w: w, h: h,
		pal:        p.Palette("palette"),
		speed:      p.Float("speed"),
		wind:       p.Float("wind"),
		accumulate: p.Bool("accumulate"),
		flakes:     make([]flake, max(n, 3)),
		pile:       make([]float64, w),
		cap:        math.Max(1, float64(h)/5),
		seed:       rng.Uint64(),
		rng:        rng,
	}
	for i := range s.flakes {
		f := &s.flakes[i]
		f.depth = rng.Float64()
		s.respawn(f)
		f.y = rng.Float64() * float64(h)
	}
	sort.Slice(s.flakes, func(i, j int) bool { return s.flakes[i].depth < s.flakes[j].depth })
	return s, nil
}

func (s *snow) respawn(f *flake) {
	f.x = s.rng.Float64() * float64(s.w)
	f.y = -s.rng.Float64() * 2
	f.phase = s.rng.Float64() * 2 * math.Pi
	f.freq = 0.6 + s.rng.Float64()*1.2
	f.amp = 0.4 + f.depth*1.6
}

func (s *snow) Step(fr *fx.Frame) {
	b := fr.Buf
	b.Clear()
	dt, t := fr.Dt, fr.T()
	W, H := float64(s.w), float64(s.h)
	var near []int
	for i := range s.flakes {
		f := &s.flakes[i]
		fall := (1.2 + 4.5*f.depth) * s.speed * math.Max(1, H/20)
		f.y += fall * dt
		f.x += s.wind * (1.5 + 6*f.depth) * dt
		f.x = math.Mod(f.x+W, W)
		sx := math.Mod(f.x+f.amp*math.Sin(t*f.freq+f.phase)+W, W)
		cx, cy := min(int(sx), s.w-1), int(f.y)
		ground := H
		if s.accumulate && f.depth > 0.45 {
			ground = H - s.pile[cx]
		}
		if f.y >= ground {
			if s.accumulate && f.depth > 0.45 {
				s.land(cx, 0.08+0.1*f.depth)
			}
			s.respawn(f)
			continue
		}
		if f.depth > 0.7 {
			near = append(near, i)
			continue
		}
		s.drawFlake(b, f, cx, cy, t)
	}
	if s.accumulate {
		s.drawPile(b, t)
	}
	for _, i := range near {
		f := &s.flakes[i]
		sx := math.Mod(f.x+f.amp*math.Sin(t*f.freq+f.phase)+W, W)
		s.drawFlake(b, f, int(sx), int(f.y), t)
	}
}

func (s *snow) drawFlake(b *cell.Buffer, f *flake, x, y int, t float64) {
	if y < 0 {
		return
	}
	r := '.'
	switch {
	case f.depth > 0.85:
		r = '*'
	case f.depth > 0.6:
		r = '+'
	case f.depth > 0.3:
		r = '⠂' // braille dot: '·' is ambiguous-width under CJK locales
	}
	// Each flake glints on its own slow cycle.
	glint := 0.5 + 0.5*math.Sin(t*(1.5+f.freq)+f.phase*3)
	v := 0.25 + 0.6*f.depth + 0.15*glint*f.depth
	c := s.pal.At(v)
	if f.depth > 0.85 && glint > 0.9 {
		c = tint.Lerp(c, tint.RGB(255, 255, 255), 0.6)
	}
	b.Set(x, y, cell.Cell{Rune: r, FG: c})
}

// land adds snow at column x and lets it slump into lower neighbours so the
// drift keeps a soft profile.
func (s *snow) land(x int, amt float64) {
	s.pile[x] = math.Min(s.cap, s.pile[x]+amt)
	for pass := 0; pass < 2; pass++ {
		for i := range s.pile {
			for _, j := range [2]int{i - 1, i + 1} {
				if j < 0 || j >= s.w {
					continue
				}
				if d := s.pile[i] - s.pile[j]; d > 0.6 {
					s.pile[i] -= d * 0.25
					s.pile[j] += d * 0.25
				}
			}
		}
	}
}

func (s *snow) drawPile(b *cell.Buffer, t float64) {
	const eighths = " ▁▂▃▄▅▆▇█"
	levels := []rune(eighths)
	for x := 0; x < s.w; x++ {
		p := s.pile[x]
		full := int(p)
		part := int((p - float64(full)) * 8)
		for k := 0; k <= full && k < s.h; k++ {
			y := s.h - 1 - k
			r := '█'
			if k == full {
				if part == 0 {
					continue
				}
				r = levels[part]
			}
			// Deep snow is shadowed; the surface catches light.
			depth := (p - float64(k)) / s.cap
			sparkle := fx.Hash01(x, k, s.seed+uint64(t*2))
			v := 0.95 - 0.35*ambClamp(depth)
			if sparkle > 0.97 && k == full-1+min(part, 1) {
				v = 1
			}
			b.Set(x, y, cell.Cell{Rune: r, FG: s.pal.At(v)})
		}
	}
}
