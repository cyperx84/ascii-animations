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
		Name:  "typewriter",
		Title: "Typewriter",
		Description: "Content types itself out in reading order with a human, uneven rhythm: each line accelerates " +
			"then slows, keys land as bright sparks that cool to colour, and a block cursor rides the caret and blinks when done.",
		Kind:     fx.Transition,
		Content:  true,
		Tags:     []string{"intro", "text", "terminal", "banner"},
		Glyphs:   []string{"ascii", "block"},
		FPS:      30,
		Duration: 2.5,
		MinW:     1,
		MinH:     1,
		Params: []fx.Param{
			fx.FloatParam("duration", 2.5, 0.2, 60, "Seconds for the whole run, including the closing cursor blink."),
			fx.FloatParam("blink", 0.2, 0, 0.8, "Fraction of the run the finished text waits with a blinking cursor."),
			fx.FloatParam("line_pause", 2, 0, 20, "Pause between lines, measured in keystrokes."),
			fx.FloatParam("jitter", 0.7, 0, 1.5, "Unevenness of keystroke timing."),
			fx.FloatParam("flash", 3, 0.5, 20, "Keystrokes a character takes to cool from its flash."),
			fx.EasingParam("easing", "in-out-sine", "Rhythm within each line."),
			fx.PaletteParam("palette", "sunset", "Colours; the brightest stop is the key flash and cursor."),
			fx.EnumParam("final", "gradient", []string{"content", "gradient"}, "Resting colour: the content's own colour, or the palette laid across it."),
			fx.EnumParam("direction", "diagonal", tint.Directions, "Direction of the final gradient."),
			fx.StringParam("cursor", "▌", "Cursor glyph. Single-width only."),
		},
		Example: `asciifx play typewriter --text "hello, world" -p palette=catppuccin`,
		New:     newTypewriter,
	})
}

type typewriter struct {
	dur, blink, pause, jitter, flash float64
	ease                             ease.Func
	pal                              tint.Gradient
	useGradient                      bool
	dir                              tint.Direction
	cursor                           rune
	seed                             uint64
}

func newTypewriter(p fx.Values, w, h int, rng *rand.Rand) (fx.Effect, error) {
	return &typewriter{
		dur:         p.Float("duration"),
		blink:       p.Float("blink"),
		pause:       p.Float("line_pause"),
		jitter:      p.Float("jitter"),
		flash:       p.Float("flash"),
		ease:        p.Easing("easing"),
		pal:         p.Palette("palette"),
		useGradient: p.String("final") == "gradient",
		dir:         tint.Direction(p.String("direction")),
		cursor:      safeRunes(p.String("cursor"), "▌")[0],
		seed:        rng.Uint64(),
	}, nil
}

func (t *typewriter) Duration() float64 { return t.dur }

// twLine is one row of content to type: columns x0..x1 inclusive, with the
// run's keystroke clock reaching it at start.
type twLine struct {
	y, x0, x1   int
	start, keys float64
}

func (t *typewriter) Step(f *fx.Frame) {
	b := f.Buf
	p := tickProgress(f, t.dur)
	if p >= 1 {
		settle(b, t.pal, t.useGradient, t.dir)
		return
	}
	bx, by, w, h := fx.Bounds(b)
	// Find the lines to type and lay them on one keystroke clock.
	var lines []twLine
	clock := 0.0
	for y := by; y < by+h; y++ {
		x0, x1 := -1, -1
		for x := bx; x < bx+w; x++ {
			if fx.Ink(b.At(x, y)) {
				if x0 < 0 {
					x0 = x
				}
				x1 = x
			}
		}
		if x0 < 0 {
			continue
		}
		if len(lines) > 0 {
			clock += t.pause
		}
		n := float64(x1 - x0 + 1)
		lines = append(lines, twLine{y: y, x0: x0, x1: x1, start: clock, keys: n})
		clock += n
	}
	typing := 1 - t.blink
	now := clock * math.Min(1, p/math.Max(typing, 1e-9))
	bright := tint.Lerp(t.pal.At(1), tint.RGB(255, 255, 255), 0.5)

	curX, curY := -1, -1
	for _, ln := range lines {
		// Keys typed on this line so far, eased so each line has its own
		// accelerate-then-settle rhythm.
		local := math.Max(0, math.Min(1, (now-ln.start)/ln.keys))
		typed := t.ease(local) * ln.keys
		if now >= ln.start {
			curX, curY = ln.x0+min(int(typed), int(ln.keys)), ln.y
		}
		// Uneven keystrokes: each key's arrival is nudged by its own hash by
		// less than a keystroke at default jitter, so text still reads left
		// to right.
		for x := ln.x0; x <= ln.x1; x++ {
			c := b.At(x, ln.y)
			if !fx.Ink(c) {
				continue
			}
			i := float64(x - ln.x0)
			at := math.Max(0.01, i+0.5+(fx.Hash01(x, ln.y, t.seed)-0.5)*t.jitter*0.9)
			age := typed - at
			if local >= 1 {
				age = math.Max(age, ln.keys-at+(now-ln.start-ln.keys))
			}
			if age <= 0 {
				*c = cell.Blank
				continue
			}
			final := restColor(c.FG, t.pal, t.useGradient, t.dir, x-bx, ln.y-by, w, h)
			c.FG = tint.Lerp(bright, final, ease.OutQuad(math.Min(1, age/t.flash)))
		}
	}
	if curY < 0 {
		// Nothing typed yet: the cursor waits at the start of the first line.
		if len(lines) == 0 {
			return
		}
		curX, curY = lines[0].x0, lines[0].y
	}
	// Solid while typing; blinks at about 2 Hz once the text is done.
	if now >= clock && math.Mod(f.T()*2.2, 1) >= 0.5 {
		return
	}
	if c := b.At(curX, curY); c != nil && (!fx.Ink(c) || now < clock) {
		*c = cell.Cell{Rune: t.cursor, FG: t.pal.At(1)}
	}
}
