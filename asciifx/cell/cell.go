// Package cell holds the terminal cell buffer every asciifx effect draws into.
// It knows nothing about escape sequences; see package term for output.
package cell

import (
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// Attr is a set of text attributes.
type Attr uint8

const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	Reverse
)

// Cell is one terminal cell. Rune 0 is treated as a space.
type Cell struct {
	Rune rune
	FG   tint.Color
	BG   tint.Color
	Attr Attr
}

// Blank is an empty cell in the terminal's default colours.
var Blank = Cell{Rune: ' '}

// Buffer is a W×H grid of cells, stored row-major.
type Buffer struct {
	W, H  int
	Cells []Cell
}

// New allocates a blank buffer.
func New(w, h int) *Buffer {
	b := &Buffer{}
	b.Resize(w, h)
	return b
}

// Resize reallocates to w×h and clears. Negative sizes become zero.
func (b *Buffer) Resize(w, h int) {
	w, h = max(w, 0), max(h, 0)
	b.W, b.H = w, h
	if cap(b.Cells) >= w*h {
		b.Cells = b.Cells[:w*h]
	} else {
		b.Cells = make([]Cell, w*h)
	}
	b.Clear()
}

// Clear resets every cell to Blank.
func (b *Buffer) Clear() {
	for i := range b.Cells {
		b.Cells[i] = Blank
	}
}

// In reports whether (x,y) is inside the buffer.
func (b *Buffer) In(x, y int) bool { return x >= 0 && y >= 0 && x < b.W && y < b.H }

// At returns a pointer to the cell at (x,y), or nil when out of bounds.
func (b *Buffer) At(x, y int) *Cell {
	if !b.In(x, y) {
		return nil
	}
	return &b.Cells[y*b.W+x]
}

// Set writes a cell, ignoring out-of-bounds writes.
func (b *Buffer) Set(x, y int, c Cell) {
	if b.In(x, y) {
		b.Cells[y*b.W+x] = c
	}
}

// SetRune writes a rune and foreground, keeping the existing background.
func (b *Buffer) SetRune(x, y int, r rune, fg tint.Color) {
	if c := b.At(x, y); c != nil {
		c.Rune, c.FG = r, fg
	}
}

// CopyFrom copies src into b, resizing b to match.
func (b *Buffer) CopyFrom(src *Buffer) {
	if b.W != src.W || b.H != src.H {
		b.Resize(src.W, src.H)
	}
	copy(b.Cells, src.Cells)
}

// Clone returns a deep copy.
func (b *Buffer) Clone() *Buffer {
	c := &Buffer{W: b.W, H: b.H, Cells: make([]Cell, len(b.Cells))}
	copy(c.Cells, b.Cells)
	return c
}

// WriteString draws s starting at (x,y) without wrapping. Newlines move to
// the next row back at column x. Runes that are not single-width are replaced
// with '?' so the grid never drifts.
func (b *Buffer) WriteString(x, y int, s string, fg tint.Color) {
	cx := x
	for _, r := range s {
		if r == '\n' {
			y++
			cx = x
			continue
		}
		if Width(r) != 1 {
			r = '?'
		}
		b.SetRune(cx, y, r, fg)
		cx++
	}
}

// Plain renders the buffer as text lines with trailing spaces kept, so every
// line has exactly W columns.
func (b *Buffer) Plain() string {
	var sb strings.Builder
	sb.Grow((b.W + 1) * b.H)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			r := b.Cells[y*b.W+x].Rune
			if r == 0 {
				r = ' '
			}
			sb.WriteRune(r)
		}
		if y < b.H-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// Lines is Plain split into rows.
func (b *Buffer) Lines() []string {
	if b.H == 0 {
		return nil
	}
	return strings.Split(b.Plain(), "\n")
}

// Equal reports whether two buffers hold identical cells.
func (b *Buffer) Equal(o *Buffer) bool {
	if b.W != o.W || b.H != o.H {
		return false
	}
	for i := range b.Cells {
		if b.Cells[i] != o.Cells[i] {
			return false
		}
	}
	return true
}
