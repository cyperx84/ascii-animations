// Package effects provides full-screen terminal effects.
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

// RenderSnow produces a frame of falling snow with accumulation.
func RenderSnow(w, h, frame int) string {
	if w < 2 || h < 2 {
		return ""
	}
	src := rand.New(rand.NewSource(55))
	flakes := []rune{'*', '·', '•', '∘', '○', '❄', '❅'}

	type snowflake struct {
		x, startY, speed int
		ch               rune
	}
	numFlakes := w * h / 10
	if numFlakes < 20 {
		numFlakes = 20
	}
	sf := make([]snowflake, numFlakes)
	for i := range sf {
		sf[i] = snowflake{
			x:      src.Intn(w),
			startY: src.Intn(h * 3),
			speed:  1 + src.Intn(2),
			ch:     flakes[src.Intn(len(flakes))],
		}
	}

	buf := make([][]rune, h)
	for y := range buf {
		buf[y] = make([]rune, w)
		for x := range buf[y] {
			buf[y][x] = ' '
		}
	}

	// accumulation at bottom
	for x := 0; x < w; x++ {
		depth := (src.Intn(3) + frame/30) % 4
		for d := 0; d < depth && h-1-d >= 0; d++ {
			buf[h-1-d][x] = '▓'
		}
	}

	for _, f := range sf {
		// wind drift
		drift := int(math.Sin(float64(frame)/10+float64(f.x)) * 2)
		y := (f.startY + frame*f.speed) % (h + 8)
		x := (f.x + drift) % w
		if x < 0 {
			x += w
		}
		if y >= 0 && y < h-2 && buf[y][x] == ' ' {
			buf[y][x] = f.ch
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := buf[y][x]
			if ch == '▓' {
				sb.WriteString("\033[97m")
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			} else if ch != ' ' {
				sb.WriteString("\033[37m")
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

// RenderDNA produces a frame of a rotating DNA double helix.
func RenderDNA(w, h, frame int) string {
	if w < 10 || h < 4 {
		return ""
	}
	cx := float64(w) / 2
	amplitude := float64(w) / 5
	if amplitude > 15 {
		amplitude = 15
	}
	pairs := []rune{'A', 'T', 'G', 'C'}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		line := make([]rune, w)
		for x := range line {
			line[x] = ' '
		}

		phase := float64(y)*0.6 + float64(frame)*0.15
		x1 := int(cx + amplitude*math.Sin(phase))
		x2 := int(cx - amplitude*math.Sin(phase))

		if x1 < 0 {
			x1 = 0
		}
		if x1 >= w {
			x1 = w - 1
		}
		if x2 < 0 {
			x2 = 0
		}
		if x2 >= w {
			x2 = w - 1
		}

		// draw connecting bars at certain intervals
		if y%3 == 0 {
			lo, hi := x1, x2
			if lo > hi {
				lo, hi = hi, lo
			}
			for x := lo; x <= hi; x++ {
				line[x] = '─'
			}
			pi := (y + frame) % len(pairs)
			mid := (lo + hi) / 2
			if mid >= 0 && mid < w {
				line[mid] = pairs[pi]
			}
		}

		// draw strand positions
		if x1 >= 0 && x1 < w {
			line[x1] = '●'
		}
		if x2 >= 0 && x2 < w {
			line[x2] = '●'
		}

		for x := 0; x < w; x++ {
			ch := line[x]
			switch ch {
			case '●':
				sb.WriteString("\033[96m") // cyan
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			case '─':
				sb.WriteString("\033[90m") // dim
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			case 'A', 'T':
				sb.WriteString("\033[91m") // red
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			case 'G', 'C':
				sb.WriteString("\033[93m") // yellow
				sb.WriteRune(ch)
				sb.WriteString("\033[0m")
			default:
				sb.WriteRune(' ')
			}
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// RenderWave produces a frame of sine wave animation.
func RenderWave(w, h, frame int) string {
	if w < 4 || h < 4 {
		return ""
	}
	screen := make([][]rune, h)
	for y := range screen {
		screen[y] = make([]rune, w)
		for x := range screen[y] {
			screen[y][x] = ' '
		}
	}

	colors := []string{"\033[91m", "\033[93m", "\033[92m", "\033[96m", "\033[94m", "\033[95m"}
	numWaves := 6
	if numWaves > h/2 {
		numWaves = h / 2
	}
	if numWaves < 1 {
		numWaves = 1
	}

	for wi := 0; wi < numWaves; wi++ {
		baseY := float64(h) / float64(numWaves+1) * float64(wi+1)
		amp := float64(h) / float64(numWaves+1) * 0.3
		freq := 0.1 + float64(wi)*0.02
		phaseOff := float64(wi) * 0.8

		for x := 0; x < w; x++ {
			yf := baseY + amp*math.Sin(float64(x)*freq+float64(frame)*0.12+phaseOff)
			y := int(yf)
			if y >= 0 && y < h {
				ch := '~'
				if (x+frame+wi)%4 == 0 {
					ch = '≈'
				}
				screen[y][x] = ch
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := screen[y][x]
			if ch != ' ' {
				// determine which wave this belongs to
				closestWave := 0
				minDist := float64(h)
				for wi := 0; wi < numWaves; wi++ {
					baseY := float64(h) / float64(numWaves+1) * float64(wi+1)
					d := math.Abs(float64(y) - baseY)
					if d < minDist {
						minDist = d
						closestWave = wi
					}
				}
				ci := closestWave % len(colors)
				sb.WriteString(colors[ci])
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

// RenderPlasma produces a frame of a colorful plasma effect.
func RenderPlasma(w, h, frame int) string {
	if w < 4 || h < 4 {
		return ""
	}
	chars := []rune{' ', '░', '▒', '▓', '█', '▓', '▒', '░'}
	t := float64(frame) * 0.08

	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx := float64(x) / float64(w) * 4
			fy := float64(y) / float64(h) * 4

			v := math.Sin(fx + t)
			v += math.Sin(fy + t*0.7)
			v += math.Sin((fx + fy + t) * 0.5)
			v += math.Sin(math.Sqrt(fx*fx+fy*fy+1) + t*0.8)
			v = (v + 4) / 8 // normalize to 0..1

			ci := int(v * float64(len(chars)))
			if ci >= len(chars) {
				ci = len(chars) - 1
			}
			if ci < 0 {
				ci = 0
			}

			// color based on a second function
			hue := math.Mod(v*360+float64(frame)*3, 360)
			var color string
			switch {
			case hue < 60:
				color = "\033[91m" // red
			case hue < 120:
				color = "\033[93m" // yellow
			case hue < 180:
				color = "\033[92m" // green
			case hue < 240:
				color = "\033[96m" // cyan
			case hue < 300:
				color = "\033[94m" // blue
			default:
				color = "\033[95m" // magenta
			}

			sb.WriteString(color)
			sb.WriteRune(chars[ci])
			sb.WriteString("\033[0m")
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// Source snippets for effects.
const (
	SourceMatrix = `// Matrix rain with Bubble Tea
// Each column tracks a falling head position and speed.
// Render bright head char + dimming trail behind it.
// Use lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))`

	SourceFire = `// Fire simulation
// Bottom row = max heat, propagate upward with random cooling.
// Map heat (0-9) to chars: " .:-=+*#%@"
// Color with ANSI: high heat = yellow/white, low = red/dark`

	SourceRain = `// Rain effect with Bubble Tea
// Spawn droplets at random x positions, fall downward.
// Use │ for drops, . for trail, ╨ for splash.
// Color with ANSI blue: \033[94m`

	SourceStarfield = `// Starfield: stars move outward from center
// Each star has (x,y) in [-1,1] and a speed.
// Project to screen coords, scale by distance from center.
// Brightness: closer to edge = brighter (farther traveled).`

	SourceSnow = `// Snow: falling flakes with wind drift and accumulation
// Use sin() for horizontal drift, random speed per flake.
// Accumulate ▓ blocks at bottom over time.
// White ANSI color: \033[37m`

	SourceDNA = `// DNA double helix: two strands oscillate in opposition
// sin(y + frame) gives x offset for each strand.
// Draw connecting bars with base pairs (A-T, G-C).
// Color: cyan strands, red/yellow base pairs.`

	SourceWave = `// Sine waves: multiple colored waves scroll across screen
// Each wave: baseY + amp * sin(x*freq + frame*speed)
// Use ~ and ≈ glyphs, one ANSI color per wave.
// Waves stack vertically with different frequencies.`

	SourcePlasma = `// Plasma: layered sine functions produce organic patterns
// v = sin(x+t) + sin(y+t) + sin((x+y)/2+t) + sin(sqrt(x²+y²)+t)
// Map value to density chars: ░▒▓█▓▒░
// Color with hue rotation based on plasma value.`
)
