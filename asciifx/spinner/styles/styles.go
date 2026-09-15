// Package styles holds the built-in spinner frame sets.
//
// It is a leaf package on purpose: both the spinner effect (which the headless
// CLI uses) and the Bubble Tea spinner component need these frames, and neither
// should have to depend on the other. Nothing here imports Bubble Tea, so
// `asciifx render spinner` does not link a TUI framework to read a table of
// characters.
package styles

import (
	"sort"
	"time"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// Style is one frame set. Every frame has the same number of runes, so a label
// drawn after the glyph never jitters. Interval is the time per frame at speed
// 1, which is the quantity bubbles/spinner calls FPS.
type Style struct {
	Frames   []string
	Interval time.Duration
}

// all are single-width glyphs only: no emoji, and no East Asian Ambiguous runes
// (arrows, ·, ×), which draw double-width under CJK locales. cell.Safe enforces
// this at init.
var all = map[string]Style{
	"dots":          {[]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, 80 * time.Millisecond},
	"dots2":         {[]string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}, 80 * time.Millisecond},
	"line":          {[]string{"|", "/", "-", `\`}, 130 * time.Millisecond},
	"pipe":          {[]string{"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"}, 100 * time.Millisecond},
	"arc":           {[]string{"◜", "◠", "◝", "◞", "◡", "◟"}, 100 * time.Millisecond},
	"circle":        {[]string{"◐", "◓", "◑", "◒"}, 120 * time.Millisecond},
	"bounce":        {[]string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}, 100 * time.Millisecond},
	"pulse":         {[]string{"░", "▒", "▓", "█", "▓", "▒"}, 110 * time.Millisecond},
	"bar":           {[]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}, 70 * time.Millisecond},
	"blocks":        {[]string{"▖", "▌", "▘", "▀", "▝", "▐", "▗", "▄"}, 100 * time.Millisecond},
	"braille-snake": {[]string{"⠏", "⠛", "⠹", "⢸", "⣰", "⣤", "⣆", "⡇"}, 80 * time.Millisecond},
	"arrow":         {[]string{"▲", "▶", "▼", "◀"}, 130 * time.Millisecond},
	"toggle":        {[]string{"⊶", "⊷"}, 250 * time.Millisecond},
	"star":          {[]string{".", "+", "*", "+"}, 120 * time.Millisecond},
}

// All returns the built-in styles by name.
func All() map[string]Style { return all }

// Names lists the style names, sorted.
func Names() []string {
	out := make([]string, 0, len(all))
	for n := range all {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Lookup returns the style with the given name.
func Lookup(name string) (Style, bool) {
	s, ok := all[name]
	return s, ok
}

func init() {
	// Fail at startup, like fx.Register, rather than on a user's screen.
	for name, s := range all {
		if len(s.Frames) == 0 || s.Interval <= 0 {
			panic("asciifx: spinner style " + name + " is empty")
		}
		n := len([]rune(s.Frames[0]))
		for _, fr := range s.Frames {
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
