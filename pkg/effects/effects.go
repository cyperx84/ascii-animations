// Package effects provides full-screen terminal effects (matrix rain, fire, rain).
package effects

import (
	"math"
	"math/rand"
	"strings"
)

// RenderMatrix produces a frame of the matrix rain effect.
func RenderMatrix(w, h, frame int) string {
	if w < 2 || h < 2 {
		return ""
	}
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789@#$%&*ﾊﾐﾋｰｳｼﾅﾓﾆｻﾜﾂｵﾘ"
	runes := []rune(chars)

	src := rand.New(rand.NewSource(42))
	cols := make([]int, w)
	speeds := make([]int, w)
	for i := range cols {
		cols[i] = src.Intn(h * 2)
		speeds[i] = src.Intn(3) + 1
	}

	buf := make([][]rune, h)
	for y := range buf {
		buf[y] = make([]rune, w)
		for x := range buf[y] {
			buf[y][x] = ' '
		}
	}

	for x := 0; x < w; x++ {
		head := (cols[x] + frame*speeds[x]) % (h + 10)
		trailLen := 5 + src.Intn(10)
		for t := 0; t < trailLen; t++ {
			y := head - t
			if y >= 0 && y < h {
				buf[y][x] = runes[src.Intn(len(runes))]
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := buf[y][x]
			if ch == ' ' {
				sb.WriteRune(' ')
			} else {
				sb.WriteString("\033[32m")
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			}
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RenderFire produces a frame of the fire simulation.
func RenderFire(w, h, frame int) string {
	if w < 2 || h < 2 {
		return ""
	}
	palette := []rune{' ', '.', ':', '-', '=', '+', '*', '#', '%', '@'}
	src := rand.New(rand.NewSource(int64(frame * 7)))

	heat := make([][]int, h+1)
	for y := range heat {
		heat[y] = make([]int, w)
	}

	// seed bottom row
	for x := 0; x < w; x++ {
		heat[h][x] = 7 + src.Intn(3)
	}

	// propagate upward
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			left := x - 1
			if left < 0 {
				left = 0
			}
			right := x + 1
			if right >= w {
				right = w - 1
			}
			avg := (heat[y+1][left] + heat[y+1][x] + heat[y+1][right]) / 3
			cool := src.Intn(2)
			val := avg - cool
			if val < 0 {
				val = 0
			}
			heat[y][x] = val
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := heat[y][x]
			if v >= len(palette) {
				v = len(palette) - 1
			}
			switch {
			case v >= 8:
				sb.WriteString("\033[97m") // bright white
			case v >= 6:
				sb.WriteString("\033[93m") // yellow
			case v >= 4:
				sb.WriteString("\033[91m") // red
			case v >= 2:
				sb.WriteString("\033[31m") // dark red
			default:
				sb.WriteString("\033[90m") // gray
			}
			sb.WriteRune(palette[v])
			sb.WriteString("\033[0m")
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RenderRain produces a frame of a rain effect with falling droplets.
func RenderRain(w, h, frame int) string {
	if w < 2 || h < 2 {
		return ""
	}
	src := rand.New(rand.NewSource(73))

	type drop struct {
		x, startY, speed int
	}
	numDrops := w * h / 8
	drops := make([]drop, numDrops)
	for i := range drops {
		drops[i] = drop{
			x:      src.Intn(w),
			startY: src.Intn(h * 3),
			speed:  1 + src.Intn(2),
		}
	}

	buf := make([][]rune, h)
	for y := range buf {
		buf[y] = make([]rune, w)
		for x := range buf[y] {
			buf[y][x] = ' '
		}
	}

	for _, d := range drops {
		y := (d.startY + frame*d.speed) % (h + 5)
		if y >= 0 && y < h {
			buf[y][d.x] = '│'
		}
		// trail
		if y-1 >= 0 && y-1 < h {
			buf[y-1][d.x] = '.'
		}
		// splash at bottom
		if y == h-1 && d.x > 0 && d.x < w-1 {
			buf[y][d.x] = '╨'
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := buf[y][x]
			if ch != ' ' {
				sb.WriteString("\033[94m") // light blue
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(' ')
			}
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RenderStarfield produces a frame of a warp-speed starfield.
func RenderStarfield(w, h, frame int) string {
	if w < 4 || h < 4 {
		return ""
	}
	type star struct {
		x, y, speed float64
	}
	const numStars = 80
	src := rand.New(rand.NewSource(99))
	stars := make([]star, numStars)
	for i := range stars {
		stars[i] = star{
			x:     src.Float64()*2 - 1,
			y:     src.Float64()*2 - 1,
			speed: 0.01 + src.Float64()*0.03,
		}
	}

	cx := float64(w) / 2
	cy := float64(h) / 2
	screen := make([][]rune, h)
	for y := range screen {
		screen[y] = make([]rune, w)
		for x := range screen[y] {
			screen[y][x] = ' '
		}
	}

	glyphs := []rune{'.', '+', '*', '#'}
	for i := range stars {
		t := float64(frame) * stars[i].speed
		dist := math.Mod(t, 1.0)
		if dist < 0.01 {
			dist = 0.01
		}
		sx := int(cx + stars[i].x*dist*cx)
		sy := int(cy + stars[i].y*dist*cy)
		if sx >= 0 && sx < w && sy >= 0 && sy < h {
			gi := int(dist * float64(len(glyphs)))
			if gi >= len(glyphs) {
				gi = len(glyphs) - 1
			}
			screen[sy][sx] = glyphs[gi]
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := screen[y][x]
			if ch != ' ' {
				sb.WriteString("\033[97m")
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			} else {
				sb.WriteRune(' ')
			}
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// SourceMatrix is example source code for matrix rain.
const SourceMatrix = `// Matrix rain with Bubble Tea
// Each column tracks a falling head position and speed.
// Render bright head char + dimming trail behind it.
// Use lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))`

// SourceFire is example source code for fire effect.
const SourceFire = `// Fire simulation
// Bottom row = max heat, propagate upward with random cooling.
// Map heat (0-9) to chars: " .:-=+*#%@"
// Color with ANSI: high heat = yellow/white, low = red/dark`

// SourceRain is example source code for rain effect.
const SourceRain = `// Rain effect with Bubble Tea
// Spawn droplets at random x positions, fall downward.
// Use │ for drops, . for trail, ╨ for splash.
// Color with ANSI blue: \033[94m`

// SourceStarfield is example source code for starfield.
const SourceStarfield = `// Starfield: stars move outward from center
// Each star has (x,y) in [-1,1] and a speed.
// Project to screen coords, scale by distance from center.
// Brightness: closer to edge = brighter (farther traveled).`
