package effects

import (
	"sort"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// spinnerStyle is one frame set. Every frame of a style has the same number
// of runes, so a label after it never jitters. Interval is seconds per frame
// at speed 1.
type spinnerStyle struct {
	frames   []string
	interval float64
}

// spinnerStyles are single-width glyphs only: no emoji, and no East Asian
// Ambiguous runes (arrows, ·, ×), which draw double-width under CJK locales.
// cell.Safe enforces this at init.
var spinnerStyles = map[string]spinnerStyle{
	"dots":          {frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, interval: 0.08},
	"dots2":         {frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}, interval: 0.08},
	"line":          {frames: []string{"|", "/", "-", `\`}, interval: 0.13},
	"pipe":          {frames: []string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"}, interval: 0.1},
	"arc":           {frames: []string{"◜", "◠", "◝", "◞", "◡", "◟"}, interval: 0.1},
	"circle":        {frames: []string{"◐", "◓", "◑", "◒"}, interval: 0.12},
	"bounce":        {frames: []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}, interval: 0.1},
	"pulse":         {frames: []string{"░", "▒", "▓", "█", "▓", "▒"}, interval: 0.11},
	"bar":           {frames: []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}, interval: 0.07},
	"blocks":        {frames: []string{"▖", "▌", "▘", "▀", "▝", "▐", "▗", "▄"}, interval: 0.1},
	"braille-snake": {frames: []string{"⠏", "⠛", "⠹", "⢸", "⣰", "⣤", "⣆", "⡇"}, interval: 0.08},
	"arrow":         {frames: []string{"▲", "▶", "▼", "◀"}, interval: 0.13},
	"toggle":        {frames: []string{"⊶", "⊷"}, interval: 0.25},
	"star":          {frames: []string{".", "+", "*", "+"}, interval: 0.12},
}

// spinnerStyleNames lists the styles, sorted, for the style enum.
func spinnerStyleNames() []string {
	out := make([]string, 0, len(spinnerStyles))
	for n := range spinnerStyles {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func init() {
	// Fail at startup, like fx.Register, rather than on a user's screen.
	for name, s := range spinnerStyles {
		if len(s.frames) == 0 || s.interval <= 0 {
			panic("asciifx: spinner style " + name + " is empty")
		}
		n := len([]rune(s.frames[0]))
		for _, fr := range s.frames {
			if len([]rune(fr)) != n {
				panic("asciifx: spinner style " + name + " has frames of different widths")
			}
			for _, r := range fr {
				if !cell.Safe(r) {
					panic("asciifx: spinner style " + name + " uses unsafe glyph " + string(r))
				}
			}
		}
	}
}
