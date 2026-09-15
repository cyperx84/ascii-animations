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
		Name:  "beams",
		Title: "Beams",
		Description: "Light beams streak along rows and down columns in a staggered barrage, leaving the content " +
			"glowing dimly in their wake; then a bright wave sweeps diagonally and ignites everything to its full gradient.",
		Kind:     fx.Transition,
		Content:  true,
		Tags:     []string{"intro", "light", "splash", "banner"},
		Glyphs:   []string{"ascii", "block"},
		FPS:      30,
		Duration: 3,
		MinW:     1,
		MinH:     1,
		Params: []fx.Param{
			fx.FloatParam("duration", 3, 0.3, 30, "Seconds for beams and the closing wave."),
			fx.FloatParam("beam_phase", 0.65, 0.2, 0.9, "Fraction of the run the beams take; the wave starts just before they finish."),
			fx.FloatParam("beam_time", 0.3, 0.05, 1, "Fraction of the beam phase one beam takes to cross."),
			fx.IntParam("row_trail", 8, 1, 40, "Length of a row beam's glowing tail, in columns."),
			fx.IntParam("column_trail", 3, 1, 20, "Length of a column beam's tail, in rows."),
			fx.FloatParam("columns", 0.5, 0, 1, "Share of columns that fire their own beam."),
			fx.FloatParam("dim", 0.45, 0.1, 0.9, "Brightness of content lit by beams before the wave."),
			fx.FloatParam("softness", 0.3, 0.02, 1, "Width of the final wave's edge."),
			fx.PaletteParam("palette", "aurora", "Beam and final colours, dark to bright."),
			fx.EnumParam("final", "gradient", []string{"content", "gradient"}, "Resting colour: the content's own colour, or the palette laid across it."),
			fx.EnumParam("direction", "diagonal", tint.Directions, "Direction of the final gradient."),
		},
		Example: `asciifx play beams --banner "LAUNCH" -p palette=synthwave`,
		New:     newBeams,
	})
}

type beams struct {
	dur, phase, beamTime, columns, dim, soft float64
	rowTrail, colTrail                       int
	pal                                      tint.Gradient
	useGradient                              bool
	dir                                      tint.Direction
	seed                                     uint64
}

func newBeams(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &beams{
		dur:         p.Float("duration"),
		phase:       p.Float("beam_phase"),
		beamTime:    p.Float("beam_time"),
		columns:     p.Float("columns"),
		dim:         p.Float("dim"),
		soft:        p.Float("softness"),
		rowTrail:    p.Int("row_trail"),
		colTrail:    p.Int("column_trail"),
		pal:         p.Palette("palette"),
		useGradient: p.String("final") == "gradient",
		dir:         tint.Direction(p.String("direction")),
		seed:        rng.Uint64(),
	}, nil
}

func (e *beams) Duration() float64 { return e.dur }

var (
	rowBeamGlyphs = []rune("▁▂▃▄▅▆▇█")
	colBeamGlyphs = []rune("▏▎▍▌▋▊▉█")
)

// lane is where one beam is relative to a cell along its path: k is the
// trail brightness there (1 at the head, 0 outside), lit whether the head has
// already passed.
func (e *beams) lane(p float64, id, salt uint64, s, length, trail int) (k float64, lit bool) {
	span := e.phase * (1 - e.beamTime)
	start := fx.Hash01(int(id), int(salt), e.seed) * span
	t := (p - start) / (e.phase * e.beamTime)
	head := t * float64(length+trail)
	dist := head - float64(s)
	if dist < 0 {
		return 0, false
	}
	if dist < float64(trail) {
		k = 1 - dist/float64(trail)
	}
	return k, true
}

func (e *beams) Step(f *fx.Frame) {
	b := f.Buf
	p := tickProgress(f, e.dur)
	if p >= 1 {
		settle(b, e.pal, e.useGradient, e.dir)
		return
	}
	bx, by, w, h := fx.Bounds(b)
	white := tint.RGB(255, 255, 255)
	// The wave starts a little before the last beams land so the two phases
	// overlap instead of pausing.
	waveStart := e.phase * 0.85
	q := math.Max(0, (p-waveStart)/(1-waveStart))
	for y := by; y < by+h; y++ {
		ly := y - by
		rowRev := fx.Hash01(ly, 3, e.seed) < 0.5
		for x := bx; x < bx+w; x++ {
			lx := x - bx
			c := b.At(x, y)
			inked := fx.Ink(c)

			s := lx
			if rowRev {
				s = w - 1 - lx
			}
			k, lit := e.lane(p, uint64(ly), 1, s, w, e.rowTrail)
			vertical := false
			if fx.Hash01(lx, 4, e.seed) < e.columns {
				s := ly
				if fx.Hash01(lx, 5, e.seed) < 0.5 {
					s = h - 1 - ly
				}
				ck, clit := e.lane(p, uint64(lx), 2, s, h, e.colTrail)
				lit = lit || clit
				if ck > k {
					k, vertical = ck, true
				}
			}

			final := restColor(c.FG, e.pal, e.useGradient, e.dir, lx, ly, w, h)
			wave := fx.Local(q, tint.Position(tint.Diagonal, lx, ly, w, h), e.soft)
			beamColor := tint.Lerp(e.pal.At(0.55+0.45*k), white, k*k*0.7)

			switch {
			case k > 0.55 || (k > 0 && !inked):
				glyphs := rowBeamGlyphs
				if vertical {
					glyphs = colBeamGlyphs
				}
				*c = cell.Cell{Rune: glyphs[min(int(k*float64(len(glyphs))), len(glyphs)-1)], FG: beamColor}
			case !inked || (!lit && wave <= 0):
				*c = cell.Blank
			default:
				glow := tint.Scale(final, e.dim)
				if k > 0 {
					glow = tint.Lerp(glow, beamColor, k/0.55)
				}
				switch {
				case wave <= 0:
					c.FG = glow
				case wave < 0.4:
					c.FG = tint.Lerp(glow, white, ease.OutQuad(wave/0.4)*0.85)
				default:
					c.FG = tint.Lerp(tint.Lerp(final, white, 0.85), final, ease.InOutSine((wave-0.4)/0.6))
				}
			}
		}
	}
}
