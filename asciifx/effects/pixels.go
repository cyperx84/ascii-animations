// Package effects holds the built-in asciifx effects. Importing it registers
// them all with package fx.
package effects

import (
	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// HalfBlock packs two vertical pixels into one cell. Unset colours are
// transparent, so the terminal background shows through.
func HalfBlock(top, bottom tint.Color) cell.Cell {
	switch {
	case !top.Valid && !bottom.Valid:
		return cell.Blank
	case !bottom.Valid:
		return cell.Cell{Rune: '▀', FG: top}
	case !top.Valid:
		return cell.Cell{Rune: '▄', FG: bottom}
	case top == bottom:
		return cell.Cell{Rune: '█', FG: top}
	default:
		return cell.Cell{Rune: '▀', FG: top, BG: bottom}
	}
}

// Braille packs a 2×4 dot grid into one cell. bits[row][col] is set when
// that dot is lit; the result is a blank cell when none are.
func Braille(bits [4][2]bool, fg tint.Color) cell.Cell {
	offsets := [4][2]rune{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	var r rune
	for y := range bits {
		for x := range bits[y] {
			if bits[y][x] {
				r |= offsets[y][x]
			}
		}
	}
	if r == 0 {
		return cell.Blank
	}
	return cell.Cell{Rune: 0x2800 + r, FG: fg}
}

// Ramp is a light-to-dense ASCII shading ramp, as in donut.c.
const Ramp = " .,-~:;=!*#$@"

// RampRune maps intensity in [0,1] onto Ramp.
func RampRune(v float64) rune {
	i := int(v * float64(len(Ramp)-1))
	return rune(Ramp[max(0, min(i, len(Ramp)-1))])
}
