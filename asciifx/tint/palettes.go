package tint

import (
	"fmt"
	"sort"
	"strings"
)

// Palettes are named gradients effects accept through their "palette" param.
// Ordered dark to bright where that makes sense, so effects can map
// intensity straight onto them.
var Palettes = map[string]Gradient{
	"aurora":     hexes("#0b1026", "#1b3a6b", "#1fa39a", "#7ee081", "#e8f7a1"),
	"fire":       hexes("#070707", "#47070f", "#9f1d0b", "#df5714", "#efb31f", "#fff0b3", "#ffffff"),
	"matrix":     hexes("#020a04", "#0b3d17", "#159a3c", "#3cf06e", "#d6ffe0"),
	"ocean":      hexes("#03071e", "#023e8a", "#0096c7", "#48cae4", "#caf0f8"),
	"synthwave":  hexes("#1a0b2e", "#5b1a8a", "#e0218a", "#ff8e3c", "#ffe66d"),
	"sunset":     hexes("#2d1b3d", "#8e3b5f", "#e76f51", "#f4a261", "#ffe8a3"),
	"dracula":    hexes("#282a36", "#6272a4", "#bd93f9", "#ff79c6", "#f8f8f2"),
	"nord":       hexes("#2e3440", "#4c566a", "#5e81ac", "#88c0d0", "#eceff4"),
	"catppuccin": hexes("#1e1e2e", "#585b70", "#89b4fa", "#cba6f7", "#f5e0dc"),
	"gruvbox":    hexes("#282828", "#665c54", "#d65d0e", "#fabd2f", "#fbf1c7"),
	"ice":        hexes("#0a1128", "#274c77", "#6096ba", "#a3cef1", "#ffffff"),
	"mono":       hexes("#111111", "#444444", "#888888", "#cccccc", "#ffffff"),
	"rainbow":    hexes("#ff3b3b", "#ffb13b", "#f5ff3b", "#3bff72", "#3bc8ff", "#8a3bff", "#ff3bd1"),
}

func hexes(stops ...string) Gradient {
	g := make(Gradient, len(stops))
	for i, s := range stops {
		g[i] = MustHex(s)
	}
	return g
}

// PaletteNames lists the built-in palette names, sorted.
func PaletteNames() []string {
	names := make([]string, 0, len(Palettes))
	for n := range Palettes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ParsePalette accepts a palette name or a comma-separated list of hex stops.
func ParsePalette(s string) (Gradient, error) {
	if g, ok := Palettes[strings.ToLower(strings.TrimSpace(s))]; ok {
		return g, nil
	}
	if !strings.Contains(s, "#") && !strings.Contains(s, ",") {
		return nil, fmt.Errorf("unknown palette %q: use one of %s, or hex stops like \"#ff0000,#0000ff\"",
			s, strings.Join(PaletteNames(), ", "))
	}
	var g Gradient
	for _, part := range strings.Split(s, ",") {
		c, err := Hex(part)
		if err != nil {
			return nil, err
		}
		g = append(g, c)
	}
	return g, nil
}
