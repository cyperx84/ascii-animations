// Package spinners provides spinner animation definitions.
package spinners

import "time"

// Spinner defines a named spinner with its frame sequence.
type Spinner struct {
	Name     string
	Desc     string
	Category string
	Frames   []string
}

// All returns all available spinner definitions.
func All() []Spinner {
	return []Spinner{
		// ── Braille ──
		{"Dots", "Braille dot spinner", "braille", []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}},
		{"Bounce", "Bouncing braille dot", "braille", []string{"⠁", "⠂", "⠄", "⠂"}},
		{"Dots2", "Braille sweep", "braille", []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}},
		{"Dots3", "Braille scroll", "braille", []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"}},
		{"BrailleSnake", "Braille snake crawl", "braille", []string{"⠏", "⠛", "⠹", "⢸", "⣰", "⣤", "⣆", "⡇"}},

		// ── Classic ──
		{"Line", "Classic line spinner", "classic", []string{"|", "/", "-", "\\"}},
		{"Grow", "Growing dots", "classic", []string{"·", "•", "●", "•"}},
		{"Toggle", "Toggle switch", "classic", []string{"⊶", "⊷"}},
		{"Pipe", "Pipe spinner", "classic", []string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"}},

		// ── Arrows ──
		{"Arrow", "Directional arrow spinner", "arrows", []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}},
		{"Arrow2", "Simple arrow spinner", "arrows", []string{"⬆️ ", "↗️ ", "➡️ ", "↘️ ", "⬇️ ", "↙️ ", "⬅️ ", "↖️ "}},
		{"Bounce2", "Bouncing arrow", "arrows", []string{"▹▹▹▹▹", "▸▹▹▹▹", "▹▸▹▹▹", "▹▹▸▹▹", "▹▹▹▸▹", "▹▹▹▹▸"}},

		// ── Blocks ──
		{"Bar", "Vertical bar bounce", "blocks", []string{"▁", "▃", "▅", "▇", "▅", "▃"}},
		{"Pulse", "Block pulse", "blocks", []string{"░", "▒", "▓", "█", "▓", "▒"}},
		{"BarH", "Horizontal bar fill", "blocks", []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█", "▉", "▊", "▋", "▌", "▍", "▎"}},
		{"BlockScroll", "Scrolling blocks", "blocks", []string{"█▒░", "░█▒", "▒░█"}},

		// ── Shapes ──
		{"Circle", "Quarter circle spinner", "shapes", []string{"◐", "◓", "◑", "◒"}},
		{"Square", "Quarter square spinner", "shapes", []string{"◰", "◳", "◲", "◱"}},
		{"Star", "Twinkling star", "shapes", []string{"✶", "✸", "✹", "✺", "✹", "✸"}},
		{"Diamond", "Rotating diamond", "shapes", []string{"◇", "◈", "◆", "◈"}},
		{"Triangle", "Rotating triangle", "shapes", []string{"◢", "◣", "◤", "◥"}},

		// ── Points ──
		{"BouncingBall", "Bouncing ball", "points", []string{"( ●    )", "(  ●   )", "(   ●  )", "(    ● )", "(     ●)", "(    ● )", "(   ●  )", "(  ●   )", "( ●    )", "(●     )"}},
		{"Dots4", "Ellipsis dots", "points", []string{".  ", ".. ", "...", "   "}},
		{"Flip", "Character flip", "points", []string{"_", "_", "_", "-", "`", "`", "'", "´", "-", "_", "_", "_"}},

		// ── Emoji ──
		{"Moon", "Moon phase spinner", "emoji", []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}},
		{"Clock", "Clock face spinner", "emoji", []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}},
		{"Earth", "Spinning earth", "emoji", []string{"🌍", "🌎", "🌏"}},
		{"Weather", "Weather cycle", "emoji", []string{"☀️ ", "🌤️", "⛅", "🌥️", "☁️ ", "🌧️", "⛈️ ", "🌩️"}},
	}
}

// Categories returns the unique spinner categories in order.
func Categories() []string {
	return []string{"braille", "classic", "arrows", "blocks", "shapes", "points", "emoji"}
}

// Interval is the default tick speed for spinners.
const Interval = 80 * time.Millisecond

// SourceSnippet is example source code shown for spinners.
const SourceSnippet = `s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
s.Suffix = " Loading..."
s.Start()
defer s.Stop()
// Do work...
time.Sleep(2 * time.Second)`
