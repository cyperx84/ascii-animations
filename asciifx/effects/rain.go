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
		Name:        "rain",
		Title:       "Parallax Rain",
		Description: "Rain in three depths of parallax: far drops drift down as faint dashes, near ones slash past bright and fast and burst into splashes on the ground.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "weather", "calm"},
		Glyphs:      []string{"box", "ascii"},
		FPS:         30,
		MinW:        3,
		MinH:        4,
		Params: []fx.Param{
			fx.PaletteParam("palette", "ocean", "Drop colours, far (dark) to near (bright)."),
			fx.FloatParam("density", 1, 0.1, 4, "Drop count multiplier."),
			fx.FloatParam("speed", 1, 0.1, 4, "Fall speed multiplier."),
			fx.FloatParam("wind", 0, -1, 1, "Slant: negative blows left, positive right."),
			fx.BoolParam("splash", true, "Near drops splash on the bottom row."),
		},
		Example: "asciifx play rain -p palette=nord -p density=2 -p wind=0.4",
		New:     newRain,
	})
}

type drop struct {
	x, y, speed, length, depth float64
}

type splash struct {
	x   int
	age float64
	col tint.Color
}

type rain struct {
	w, h     int
	pal      tint.Gradient
	speed    float64
	wind     float64
	splashOn bool
	drops    []drop
	splashes []splash
	rng      *rand.Rand
}

func newRain(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	n := int(p.Float("density") * float64(w*h) / 24)
	r := &rain{
		w: w, h: h,
		pal:      p.Palette("palette"),
		speed:    p.Float("speed"),
		wind:     p.Float("wind"),
		splashOn: p.Bool("splash"),
		drops:    make([]drop, max(n, 3)),
		rng:      rng,
	}
	for i := range r.drops {
		d := &r.drops[i]
		d.depth = math.Pow(rng.Float64(), 1.6) // most rain is far away
		r.respawn(d)
		d.y = rng.Float64() * float64(h+int(d.length))
	}
	// Far drops draw first so near ones pass in front.
	sort.Slice(r.drops, func(i, j int) bool { return r.drops[i].depth < r.drops[j].depth })
	return r, nil
}

func (r *rain) respawn(d *drop) {
	d.speed = (10 + 38*d.depth*d.depth) * r.speed * (0.9 + 0.2*r.rng.Float64()) * math.Max(1, float64(r.h)/16)
	d.length = 1 + d.depth*math.Min(4, float64(r.h)/4)
	d.y = -r.rng.Float64() * 4
	d.x = r.rng.Float64()*float64(r.w+r.h)*math.Abs(r.wind) + r.rng.Float64()*float64(r.w)
	if r.wind > 0 {
		d.x -= float64(r.h) * r.wind
	}
}

func (r *rain) Step(fr *fx.Frame) {
	b := fr.Buf
	b.Clear()
	dt := fr.Dt
	white := tint.RGB(255, 255, 255)
	for i := range r.drops {
		d := &r.drops[i]
		d.y += d.speed * dt
		d.x += r.wind * d.speed * dt * 0.5
		if d.y >= float64(r.h) {
			if r.splashOn && d.depth > 0.6 && d.x >= 0 && d.x < float64(r.w) {
				r.splashes = append(r.splashes, splash{x: int(d.x), col: r.pal.At(0.55 + 0.45*d.depth)})
			}
			r.respawn(d)
			continue
		}
		glyph := '┆'
		switch {
		case d.depth > 0.66:
			glyph = '│'
		case d.depth > 0.33:
			glyph = '╎'
		}
		// Screen y grows downward, so a rightward wind leans drops like '\'.
		if r.wind > 0.3 {
			glyph = '\\'
		} else if r.wind < -0.3 {
			glyph = '/'
		}
		base := 0.2 + 0.75*d.depth
		n := int(math.Ceil(d.length))
		for k := 0; k < n; k++ {
			y := int(d.y) - k
			x := int(d.x - r.wind*float64(k)*0.5)
			if y < 0 || !b.In(x, y) {
				continue
			}
			f := 1 - float64(k)/float64(n) // 1 at the head
			c := r.pal.At(base * (0.45 + 0.55*f))
			if k == 0 && d.depth > 0.75 {
				c = tint.Lerp(c, white, 0.35)
			}
			b.Set(x, y, cell.Cell{Rune: glyph, FG: c})
		}
	}
	live := r.splashes[:0]
	for _, s := range r.splashes {
		s.age += dt
		const life = 0.4
		p := s.age / life
		if p >= 1 {
			continue
		}
		live = append(live, s)
		c := tint.Lerp(tint.Lerp(s.col, white, 0.3), r.pal.At(0.25), ambSmooth(p))
		y := r.h - 1
		if p < 0.3 {
			b.Set(s.x, y, cell.Cell{Rune: 'o', FG: c})
			continue
		}
		spread := 1 + int(p*2.5)
		lift := y - int(math.Round(math.Sin(p*math.Pi)))
		g := '\''
		if lift == y {
			g = '.'
		}
		b.Set(s.x-spread, lift, cell.Cell{Rune: g, FG: c})
		b.Set(s.x+spread, lift, cell.Cell{Rune: g, FG: c})
		if p < 0.6 {
			b.Set(s.x, y, cell.Cell{Rune: '_', FG: r.pal.At(0.3)})
		}
	}
	r.splashes = live
}
