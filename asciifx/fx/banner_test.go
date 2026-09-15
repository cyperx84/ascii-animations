package fx

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

func TestBannerGlyphs(t *testing.T) {
	for _, font := range BannerFonts() {
		f := bannerFonts[font]
		runes := BannerRunes(font)
		for _, want := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !.,:-?'" {
			if _, ok := f.glyphs[want]; !ok {
				t.Errorf("font %s lacks %q", font, want)
			}
		}
		for _, r := range runes {
			rows, _ := BannerGlyph(font, r)
			if len(rows) != f.height {
				t.Errorf("font %s %q: %d rows, want %d", font, r, len(rows), f.height)
				continue
			}
			w := len([]rune(rows[0]))
			if w == 0 {
				t.Errorf("font %s %q: empty glyph", font, r)
			}
			for i, row := range rows {
				if n := len([]rune(row)); n != w {
					t.Errorf("font %s %q row %d: width %d, want %d", font, r, i, n, w)
				}
				for _, g := range row {
					if !cell.Safe(g) {
						t.Errorf("font %s %q: unsafe glyph %q (U+%04X)", font, r, g, g)
					}
				}
			}
		}
	}
}

func TestBanner(t *testing.T) {
	for _, font := range BannerFonts() {
		s, err := Banner("Hi 42!\nok", font)
		if err != nil {
			t.Fatalf("%s: %v", font, err)
		}
		lines := strings.Split(s, "\n")
		h := bannerFonts[font].height
		if len(lines) != 2*h+1 {
			t.Fatalf("%s: %d lines, want %d", font, len(lines), 2*h+1)
		}
		for i := 1; i < h; i++ {
			if len([]rune(lines[i])) != len([]rune(lines[0])) {
				t.Errorf("%s: ragged block rows\n%s", font, s)
			}
		}
	}
	if _, err := Banner("~", "block"); err == nil {
		t.Error("want error for unsupported rune")
	}
	if _, err := Banner("A", "nope"); err == nil {
		t.Error("want error for unknown font")
	}
}
