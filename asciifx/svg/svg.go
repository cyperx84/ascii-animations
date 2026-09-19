// Package svg writes an animation as a self-contained animated SVG.
//
// It exists because a terminal animation is hard to show anywhere that is not
// a terminal. A recording needs a player, a GIF needs a rasteriser and a font,
// and neither survives a README. An SVG needs nothing: the frames are text,
// the viewer's own monospace font draws them, and a CSS keyframe shows one
// frame at a time. Markdown renderers that allow an <img> — GitHub's among
// them — animate it.
//
//	frames := make([]*cell.Buffer, 0, 60)
//	for tick := range 60 {
//		b, err := run.Seek(tick)
//		...
//		frames = append(frames, b.Clone())
//	}
//	err := svg.Encode(w, frames, svg.Options{FPS: run.FPS()})
//
// Buffers are read once, in order, and never retained, but Seek reuses its
// buffer — clone each frame before collecting it, as above.
package svg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// Options configure the document. The zero value is valid and gives a
// GitHub-dark 16px rendering at 30 fps.
type Options struct {
	// FontSize in CSS pixels.
	FontSize float64
	// CellW and CellH are the advance width and line height in pixels. Zero
	// derives them from FontSize at the proportions a monospace face uses,
	// which is what keeps columns aligned without measuring a font.
	CellW, CellH float64
	// FontFamily is the CSS font stack. The fallbacks matter more than the
	// first entry: whoever opens this will not have your terminal font.
	FontFamily string
	// Background is a CSS colour, or "none" for a transparent document.
	Background string
	// Foreground is the colour of cells the animation left uncoloured.
	Foreground string
	// FPS is the playback rate. Zero is 30.
	FPS int
	// Padding in pixels around the frame.
	Padding float64
	// Title becomes the document's <title>, which is also its accessible
	// name. Empty leaves it out.
	Title string
	// Quantize rounds each colour channel to this many levels before the
	// runs are cut, which merges neighbouring cells of an almost-identical
	// colour into one element. A gradient is where the file size is, and 32
	// levels a channel is below what the eye picks out of terminal art. Zero
	// keeps every colour exactly as the effect produced it.
	Quantize int
}

func (o *Options) defaults() {
	if o.FontSize <= 0 {
		o.FontSize = 16
	}
	if o.CellW <= 0 {
		// 0.6em is the advance of the common monospace faces (Menlo, DejaVu
		// Sans Mono, Consolas, Liberation Mono) to within a rounding error.
		o.CellW = o.FontSize * 0.6
	}
	if o.CellH <= 0 {
		o.CellH = o.FontSize * 1.2
	}
	if o.FontFamily == "" {
		o.FontFamily = "ui-monospace,SFMono-Regular,Menlo,Consolas,'DejaVu Sans Mono',monospace"
	}
	if o.Background == "" {
		o.Background = "#0d1117"
	}
	if o.Foreground == "" {
		o.Foreground = "#c9d1d9"
	}
	if o.FPS <= 0 {
		o.FPS = 30
	}
	if o.Quantize < 0 {
		o.Quantize = 0
	}
}

// quant rounds a colour to Quantize levels a channel. Rounding rather than
// truncating keeps white white, which truncation would dim by a step.
func (o Options) quant(c tint.Color) tint.Color {
	if o.Quantize <= 1 || !c.Valid {
		return c
	}
	step := 255.0 / float64(o.Quantize-1)
	q := func(v uint8) uint8 {
		return uint8(float64(int(float64(v)/step+0.5))*step + 0.5)
	}
	return tint.Color{R: q(c.R), G: q(c.G), B: q(c.B), Valid: true}
}

// ErrNoFrames is returned when there is nothing to animate.
var ErrNoFrames = errors.New("svg: no frames")

// Encode writes frames as one animated SVG document. Every frame must be the
// same size, because a document has one viewBox; a frame of a different size
// is an error rather than a silently clipped one.
func Encode(w io.Writer, frames []*cell.Buffer, o Options) error {
	if len(frames) == 0 {
		return ErrNoFrames
	}
	o.defaults()
	cols, rows := frames[0].W, frames[0].H
	for i, f := range frames {
		if f.W != cols || f.H != rows {
			return fmt.Errorf("svg: frame %d is %dx%d, frame 0 is %dx%d", i, f.W, f.H, cols, rows)
		}
	}
	width := float64(cols)*o.CellW + 2*o.Padding
	height := float64(rows)*o.CellH + 2*o.Padding
	total := float64(len(frames)) / float64(o.FPS)

	out := bufio.NewWriter(w)
	fmt.Fprintf(out, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" font-family="%s" font-size="%.0fpx">`,
		width, height, width, height, o.FontFamily, o.FontSize)
	out.WriteByte('\n')
	if o.Title != "" {
		fmt.Fprintf(out, "<title>%s</title>\n", escape(o.Title))
	}
	writeStyle(out, len(frames), total, o)
	if o.Background != "none" {
		fmt.Fprintf(out, `<rect width="100%%" height="100%%" fill="%s"/>`+"\n", o.Background)
	}
	for i, f := range frames {
		fmt.Fprintf(out, `<g class="f f%d">`+"\n", i)
		writeFrame(out, f, o)
		out.WriteString("</g>\n")
	}
	out.WriteString("</svg>\n")
	return out.Flush()
}

