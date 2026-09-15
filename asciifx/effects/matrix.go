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
		Name:        "matrix",
		Title:       "Digital Rain",
		Description: "Cascading code columns, each at its own speed, with a white-hot head and a trail that mutates and decays through the palette into the background.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "classic", "hacker"},
		Glyphs:      []string{"ascii"},
		FPS:         30,
		MinW:        2,
		MinH:        3,
		Params: []fx.Param{
			fx.PaletteParam("palette", "matrix", "Trail colours, darkest first; the head glows past the brightest stop."),
			fx.FloatParam("speed", 1, 0.1, 5, "Fall speed multiplier."),
			fx.FloatParam("density", 0.6, 0.05, 1, "Fraction of columns raining at once."),
			fx.FloatParam("mutate", 1, 0, 5, "How often trail glyphs flicker to a new symbol."),
		},
		Example: "asciifx play matrix -p palette=ice -p density=0.9",
		New:     newMatrix,
	})
}

const matrixGlyphs = "0123456789ABCDEFHKLMNPRTUVXZ$%&*+=<>:;|/\\{}[]?#@"

type matrixCol struct {
	y, speed, length, wait float64
	head                   int
}

type matrix struct {
	w, h    int
	pal     tint.Gradient
	head    tint.Color
	speed   float64
	density float64
	mutate  float64
	cols    []matrixCol
	glyph   []rune
	life    []float64
	fade    []float64 // per-cell decay per second, from the column that lit it
	rng     *rand.Rand
}

func newMatrix(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	m := &matrix{
		w: w, h: h,
		pal:     p.Palette("palette"),
		speed:   p.Float("speed"),
		density: p.Float("density"),
		mutate:  p.Float("mutate"),
		cols:    make([]matrixCol, w),
		glyph:   make([]rune, w*h),
		life:    make([]float64, w*h),
		fade:    make([]float64, w*h),
		rng:     rng,
	}
	m.head = tint.Lerp(m.pal.At(1), tint.RGB(255, 255, 255), 0.65)
	for i := range m.glyph {
		m.glyph[i] = m.randGlyph()
	}
	for x := range m.cols {
		m.spawn(&m.cols[x])
		// Stagger the start so the screen is already raining at tick 0.
		m.cols[x].wait *= 0.3
		m.cols[x].y = -rng.Float64() * float64(h)
	}
	return m, nil
}

func (m *matrix) randGlyph() rune { return rune(matrixGlyphs[m.rng.IntN(len(matrixGlyphs))]) }

func (m *matrix) spawn(c *matrixCol) {
	c.speed = float64(m.h) * (0.3 + m.rng.Float64()*0.7) * m.speed // 0.3–1 screens per second
	c.speed = math.Max(c.speed, 4*m.speed)
	c.length = float64(m.h) * (0.35 + 0.65*m.rng.Float64())
	c.length = math.Max(c.length, 3)
	c.y = -m.rng.Float64() * 3
	c.head = -1
	cycle := (float64(m.h) + c.length) / c.speed
	c.wait = m.rng.Float64() * cycle * 2 * (1/m.density - 1)
}

func (m *matrix) Step(fr *fx.Frame) {
	dt := fr.Dt
	for i := range m.life {
		if m.life[i] <= 0 {
			continue
		}
		m.life[i] -= dt * m.fade[i]
		if m.rng.Float64() < m.mutate*dt*0.6 {
			m.glyph[i] = m.randGlyph()
		}
	}
	for x := range m.cols {
		c := &m.cols[x]
		if c.wait > 0 {
			c.wait -= dt
			continue
		}
		prev := c.y
		c.y += c.speed * dt
		for yy := int(math.Ceil(prev)); float64(yy) <= c.y; yy++ {
			if yy >= 0 && yy < m.h {
				i := yy*m.w + x
				m.life[i] = 1
				m.fade[i] = c.speed / c.length
				m.glyph[i] = m.randGlyph()
			}
		}
		c.head = int(math.Floor(c.y))
		if c.y-c.length > float64(m.h) {
			m.spawn(c)
		}
	}

	b := fr.Buf
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			i := y*m.w + x
			l := m.life[i]
			if l <= 0.02 {
				b.Set(x, y, cell.Blank)
				continue
			}
			// Ease the trail so it holds brightness then falls away, and
			// drift the hue slightly with height for depth.
			v := math.Pow(l, 1.6)
			col := m.pal.At(0.18 + 0.72*v - 0.06*float64(y)/float64(m.h))
			b.Set(x, y, cell.Cell{Rune: m.glyph[i], FG: col})
		}
	}
	for x := range m.cols {
		c := &m.cols[x]
		if c.wait <= 0 && c.head >= 0 && c.head < m.h {
			b.Set(x, c.head, cell.Cell{Rune: m.glyph[c.head*m.w+x], FG: m.head, Attr: cell.Bold})
		}
	}
}
