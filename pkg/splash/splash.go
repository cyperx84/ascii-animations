// Package splash provides animated splash screen demos.
package splash

import "strings"

// TypingFrames generates frames for a typewriter text effect.
func TypingFrames(text string) []string {
	frames := make([]string, len(text)+5)
	for i := 0; i <= len(text); i++ {
		cursor := "█"
		if i == len(text) {
			cursor = " "
		}
		frames[i] = text[:i] + cursor
	}
	// blinking cursor hold
	for i := len(text) + 1; i < len(frames); i++ {
		if i%2 == 0 {
			frames[i] = text + "█"
		} else {
			frames[i] = text + " "
		}
	}
	return frames
}

// ExpandBorderFrames generates frames for a line-by-line logo reveal.
func ExpandBorderFrames() []string {
	logo := []string{
		"┌─────────────────────┐",
		"│  ASCII  ANIMATIONS  │",
		"│     ◆ ◆ ◆ ◆ ◆      │",
		"│   Go + Bubble Tea   │",
		"└─────────────────────┘",
	}

	totalFrames := len(logo) + 3
	frames := make([]string, totalFrames)
	for f := 0; f < totalFrames; f++ {
		visible := f + 1
		if visible > len(logo) {
			visible = len(logo)
		}
		var sb strings.Builder
		for i := 0; i < visible; i++ {
			sb.WriteString(logo[i])
			if i < visible-1 {
				sb.WriteByte('\n')
			}
		}
		frames[f] = sb.String()
	}
	return frames
}

// FadeInFrames generates frames for a density-character fade-in effect.
func FadeInFrames() []string {
	final := []string{
		"  ╔═══════════════╗  ",
		"  ║  POWERED  BY  ║  ",
		"  ║   BUBBLE TEA  ║  ",
		"  ╚═══════════════╝  ",
	}
	densityChars := []rune{'░', '▒', '▓', '█'}

	numFrames := len(densityChars) + 2
	frames := make([]string, numFrames)

	for f := 0; f < len(densityChars); f++ {
		var sb strings.Builder
		ch := densityChars[f]
		for _, line := range final {
			for range line {
				sb.WriteRune(ch)
			}
			sb.WriteByte('\n')
		}
		frames[f] = strings.TrimRight(sb.String(), "\n")
	}

	var sb strings.Builder
	for i, line := range final {
		sb.WriteString(line)
		if i < len(final)-1 {
			sb.WriteByte('\n')
		}
	}
	finalStr := sb.String()
	frames[len(densityChars)] = finalStr
	frames[len(densityChars)+1] = finalStr

	return frames
}

// SpinnerSplashFrames generates frames with a spinner leading into a logo reveal.
func SpinnerSplashFrames() []string {
	spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	logo := []string{
		"╔═══════════════════════╗",
		"║   ★ ASCII SHOWCASE ★  ║",
		"║      Loading...       ║",
		"╚═══════════════════════╝",
	}
	revealed := []string{
		"╔═══════════════════════╗",
		"║   ★ ASCII SHOWCASE ★  ║",
		"║       ✓ Ready!        ║",
		"╚═══════════════════════╝",
	}

	frames := make([]string, 0, len(spinChars)*2+4)

	// spinner phase
	for i := 0; i < len(spinChars)*2; i++ {
		ch := spinChars[i%len(spinChars)]
		var sb strings.Builder
		sb.WriteString("\n\n")
		sb.WriteString("        " + ch + " Loading...\n")
		frames = append(frames, sb.String())
	}

	// reveal logo line by line
	for lines := 1; lines <= len(logo); lines++ {
		var sb strings.Builder
		for i := 0; i < lines; i++ {
			sb.WriteString(logo[i])
			sb.WriteByte('\n')
		}
		frames = append(frames, sb.String())
	}

	// swap to "Ready!"
	var sb strings.Builder
	for _, line := range revealed {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	revealedStr := sb.String()
	frames = append(frames, revealedStr, revealedStr)

	return frames
}

// GlitchFrames generates frames for a glitch/corruption text effect.
func GlitchFrames() []string {
	text := "ASCII ANIMATIONS"
	glitchChars := []rune{'#', '@', '!', '%', '&', '/', '?', '~', '^', '*'}
	runes := []rune(text)

	frames := make([]string, 0, 30)

	// full glitch to clear
	for intensity := len(runes); intensity >= 0; intensity-- {
		line := make([]rune, len(runes))
		copy(line, runes)
		for i := 0; i < intensity && i < len(line); i++ {
			idx := (i * 7) % len(line)
			line[idx] = glitchChars[(i+intensity)%len(glitchChars)]
		}
		frames = append(frames, string(line))
	}

	// hold clean text
	for i := 0; i < 5; i++ {
		frames = append(frames, text)
	}

	return frames
}

// ScanLineFrames generates frames for a top-to-bottom scan line reveal.
func ScanLineFrames() []string {
	art := []string{
		"┌──────────────────────┐",
		"│                      │",
		"│   ▄▀▀▀▀▀▀▀▀▀▀▀▄     │",
		"│   █ ASCII ANIM █     │",
		"│   ▀▄▄▄▄▄▄▄▄▄▄▄▀     │",
		"│                      │",
		"│   Go + Bubble Tea    │",
		"│   + Lip Gloss        │",
		"│                      │",
		"└──────────────────────┘",
	}

	frames := make([]string, 0, len(art)+4)

	for scanY := 0; scanY <= len(art); scanY++ {
		var sb strings.Builder
		for y := 0; y < len(art); y++ {
			if y < scanY {
				sb.WriteString(art[y])
			} else if y == scanY {
				// scan line: bright bar
				sb.WriteString("\033[7m") // reverse video
				sb.WriteString(strings.Repeat("━", len(art[0])))
				sb.WriteString("\033[0m")
			} else {
				sb.WriteString(strings.Repeat(" ", len(art[0])))
			}
			if y < len(art)-1 {
				sb.WriteByte('\n')
			}
		}
		frames = append(frames, sb.String())
	}

	// hold final frame
	var sb strings.Builder
	for i, line := range art {
		sb.WriteString(line)
		if i < len(art)-1 {
			sb.WriteByte('\n')
		}
	}
	finalStr := sb.String()
	frames = append(frames, finalStr, finalStr, finalStr)

	return frames
}

// Source snippets for splash screens.
const (
	SourceTyping = `// Typing effect: reveal one character per tick
// Track position in string, append cursor glyph
// Use tea.Tick for timing, lipgloss for styling`

	SourceExpand = `// Expanding border: reveal logo lines progressively
// Use lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
// Animate by increasing visible content each tick`

	SourceFade = `// Fade in: cycle through density characters
// ░ → ▒ → ▓ → █ → reveal final text
// Each frame replaces all chars with next density level`

	SourceSpinner = `// Spinner splash: show spinner frames, then reveal logo
// Phase 1: cycle through braille spinner chars
// Phase 2: reveal logo box line by line
// Phase 3: swap "Loading..." → "✓ Ready!"`

	SourceGlitch = `// Glitch effect: progressively un-corrupt text
// Start with random chars replacing the text
// Each frame: reduce glitch intensity by 1
// Finally hold the clean, revealed text`

	SourceScanLine = `// Scan line reveal: bright bar sweeps top to bottom
// Above the bar: revealed content
// The bar itself: reverse-video ━━━ line
// Below the bar: blank space`
)