// writeStyle emits one keyframe rule per frame. Each frame is hidden except
// during its own slot, so exactly one is visible at a time and the browser
// does the timing. steps(1) keeps opacity from tweening between frames, which
// would show two at once.
func writeStyle(out *bufio.Writer, n int, total float64, o Options) {
	out.WriteString("<style>\n")
	out.WriteString(".f{opacity:0;animation:" + num(round4(total)) + "s infinite steps(1,end)}\n")
	fmt.Fprintf(out, "text{white-space:pre;fill:%s;dominant-baseline:text-before-edge}\n", o.Foreground)
	slot := 100.0 / float64(n)
	for i := 0; i < n; i++ {
		start, end := float64(i)*slot, float64(i+1)*slot
		fmt.Fprintf(out, ".f%d{animation-name:k%d}@keyframes k%d{%s%%{opacity:1}%s%%{opacity:0}}\n",
			i, i, i, num(round4(start)), num(round4(end)))
	}
	out.WriteString("</style>\n")
}

// block describes a glyph that is pure geometry: the fraction of the cell it
// fills, as a rectangle, and how solid that rectangle is.
type block struct {
	x, y, w, h float64
	opacity    float64
}

// blocks are the glyphs drawn as rectangles rather than as text. A terminal
// scales these to fill the cell exactly; a font does not, so drawing them as
// text leaves a seam between every row and a gap between every column. They
// are also most of what an ambient effect emits, so this is where the file
// size goes too.
var blocks = map[rune]block{
	'█': {0, 0, 1, 1, 1},
	'▀': {0, 0, 1, 0.5, 1},
	'▄': {0, 0.5, 1, 0.5, 1},
	'▌': {0, 0, 0.5, 1, 1},
	'▐': {0.5, 0, 0.5, 1, 1},
	'▁': {0, 0.875, 1, 0.125, 1},
	'▂': {0, 0.75, 1, 0.25, 1},
	'▃': {0, 0.625, 1, 0.375, 1},
	'▅': {0, 0.375, 1, 0.625, 1},
	'▆': {0, 0.25, 1, 0.75, 1},
	'▇': {0, 0.125, 1, 0.875, 1},
	'░': {0, 0, 1, 1, 0.25},
	'▒': {0, 0, 1, 1, 0.5},
	'▓': {0, 0, 1, 1, 0.75},
}

// writeFrame emits one <text> per row, split into <tspan> runs of equal
// style. A run is the unit because a colour change is the only thing that
// needs a new element, and terminal art changes colour far less often than it
// changes character.
func writeFrame(out *bufio.Writer, b *cell.Buffer, o Options) {
	for y := 0; y < b.H; y++ {
		row := b.Cells[y*b.W : (y+1)*b.W]
		writeBackgrounds(out, row, y, o)
		writeBlocks(out, row, y, o)
		if blank(row) {
			continue
		}
		out.WriteString(`<text x="` + num(o.Padding) + `" y="` + num(o.Padding+float64(y)*o.CellH) + `">`)
		var run strings.Builder
		var runFG tint.Color
		var runAttr cell.Attr
		runStart := 0
		// next is the column the previous run ended at. A run starting there
		// needs no x of its own, which is most of them.
		next := -1
		flush := func(end int) {
			if run.Len() == 0 {
				return
			}
			if writeRun(out, run.String(), runFG, runAttr, o, runStart, next) {
				next = end
			}
			run.Reset()
		}
		for x, c := range row {
			c.FG = o.quant(c.FG)
			r := runeOf(c)
			if _, isBlock := blocks[r]; isBlock {
				// Already drawn as a rectangle. A space keeps the run's
				// columns lined up without painting over it.
				r = ' '
			}
			if run.Len() > 0 && (c.FG != runFG || c.Attr != runAttr) {
				flush(x)
				runStart = x
			}
			if run.Len() == 0 {
				runFG, runAttr, runStart = c.FG, c.Attr, x
			}
			run.WriteRune(r)
		}
		flush(b.W)
		out.WriteString("</text>\n")
	}
}

