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
)
