package fx

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Banner renders text as large letters in a built-in font, ready for
// Text(banner, colour). Lowercase is drawn as uppercase, letters are one
// column apart and each input line becomes its own block separated by a
// blank row. Runes the font lacks are an error that names them.
//
// Every glyph is single-width and every row of a letter is the same width,
// so banners lay out identically in every terminal.
func Banner(text, font string) (string, error) {
	f, ok := bannerFonts[strings.ToLower(strings.TrimSpace(font))]
	if !ok {
		return "", fmt.Errorf("unknown banner font %q: use one of %s", font, strings.Join(BannerFonts(), ", "))
	}
	text = strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out []string
	for li, line := range strings.Split(text, "\n") {
		if li > 0 {
			out = append(out, "")
		}
		rows := make([]strings.Builder, f.height)
		for i, r := range line {
			g, ok := f.glyphs[unicode.ToUpper(r)]
			if !ok {
				return "", fmt.Errorf("banner font %s has no glyph for %q", font, r)
			}
			for y := range rows {
				if i > 0 {
					rows[y].WriteByte(' ')
				}
				rows[y].WriteString(g[y])
			}
		}
		for y := range rows {
			out = append(out, rows[y].String())
		}
	}
	return strings.Join(out, "\n"), nil
}

// BannerFonts lists the built-in banner fonts, sorted.
func BannerFonts() []string {
	out := make([]string, 0, len(bannerFonts))
	for n := range bannerFonts {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// BannerRunes lists the runes a font can draw, sorted. Lowercase letters are
// also accepted and drawn as their uppercase glyph.
func BannerRunes(font string) []rune {
	f, ok := bannerFonts[strings.ToLower(strings.TrimSpace(font))]
	if !ok {
		return nil
	}
	out := make([]rune, 0, len(f.glyphs))
	for r := range f.glyphs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// BannerGlyph returns the rows of one glyph, for tests and previews.
func BannerGlyph(font string, r rune) ([]string, bool) {
	f, ok := bannerFonts[strings.ToLower(strings.TrimSpace(font))]
	if !ok {
		return nil, false
	}
	g, ok := f.glyphs[r]
	return g, ok
}

type bannerFont struct {
	height int
	glyphs map[rune][]string
}

var bannerFonts = map[string]bannerFont{
	"block": {height: 5, glyphs: blockGlyphs},
	"slim":  {height: 3, glyphs: slimGlyphs},
	"mini":  {height: 3, glyphs: halfBlockFont(miniBits)},
}

// blockGlyphs is a 5×5 full-block font.
var blockGlyphs = map[rune][]string{
	'A':  {"  █  ", " █ █ ", "█████", "█   █", "█   █"},
	'B':  {"████ ", "█   █", "████ ", "█   █", "████ "},
	'C':  {" ████", "█    ", "█    ", "█    ", " ████"},
	'D':  {"████ ", "█   █", "█   █", "█   █", "████ "},
	'E':  {"█████", "█    ", "████ ", "█    ", "█████"},
	'F':  {"█████", "█    ", "████ ", "█    ", "█    "},
	'G':  {" ████", "█    ", "█  ██", "█   █", " ████"},
	'H':  {"█   █", "█   █", "█████", "█   █", "█   █"},
	'I':  {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'J':  {"█████", "    █", "    █", "█   █", " ███ "},
	'K':  {"█   █", "█  █ ", "███  ", "█  █ ", "█   █"},
	'L':  {"█    ", "█    ", "█    ", "█    ", "█████"},
	'M':  {"█   █", "██ ██", "█ █ █", "█   █", "█   █"},
	'N':  {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'O':  {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'P':  {"████ ", "█   █", "████ ", "█    ", "█    "},
	'Q':  {" ███ ", "█   █", "█ █ █", "█  █ ", " ██ █"},
	'R':  {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
	'S':  {" ████", "█    ", " ███ ", "    █", "████ "},
	'T':  {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'U':  {"█   █", "█   █", "█   █", "█   █", " ███ "},
	'V':  {"█   █", "█   █", " █ █ ", " █ █ ", "  █  "},
	'W':  {"█   █", "█   █", "█ █ █", "██ ██", "█   █"},
	'X':  {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
	'Y':  {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "},
	'Z':  {"█████", "   █ ", "  █  ", " █   ", "█████"},
	'0':  {" ███ ", "█  ██", "█ █ █", "██  █", " ███ "},
	'1':  {"  █  ", " ██  ", "  █  ", "  █  ", "█████"},
	'2':  {" ███ ", "█   █", "  ██ ", " █   ", "█████"},
	'3':  {"█████", "   █ ", "  ██ ", "   █ ", "████ "},
	'4':  {"█   █", "█   █", "█████", "    █", "    █"},
	'5':  {"█████", "█    ", "████ ", "    █", "████ "},
	'6':  {" ████", "█    ", "████ ", "█   █", " ███ "},
	'7':  {"█████", "    █", "   █ ", "  █  ", "  █  "},
	'8':  {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9':  {" ███ ", "█   █", " ████", "    █", "████ "},
	' ':  {"   ", "   ", "   ", "   ", "   "},
	'!':  {"█", "█", "█", " ", "█"},
	'.':  {" ", " ", " ", " ", "█"},
	',':  {"  ", "  ", "  ", " █", "█ "},
	':':  {" ", "█", " ", "█", " "},
	';':  {"  ", " █", "  ", " █", "█ "},
	'\'': {"█", "█", " ", " ", " "},
	'"':  {"█ █", "█ █", "   ", "   ", "   "},
	'-':  {"    ", "    ", "████", "    ", "    "},
	'+':  {"     ", "  █  ", "█████", "  █  ", "     "},
	'=':  {"    ", "████", "    ", "████", "    "},
	'_':  {"    ", "    ", "    ", "    ", "████"},
	'?':  {" ███ ", "█   █", "  ██ ", "     ", "  █  "},
	'/':  {"    █", "   █ ", "  █  ", " █   ", "█    "},
	'(':  {" █", "█ ", "█ ", "█ ", " █"},
	')':  {"█ ", " █", " █", " █", "█ "},
	'#':  {" █ █ ", "█████", " █ █ ", "█████", " █ █ "},
	'*':  {"     ", "█ █ █", " ███ ", "█ █ █", "     "},
	'<':  {"   █", "  █ ", " █  ", "  █ ", "   █"},
	'>':  {"█   ", " █  ", "  █ ", " █  ", "█   "},
}

// slimGlyphs is a 3-row light box-drawing font.
var slimGlyphs = map[rune][]string{
	'A':  {"┌─┐", "├─┤", "╵ ╵"},
	'B':  {"┌┐ ", "├┴┐", "└─┘"},
	'C':  {"┌─╴", "│  ", "└─╴"},
	'D':  {"╶┬┐", " ││", "╶┴┘"},
	'E':  {"┌─╴", "├╴ ", "└─╴"},
	'F':  {"┌─╴", "├╴ ", "╵  "},
	'G':  {"┌─╴", "│╶┐", "└─┘"},
	'H':  {"╷ ╷", "├─┤", "╵ ╵"},
	'I':  {"╷", "│", "╵"},
	'J':  {"  ╷", "  │", "└─┘"},
	'K':  {"╷┌ ", "├┴┐", "╵ ╵"},
	'L':  {"╷  ", "│  ", "└─╴"},
	'M':  {"┌┬┐", "│││", "╵ ╵"},
	'N':  {"┌┐╷", "│└┤", "╵ ╵"},
	'O':  {"┌─┐", "│ │", "└─┘"},
	'P':  {"┌─┐", "├─┘", "╵  "},
	'Q':  {"┌─┐", "│┐│", "└┴┘"},
	'R':  {"┌─┐", "├┬┘", "╵└╴"},
	'S':  {"┌─┐", "└─┐", "└─┘"},
	'T':  {"╶┬╴", " │ ", " ╵ "},
	'U':  {"╷ ╷", "│ │", "└─┘"},
	'V':  {"╷ ╷", "│┌┘", "└┘ "},
	'W':  {"╷ ╷", "│╷│", "└┴┘"},
	'X':  {"╲ ╱", " ╳ ", "╱ ╲"},
	'Y':  {"╷ ╷", "└┬┘", " ╵ "},
	'Z':  {"╶─┐", "┌─┘", "└─╴"},
	'0':  {"┌─┐", "│╱│", "└─┘"},
	'1':  {"╶┐ ", " │ ", "╶┴╴"},
	'2':  {"╶─┐", "┌─┘", "└─╴"},
	'3':  {"╶─┐", " ─┤", "╶─┘"},
	'4':  {"╷ ╷", "└─┤", "  ╵"},
	'5':  {"┌─╴", "└─┐", "╶─┘"},
	'6':  {"┌─╴", "├─┐", "└─┘"},
	'7':  {"╶─┐", "  │", "  ╵"},
	'8':  {"┌─┐", "├─┤", "└─┘"},
	'9':  {"┌─┐", "└─┤", "╶─┘"},
	' ':  {"  ", "  ", "  "},
	'!':  {"╷", "│", "╹"},
	'.':  {" ", " ", "╹"},
	',':  {" ", " ", "┘"},
	':':  {" ", "╵", "╵"},
	';':  {" ", "╵", "┘"},
	'\'': {"╷", " ", " "},
	'"':  {"╷╷", "  ", "  "},
	'-':  {"  ", "──", "  "},
	'+':  {"   ", "╶┼╴", " ╵ "},
	'=':  {"  ", "──", "──"},
	'_':  {"  ", "  ", "──"},
	'?':  {"╶─┐", " ┌┘", " ╹ "},
	'/':  {"  ╱", " ╱ ", "╱  "},
	'(':  {"┌", "│", "└"},
	')':  {"┐", "│", "┘"},
	'#':  {"┼┼", "┼┼", "  "},
	'*':  {"   ", "╲│╱", "╱│╲"},
	'<':  {" ╱", "╱ ", "╲ "},
	'>':  {"╲ ", " ╲", " ╱"},
}

// miniBits is a pixel font, five pixels tall, drawn in half blocks as three
// rows. '#' is a lit pixel.
var miniBits = map[rune][]string{
	'A':  {".#.", "#.#", "###", "#.#", "#.#"},
	'B':  {"##.", "#.#", "##.", "#.#", "##."},
	'C':  {".##", "#..", "#..", "#..", ".##"},
	'D':  {"##.", "#.#", "#.#", "#.#", "##."},
	'E':  {"###", "#..", "##.", "#..", "###"},
	'F':  {"###", "#..", "##.", "#..", "#.."},
	'G':  {".##", "#..", "#.#", "#.#", ".##"},
	'H':  {"#.#", "#.#", "###", "#.#", "#.#"},
	'I':  {"###", ".#.", ".#.", ".#.", "###"},
	'J':  {"..#", "..#", "..#", "#.#", ".#."},
	'K':  {"#.#", "#.#", "##.", "#.#", "#.#"},
	'L':  {"#..", "#..", "#..", "#..", "###"},
	'M':  {"#...#", "##.##", "#.#.#", "#...#", "#...#"},
	'N':  {"#..#", "##.#", "#.##", "#..#", "#..#"},
	'O':  {".#.", "#.#", "#.#", "#.#", ".#."},
	'P':  {"##.", "#.#", "##.", "#..", "#.."},
	'Q':  {".#.", "#.#", "#.#", "#.#", ".##"},
	'R':  {"##.", "#.#", "##.", "#.#", "#.#"},
	'S':  {".##", "#..", ".#.", "..#", "##."},
	'T':  {"###", ".#.", ".#.", ".#.", ".#."},
	'U':  {"#.#", "#.#", "#.#", "#.#", "###"},
	'V':  {"#.#", "#.#", "#.#", "#.#", ".#."},
	'W':  {"#...#", "#...#", "#.#.#", "##.##", "#...#"},
	'X':  {"#.#", "#.#", ".#.", "#.#", "#.#"},
	'Y':  {"#.#", "#.#", ".#.", ".#.", ".#."},
	'Z':  {"###", "..#", ".#.", "#..", "###"},
	'0':  {"###", "#.#", "#.#", "#.#", "###"},
	'1':  {".#.", "##.", ".#.", ".#.", "###"},
	'2':  {"##.", "..#", ".#.", "#..", "###"},
	'3':  {"##.", "..#", ".#.", "..#", "##."},
	'4':  {"#.#", "#.#", "###", "..#", "..#"},
	'5':  {"###", "#..", "##.", "..#", "##."},
	'6':  {".##", "#..", "###", "#.#", "###"},
	'7':  {"###", "..#", ".#.", ".#.", ".#."},
	'8':  {"###", "#.#", "###", "#.#", "###"},
	'9':  {"###", "#.#", "###", "..#", "##."},
	' ':  {"..", "..", "..", "..", ".."},
	'!':  {"#", "#", "#", ".", "#"},
	'.':  {".", ".", ".", ".", "#"},
	',':  {"..", "..", "..", ".#", "#."},
	':':  {".", "#", ".", "#", "."},
	';':  {"..", ".#", "..", ".#", "#."},
	'\'': {"#", "#", ".", ".", "."},
	'"':  {"#.#", "#.#", "...", "...", "..."},
	'-':  {"...", "...", "###", "...", "..."},
	'+':  {"...", ".#.", "###", ".#.", "..."},
	'=':  {"...", "###", "...", "###", "..."},
	'_':  {"...", "...", "...", "...", "###"},
	'?':  {"##.", "..#", ".#.", "...", ".#."},
	'/':  {"..#", "..#", ".#.", "#..", "#.."},
	'(':  {".#", "#.", "#.", "#.", ".#"},
	')':  {"#.", ".#", ".#", ".#", "#."},
	'#':  {"#.#", "###", "#.#", "###", "#.#"},
	'*':  {"...", "#.#", ".#.", "#.#", "..."},
	'<':  {"..#", ".#.", "#..", ".#.", "..#"},
	'>':  {"#..", ".#.", "..#", ".#.", "#.."},
}

// halfBlockFont packs a five-pixel-tall bitmap font into three rows of half
// blocks: pixel rows 0-1, 2-3 and 4 (with an empty sixth row).
func halfBlockFont(bits map[rune][]string) map[rune][]string {
	out := make(map[rune][]string, len(bits))
	lit := func(rows []string, y, x int) bool { return y < len(rows) && rows[y][x] == '#' }
	for r, rows := range bits {
		w := len(rows[0])
		g := make([]string, 3)
		for cy := range g {
			var sb strings.Builder
			for x := 0; x < w; x++ {
				top, bot := lit(rows, cy*2, x), lit(rows, cy*2+1, x)
				switch {
				case top && bot:
					sb.WriteRune('█')
				case top:
					sb.WriteRune('▀')
				case bot:
					sb.WriteRune('▄')
				default:
					sb.WriteByte(' ')
				}
			}
			g[cy] = sb.String()
		}
		out[r] = g
	}
	return out
}
