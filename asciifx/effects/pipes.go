package effects

import (
	"math/rand/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func init() {
	fx.Register(fx.Spec{
		Name:        "pipes",
		Title:       "Pipes",
		Description: "The pipes.sh screensaver: glowing walkers lay down box-drawing pipes in their own colours, turning at crisp corners until the screen fills and fades away.",
		Kind:        fx.Ambient,
		Tags:        []string{"background", "loop", "classic", "screensaver"},
		Glyphs:      []string{"box"},
		FPS:         30,
		MinW:        4,
		MinH:        3,
		Params: []fx.Param{
			fx.PaletteParam("palette", "rainbow", "Each pipe takes its colour from somewhere along this palette."),
			fx.IntParam("pipes", 4, 1, 16, "Pipes drawn at once."),
			fx.FloatParam("speed", 40, 2, 300, "Segments per second, per pipe."),
			fx.EnumParam("style", "rounded", []string{"light", "heavy", "rounded", "double"}, "Box-drawing style."),
		},
		Example: "asciifx play pipes -p style=heavy -p pipes=8 -p palette=dracula",
		New:     newPipes,
	})
}

// pipeGlyphs indexed by style then by the two sides joined: up=1 right=2
// down=4 left=8.
var pipeGlyphs = map[string]map[int]rune{
	"light":   {5: '│', 10: '─', 6: '┌', 12: '┐', 3: '└', 9: '┘'},
	"heavy":   {5: '┃', 10: '━', 6: '┏', 12: '┓', 3: '┗', 9: '┛'},
	"rounded": {5: '│', 10: '─', 6: '╭', 12: '╮', 3: '╰', 9: '╯'},
	"double":  {5: '║', 10: '═', 6: '╔', 12: '╗', 3: '╚', 9: '╝'},
}

var pipeDX, pipeDY = [4]int{0, 1, 0, -1}, [4]int{-1, 0, 1, 0}

const pipeTrail = 6

type pipe struct {
	x, y, dir int
	hue       float64
	n         int
	recent    [pipeTrail]int // recent cell indices, newest first; -1 empty
}

type pipes struct {
	w, h   int
	pal    tint.Gradient
	glyphs map[int]rune
	speed  float64
	walk   []pipe
	grid   []cell.Cell
	filled int
	acc    float64
	fading float64 // seconds into the fade-out, 0 when not fading
	rng    *rand.Rand
}

func newPipes(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	e := &pipes{
		w: w, h: h,
		pal:    p.Palette("palette"),
		glyphs: pipeGlyphs[p.String("style")],
		speed:  p.Float("speed"),
		walk:   make([]pipe, p.Int("pipes")),
		grid:   make([]cell.Cell, w*h),
		rng:    rng,
	}
	e.reset()
	return e, nil
}

func (e *pipes) reset() {
	for i := range e.grid {
		e.grid[i] = cell.Blank
	}
	e.filled = 0
	for i := range e.walk {
		e.spawn(&e.walk[i])
	}
}

func (e *pipes) spawn(p *pipe) {
	p.dir = e.rng.IntN(4)
	switch p.dir { // enter from the edge we are heading away from
	case 0:
		p.x, p.y = e.rng.IntN(e.w), e.h-1
	case 1:
		p.x, p.y = 0, e.rng.IntN(e.h)
	case 2:
		p.x, p.y = e.rng.IntN(e.w), 0
	default:
		p.x, p.y = e.w-1, e.rng.IntN(e.h)
	}
	p.hue = e.rng.Float64()
	p.n = 0
	for i := range p.recent {
		p.recent[i] = -1
	}
}

func (e *pipes) advance(p *pipe) {
	nd := p.dir
	// Horizontal runs are twice as long on screen per cell, so turn less
	// often when moving vertically to keep corners evenly spaced.
	turn := 0.1
	if p.dir == 0 || p.dir == 2 {
		turn = 0.18
	}
	if p.n > 1 && e.rng.Float64() < turn {
		nd = (p.dir + 1 + 2*e.rng.IntN(2)) % 4
	}
	from := 1 << ((p.dir + 2) % 4)
	if p.n == 0 {
		from = 1 << ((nd + 2) % 4)
	}
	i := p.y*e.w + p.x
	if e.grid[i].Rune == ' ' {
		e.filled++
	}
	// The colour drifts along the pipe's length.
	e.grid[i] = cell.Cell{Rune: e.glyphs[from|1<<nd], FG: e.pal.Cyclic(p.hue + float64(p.n)*0.003)}
	copy(p.recent[1:], p.recent[:pipeTrail-1])
	p.recent[0] = i
	p.n++
	p.x += pipeDX[nd]
	p.y += pipeDY[nd]
	p.dir = nd
	if p.x < 0 || p.y < 0 || p.x >= e.w || p.y >= e.h {
		e.spawn(p)
	}
}

func (e *pipes) Step(fr *fx.Frame) {
	const fadeTime = 0.8
	b := fr.Buf
	if e.fading > 0 {
		e.fading += fr.Dt
		if e.fading >= fadeTime {
			e.fading = 0
			e.reset()
		}
	} else {
		e.acc += e.speed * fr.Dt
		for ; e.acc >= 1; e.acc-- {
			for i := range e.walk {
				e.advance(&e.walk[i])
			}
		}
		if float64(e.filled) > 0.55*float64(e.w*e.h) {
			e.fading = 1e-9
		}
	}
	k := 1.0
	if e.fading > 0 {
		k = 1 - ambSmooth(e.fading/fadeTime)
	}
	for i, c := range e.grid {
		if k < 1 && c.FG.Valid {
			c.FG = tint.Scale(c.FG, k)
		}
		b.Cells[i] = c
	}
	if e.fading > 0 {
		return
	}
	white := tint.RGB(255, 255, 255)
	for _, p := range e.walk {
		for j := pipeTrail - 1; j >= 0; j-- {
			if idx := p.recent[j]; idx >= 0 {
				glow := 0.6 * (1 - float64(j)/pipeTrail)
				b.Cells[idx].FG = tint.Lerp(e.grid[idx].FG, white, glow)
			}
		}
	}
}
