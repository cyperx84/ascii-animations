package cell

import (
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// Glyph ranges that are single-width in every terminal regardless of locale.
// East Asian Ambiguous rules would otherwise make box and block characters
// width 2 under CJK locales, and we want animations to look the same there.
var forcedSingle = [][2]rune{
	{0x2500, 0x257F},   // box drawing
	{0x2580, 0x259F},   // block elements, half blocks, quadrants
	{0x25A0, 0x25FF},   // geometric shapes
	{0x2800, 0x28FF},   // braille
	{0x1FB00, 0x1FBFF}, // legacy computing: sextants, wedges
	{0x1CD00, 0x1CDE5}, // octants (Unicode 16)
}

// Width returns the display width animations should assume for r: 0 for
// control and combining runes, 1 for safe glyphs, 2 for wide runes, and -1
// for runes whose width varies between terminals: emoji, anything with emoji
// presentation, and East Asian Ambiguous runes (·, ×, arrows, Greek) that
// render double-width under CJK locales. Callers should not place -1 or 2 runes in animated
// regions.
func Width(r rune) int {
	if r < 0x20 || r == 0x7F {
		return 0
	}
	if r < 0x7F {
		return 1
	}
	for _, rg := range forcedSingle {
		if r >= rg[0] && r <= rg[1] {
			return 1
		}
	}
	if isEmoji(r) || runewidth.IsAmbiguousWidth(r) {
		return -1
	}
	return uniseg.StringWidth(string(r))
}

// Safe reports whether r is a single-width glyph safe for animation.
func Safe(r rune) bool { return Width(r) == 1 }

func isEmoji(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF:
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols, dingbats: presentation varies
		return true
	case r == 0xFE0F || r == 0x200D:
		return true
	}
	return false
}