// writeRun emits one styled run and reports whether it drew anything.
func writeRun(out *bufio.Writer, s string, fg tint.Color, attr cell.Attr, o Options, x, next int) bool {
	// Trailing spaces in a run paint nothing, so they only cost bytes.
	if strings.TrimRight(s, " ") == "" {
		return false
	}
	out.WriteString(`<tspan`)
	if x != next {
		out.WriteString(` x="` + num(o.Padding+float64(x)*o.CellW) + `"`)
	}
	if fg.Valid {
		fmt.Fprintf(out, ` fill="#%02x%02x%02x"`, fg.R, fg.G, fg.B)
	}
	if attr&cell.Bold != 0 {
		out.WriteString(` font-weight="bold"`)
	}
	if attr&cell.Italic != 0 {
		out.WriteString(` font-style="italic"`)
	}
	if attr&cell.Dim != 0 {
		out.WriteString(` opacity="0.55"`)
	}
	if attr&cell.Underline != 0 {
		out.WriteString(` text-decoration="underline"`)
	}
	out.WriteString(">")
	out.WriteString(escape(s))
	out.WriteString("</tspan>")
	return true
}

// writeBackgrounds paints the cell backgrounds of one row as rectangles. It
// is not decoration: a half-block cell is a glyph for the top half and a
// background colour for the bottom, so the ambient effects lose half their
// resolution without it.
func writeBackgrounds(out *bufio.Writer, row []cell.Cell, y int, o Options) {
	x := 0
	for x < len(row) {
		bg := o.quant(row[x].BG)
		if !bg.Valid {
			x++
			continue
		}
		end := x + 1
		for end < len(row) && o.quant(row[end].BG) == bg {
			end++
		}
		fmt.Fprintf(out, `<rect x="%s" y="%s" width="%s" height="%s" fill="#%02x%02x%02x"/>`,
			num(o.Padding+float64(x)*o.CellW), num(o.Padding+float64(y)*o.CellH),
			num(float64(end-x)*o.CellW), num(o.CellH), bg.R, bg.G, bg.B)
		x = end
	}
}

// blank reports whether a row has nothing left for the text pass: spaces and
// glyphs the rectangle pass already drew.
func blank(row []cell.Cell) bool {
	for _, c := range row {
		r := runeOf(c)
		if _, isBlock := blocks[r]; isBlock {
			continue
		}
		if r != ' ' {
			return false
		}
	}
	return true
}

// writeBlocks draws the geometric glyphs of one row, merging neighbours that
// share a shape and a colour into one rectangle.
func writeBlocks(out *bufio.Writer, row []cell.Cell, y int, o Options) {
	x := 0
	for x < len(row) {
		bl, ok := blocks[runeOf(row[x])]
		if !ok {
			x++
			continue
		}
		fg := o.quant(row[x].FG)
		end := x + 1
		for end < len(row) {
			nb, nok := blocks[runeOf(row[end])]
			if !nok || nb != bl || o.quant(row[end].FG) != fg {
				break
			}
			end++
		}
		fmt.Fprintf(out, `<rect x="%s" y="%s" width="%s" height="%s" fill="%s"`,
			num(o.Padding+(float64(x)+bl.x)*o.CellW),
			num(o.Padding+(float64(y)+bl.y)*o.CellH),
			num(float64(end-x)*o.CellW*bl.w),
			num(bl.h*o.CellH),
			// An uncoloured block is still a block. Falling through to the
			// document foreground is what a terminal does with it.
			fill(fg, o.Foreground))
		if bl.opacity < 1 {
			out.WriteString(` opacity="` + num(bl.opacity) + `"`)
		}
		out.WriteString("/>")
		x = end
	}
}

func runeOf(c cell.Cell) rune {
	if c.Rune == 0 {
		return ' '
	}
	return c.Rune
}

// num formats a coordinate as short as it can: SVG files are mostly numbers,
// and two significant decimals of a pixel are below what any renderer shows.
// fill is a colour as CSS, or the fallback when the cell had none.
func fill(c tint.Color, fallback string) string {
	if !c.Valid {
		return fallback
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// round4 keeps four decimals, which is finer than any renderer's timing and
// short enough not to bloat a rule that repeats once per frame.
func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }

func num(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escape(s string) string { return escaper.Replace(s) }
