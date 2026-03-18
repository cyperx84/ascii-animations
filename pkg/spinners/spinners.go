// Package spinners provides spinner animation definitions.
package spinners

import "time"

// Spinner defines a named spinner with its frame sequence.
type Spinner struct {
	Name   string
	Desc   string
	Frames []string
}

// All returns all available spinner definitions.
func All() []Spinner {
	return []Spinner{
		{"Dots", "Braille dot spinner", []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}},
		{"Line", "Classic line spinner", []string{"|", "/", "-", "\\"}},
		{"Arrow", "Directional arrow spinner", []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}},
		{"Bounce", "Bouncing dot", []string{"⠁", "⠂", "⠄", "⠂"}},
		{"Circle", "Quarter circle spinner", []string{"◐", "◓", "◑", "◒"}},
		{"Square", "Quarter square spinner", []string{"◰", "◳", "◲", "◱"}},
		{"Star", "Twinkling star", []string{"✶", "✸", "✹", "✺", "✹", "✸"}},
		{"Moon", "Moon phase spinner", []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}},
		{"Clock", "Clock face spinner", []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}},
		{"Bar", "Vertical bar bounce", []string{"▁", "▃", "▅", "▇", "▅", "▃"}},
		{"Pulse", "Block pulse", []string{"░", "▒", "▓", "█", "▓", "▒"}},
		{"Grow", "Growing dots", []string{"·", "•", "●", "•"}},
	}
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
