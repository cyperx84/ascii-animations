package fx

import (
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// Text returns Content that centres multi-line text in the buffer. Runes that
// are not single-width become '?' so the layout never drifts. Lines keep
// their relative indentation.
func Text(s string, fg tint.Color) Content {
	s = strings.TrimRight(strings.ReplaceAll(s, "\t", "    "), "\n")
	lines := strings.Split(s, "\n")
	width := 0
	for _, l := range lines {
		width = max(width, len([]rune(l)))
	}
	return func(b *cell.Buffer) {
		x0 := (b.W - width) / 2
		y0 := (b.H - len(lines)) / 2
		for i, l := range lines {
			b.WriteString(max(x0, 0), y0+i, l, fg)
		}
	}
}

// Colorize returns Content that draws c, then lays gradient g across every
// non-space cell in direction dir.
func Colorize(c Content, g tint.Gradient, dir tint.Direction) Content {
	return func(b *cell.Buffer) {
		c(b)
		minX, minY, w, h := Bounds(b)
		for y := minY; y < minY+h; y++ {
			for x := minX; x < minX+w; x++ {
				if cl := b.At(x, y); Ink(cl) {
					cl.FG = g.At(tint.Position(dir, x-minX, y-minY, w, h))
				}
			}
		}
	}
}

// Ink reports whether a cell has visible content.
func Ink(c *cell.Cell) bool { return c.Rune != ' ' && c.Rune != 0 }

// Bounds returns the smallest rectangle holding every inked cell. An empty
// buffer yields the whole buffer, so callers can use it unconditionally.
func Bounds(b *cell.Buffer) (x, y, w, h int) {
	minX, minY, maxX, maxY := b.W, b.H, -1, -1
	for cy := 0; cy < b.H; cy++ {
		for cx := 0; cx < b.W; cx++ {
			if Ink(&b.Cells[cy*b.W+cx]) {
				minX, minY = min(minX, cx), min(minY, cy)
				maxX, maxY = max(maxX, cx), max(maxY, cy)
			}
		}
	}
	if maxX < 0 {
		return 0, 0, b.W, b.H
	}
	return minX, minY, maxX - minX + 1, maxY - minY + 1
}
