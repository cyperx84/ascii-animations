// Package colors provides terminal color tier showcases (16, 256, truecolor).
package colors

import (
	"fmt"
	"strings"
)

// Render16 returns a display of the standard 16 ANSI colors.
func Render16() string {
	names := []struct {
		code int
		name string
	}{
		{30, "Black"}, {31, "Red"}, {32, "Green"}, {33, "Yellow"},
		{34, "Blue"}, {35, "Magenta"}, {36, "Cyan"}, {37, "White"},
		{90, "Bright Black"}, {91, "Bright Red"}, {92, "Bright Green"}, {93, "Bright Yellow"},
		{94, "Bright Blue"}, {95, "Bright Magenta"}, {96, "Bright Cyan"}, {97, "Bright White"},
	}

	var sb strings.Builder
	sb.WriteString("  Standard 16 ANSI Colors\n\n")
	for i, c := range names {
		bg := c.code + 10
		if c.code >= 90 {
			bg = c.code - 90 + 100
		}
		sb.WriteString(fmt.Sprintf("  \033[%dm  ██  \033[0m %-14s", bg, c.name))
		if (i+1)%4 == 0 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// Render256 returns a display of the full 256-color palette.
func Render256() string {
	var sb strings.Builder
	sb.WriteString("  256-Color Palette\n\n")

	sb.WriteString("  Standard:\n  ")
	for i := 0; i < 16; i++ {
		sb.WriteString(fmt.Sprintf("\033[48;5;%dm  \033[0m", i))
	}
	sb.WriteString("\n\n")

	sb.WriteString("  Color Cube (6×6×6):\n")
	for row := 0; row < 12; row++ {
		sb.WriteString("  ")
		for col := 0; col < 18; col++ {
			idx := 16 + row*18 + col
			if idx < 232 {
				sb.WriteString(fmt.Sprintf("\033[48;5;%dm  \033[0m", idx))
			}
		}
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')

	sb.WriteString("  Grayscale:\n  ")
	for i := 232; i < 256; i++ {
		sb.WriteString(fmt.Sprintf("\033[48;5;%dm  \033[0m", i))
	}
	return sb.String()
}

// RenderTruecolor produces a frame of a scrolling truecolor gradient.
func RenderTruecolor(w, h, frame int) string {
	if w < 4 || h < 4 {
		return ""
	}
	rows := h - 4
	if rows > 20 {
		rows = 20
	}
	cols := w - 4
	if cols > 80 {
		cols = 80
	}

	var sb strings.Builder
	sb.WriteString("  Truecolor (24-bit RGB)\n\n")

	offset := frame * 3
	for y := 0; y < rows; y++ {
		sb.WriteString("  ")
		for x := 0; x < cols; x++ {
			hue := float64(x)/float64(cols)*360 + float64(offset)
			sat := 1.0
			val := 1.0 - float64(y)/float64(rows)*0.5
			r, g, b := hsvToRGB(hue, sat, val)
			sb.WriteString(fmt.Sprintf("\033[48;2;%d;%d;%dm \033[0m", r, g, b))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func hsvToRGB(h, s, v float64) (int, int, int) {
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}
	c := v * s
	x := c * (1 - abs(mod(h/60, 2)-1))
	m := v - c

	var r1, g1, b1 float64
	switch {
	case h < 60:
		r1, g1, b1 = c, x, 0
	case h < 120:
		r1, g1, b1 = x, c, 0
	case h < 180:
		r1, g1, b1 = 0, c, x
	case h < 240:
		r1, g1, b1 = 0, x, c
	case h < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	return int((r1 + m) * 255), int((g1 + m) * 255), int((b1 + m) * 255)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func mod(a, b float64) float64 {
	res := a - float64(int(a/b))*b
	if res < 0 {
		res += b
	}
	return res
}

// Source snippets for color showcases.
const (
	Source16 = `// 16-color ANSI palette
// Use \033[30m-\033[37m for standard
// Use \033[90m-\033[97m for bright
// Maximum terminal compatibility`

	Source256 = `// 256-color mode
// \033[48;5;{n}m for background, \033[38;5;{n}m for foreground
// 0-7: standard, 8-15: bright, 16-231: color cube, 232-255: grayscale`

	SourceTrue = `// Truecolor (24-bit)
// \033[48;2;R;G;Bm for background
// Over 16 million colors — requires modern terminal
// iTerm2, Ghostty, Kitty, Alacritty, Windows Terminal`
)
