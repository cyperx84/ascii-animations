package term

import (
	"strconv"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

const (
	syncStart = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
	reset     = "\x1b[0m"
)

type style struct {
	fg, bg tint.Color
	attr   cell.Attr
}

func styleOf(c cell.Cell, p Profile) style {
	if p == NoColor {
		return style{attr: c.Attr}
	}
	return style{c.FG, c.BG, c.Attr}
}

func writeStyle(sb *strings.Builder, s style, p Profile) {
	sb.WriteString("\x1b[0")
	attrs := [...]struct {
		a    cell.Attr
		code string
	}{{cell.Bold, ";1"}, {cell.Dim, ";2"}, {cell.Italic, ";3"}, {cell.Underline, ";4"}, {cell.Reverse, ";7"}}
	for _, at := range attrs {
		if s.attr&at.a != 0 {
			sb.WriteString(at.code)
		}
	}
	writeColor(sb, s.fg, p, false)
	writeColor(sb, s.bg, p, true)
	sb.WriteByte('m')
}

func writeColor(sb *strings.Builder, c tint.Color, p Profile, bg bool) {
	if !c.Valid {
		return
	}
	switch p {
	case TrueColor:
		if bg {
			sb.WriteString(";48;2;")
		} else {
			sb.WriteString(";38;2;")
		}
		sb.WriteString(strconv.Itoa(int(c.R)))
		sb.WriteByte(';')
		sb.WriteString(strconv.Itoa(int(c.G)))
		sb.WriteByte(';')
		sb.WriteString(strconv.Itoa(int(c.B)))
	case ANSI256:
		if bg {
			sb.WriteString(";48;5;")
		} else {
			sb.WriteString(";38;5;")
		}
		sb.WriteString(strconv.Itoa(int(tint.To256(c))))
	case ANSI16:
		i := int(tint.To16(c))
		base := 30
		if bg {
			base = 40
		}
		if i >= 8 {
			base += 60
			i -= 8
		}
		sb.WriteByte(';')
		sb.WriteString(strconv.Itoa(base + i))
	}
}

func runeOf(c cell.Cell) rune {
	if c.Rune == 0 {
		return ' '
	}
	return c.Rune
}

// ANSI encodes a whole buffer as self-contained lines, each ending in a
// reset. Use it for static output and for frameworks (Bubble Tea, Lip Gloss)
// that take a string view.
func ANSI(b *cell.Buffer, p Profile) string {
	var sb strings.Builder
	sb.Grow(b.W * b.H * 4)
	plain := style{}
	for y := 0; y < b.H; y++ {
		cur := plain
		for x := 0; x < b.W; x++ {
			c := b.Cells[y*b.W+x]
			if s := styleOf(c, p); s != cur {
				if s == plain {
					sb.WriteString(reset)
				} else {
					writeStyle(&sb, s, p)
				}
				cur = s
			}
			sb.WriteRune(runeOf(c))
		}
		if cur != plain {
			sb.WriteString(reset)
		}
		if y < b.H-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// Renderer encodes successive frames as minimal diffs. The region's top-left
// is wherever the cursor sat when rendering began; every frame leaves the
// cursor back there, so the same renderer works on the alternate screen and
// inline below a shell prompt.
type Renderer struct {
	Profile Profile
	// Sync wraps frames in synchronized-output mode 2026. Terminals that do
	// not know the mode ignore it, so this is safe to leave on.
	Sync bool
	prev *cell.Buffer
}

// Reset forgets the previous frame so the next one is drawn in full.
func (r *Renderer) Reset() { r.prev = nil }

// Frame returns the bytes that turn the previous frame into b. It returns
// nil when nothing changed, so idle animations cost nothing.
func (r *Renderer) Frame(b *cell.Buffer) []byte {
	full := r.prev == nil || r.prev.W != b.W || r.prev.H != b.H
	var sb strings.Builder
	cx, cy := 0, 0
	cur := style{}
	styled := false
	moveTo := func(x, y int) {
		if y > cy {
			sb.WriteString("\x1b[" + strconv.Itoa(y-cy) + "B")
		} else if y < cy {
			sb.WriteString("\x1b[" + strconv.Itoa(cy-y) + "A")
		}
		if x != cx {
			sb.WriteByte('\r')
			if x > 0 {
				sb.WriteString("\x1b[" + strconv.Itoa(x) + "C")
			}
		}
		cx, cy = x, y
	}
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.Cells[y*b.W+x]
			if !full && r.prev.Cells[y*b.W+x] == c {
				continue
			}
			if cx != x || cy != y {
				moveTo(x, y)
			}
			if s := styleOf(c, r.Profile); !styled || s != cur {
				writeStyle(&sb, s, r.Profile)
				cur, styled = s, true
			}
			sb.WriteRune(runeOf(c))
			cx++
			// Writing the last column leaves the cursor in a pending-wrap
			// state that terminals disagree about; re-anchor explicitly.
			if cx >= b.W {
				sb.WriteByte('\r')
				cx = 0
			}
		}
	}
	if sb.Len() == 0 {
		return nil
	}
	if styled {
		sb.WriteString(reset)
	}
	moveTo(0, 0)
	if r.prev == nil || r.prev.W != b.W || r.prev.H != b.H {
		r.prev = b.Clone()
	} else {
		r.prev.CopyFrom(b)
	}
	if r.Sync {
		return []byte(syncStart + sb.String() + syncEnd)
	}
	return []byte(sb.String())
}

// Luma renders the buffer as a brightness map using a density ramp, so a
// reader without colour (an agent, a log, a golden file) can still see the
// shape of colour-only effects such as half-block fire or plasma. Glyphs
// count by how much of the cell they cover.
func Luma(b *cell.Buffer) string {
	const ramp = " .:-=+*#%@"
	var sb strings.Builder
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.Cells[y*b.W+x]
			fg := lightness(c.FG, 1)
			bg := lightness(c.BG, 0)
			cov := coverage(runeOf(c))
			v := fg*cov + bg*(1-cov)
			sb.WriteByte(ramp[min(int(v*float64(len(ramp)-1)+0.5), len(ramp)-1)])
		}
		if y < b.H-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func lightness(c tint.Color, unset float64) float64 {
	if !c.Valid {
		return unset * 0.8
	}
	return c.OKLab().L
}

func coverage(r rune) float64 {
	switch {
	case r == ' ':
		return 0
	case r == '█':
		return 1
	case r == '▀' || r == '▄' || r == '▌' || r == '▐':
		return 0.5
	case r == '░':
		return 0.25
	case r == '▒':
		return 0.5
	case r == '▓':
		return 0.75
	case r >= 0x2800 && r <= 0x28FF:
		return float64(bitsSet(int(r-0x2800))) / 8
	case r < 0x80 && strings.ContainsRune(".,'`-", r):
		return 0.15
	}
	return 0.45
}

func bitsSet(v int) int {
	n := 0
	for ; v > 0; v &= v - 1 {
		n++
	}
	return n
}
