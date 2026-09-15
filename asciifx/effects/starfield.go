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
		Name:        "starfield",
		Title:       "Warp Starfield",
		Description: "A 3D warp-speed starfield: distant stars glint as single braille dots, then stretch into bright streaks and flare as they rush past the camera.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "space", "3d"},
		Glyphs:      []string{"braille", "ascii"},
		FPS:         30,
		MinW:        4,
		MinH:        2,
		Params: []fx.Param{
			fx.PaletteParam("palette", "ice", "Star colours, far (dim) to near (bright)."),
			fx.FloatParam("speed", 1, 0.05, 5, "Warp speed."),
			fx.FloatParam("density", 1, 0.1, 4, "Star count multiplier."),
		},
		Example: "asciifx play starfield -p speed=2.5 -p palette=synthwave",
		New:     newStarfield,
	})
}

type star struct{ x, y, z, hue float64 }

type starfield struct {
	pal   tint.Gradient
	speed float64
	stars []star
	d     *dots
	rng   *rand.Rand
	ry    float64 // vertical spawn range matching the screen aspect
}

func newStarfield(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	n := int(p.Float("density") * float64(w*h) / 11)
	s := &starfield{
		pal:   p.Palette("palette"),
		speed: p.Float("speed"),
		stars: make([]star, max(n, 4)),
		d:     newDots(w, h),
		rng:   rng,
		ry:    math.Max(2*float64(h)/float64(w), 0.1),
	}
	for i := range s.stars {
		s.respawn(&s.stars[i])
		s.stars[i].z = 0.05 + rng.Float64()*0.95
	}
	return s, nil
}

func (s *starfield) respawn(st *star) {
	st.x = (s.rng.Float64()*2 - 1) * 0.95
	st.y = (s.rng.Float64()*2 - 1) * s.ry * 1.1
	st.z = 0.85 + s.rng.Float64()*0.15
	st.hue = s.rng.Float64()
}

// project maps a star to sub-pixel coordinates; cells are twice as tall as
// wide, so vertical offsets are halved in cell units.
func (s *starfield) project(x, y, z float64) (float64, float64) {
	W, H := float64(s.d.w), float64(s.d.h)
	f := W / 2
	cx := W/2 + x/z*f
	cy := H/2 + y/z*f/2
	return cx * 2, cy * 4
}

func (s *starfield) Step(fr *fx.Frame) {
	d := s.d
	d.clear()
	dz := s.speed * fr.Dt * 0.35
	type flare struct {
		x, y int
		c    tint.Color
		r    rune
	}
	var flares []flare
	for i := range s.stars {
		st := &s.stars[i]
		st.z -= dz
		px, py := s.project(st.x, st.y, st.z)
		if st.z < 0.03 || px < 0 || py < 0 || px >= float64(d.w*2) || py >= float64(d.h*4) {
			s.respawn(st)
			continue
		}
		near := ambClamp((1 - st.z) / 0.95)
		bright := math.Pow(near, 1.8)
		head := s.pal.At(0.25 + 0.75*bright)
		head = tint.Lerp(head, s.pal.Cyclic(st.hue), 0.15)
		tail := st.z + math.Min(dz*(0.5+9*near*near), 0.2)
		tx, ty := s.project(st.x, st.y, tail)
		d.line(tx, ty, px, py, func(t float64) (tint.Color, float64) {
			return tint.Lerp(s.pal.At(0.15+0.4*bright), head, t), bright * (0.5 + 0.5*t)
		})
		if near > 0.82 {
			r := '+'
			if near > 0.92 {
				r = '*'
			}
			flares = append(flares, flare{int(px / 2), int(py / 4), tint.Lerp(head, tint.RGB(255, 255, 255), 0.5), r})
		}
	}
	d.draw(fr.Buf)
	for _, f := range flares {
		fr.Buf.Set(f.x, f.y, cell.Cell{Rune: f.r, FG: f.c, Attr: cell.Bold})
	}
}
