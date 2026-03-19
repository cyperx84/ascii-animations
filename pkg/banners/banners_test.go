package banners

import (
	"strings"
	"testing"
)

func TestAllFontsCount(t *testing.T) {
	fonts := AllFonts()
	if len(fonts) < 10 {
		t.Errorf("expected at least 10 fonts, got %d", len(fonts))
	}
}

func TestAllFontsHaveNames(t *testing.T) {
	for _, f := range AllFonts() {
		if f.Name == "" {
			t.Error("font has empty name")
		}
		if f.Chars == nil || len(f.Chars) == 0 {
			t.Errorf("font %q has no characters", f.Name)
		}
	}
}

func TestRenderBasic(t *testing.T) {
	fonts := AllFonts()
	for _, f := range fonts {
		t.Run(f.Name, func(t *testing.T) {
			result := Render("HELLO", f.Chars)
			if result == "" {
				t.Error("expected non-empty render output")
			}
			lines := strings.Split(result, "\n")
			if len(lines) != 5 {
				t.Errorf("expected 5 lines, got %d", len(lines))
			}
		})
	}
}

func TestRenderEmpty(t *testing.T) {
	fonts := AllFonts()
	if len(fonts) == 0 {
		t.Fatal("no fonts available")
	}
	result := Render("", fonts[0].Chars)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("empty input should still produce 5 lines, got %d", len(lines))
	}
}

func TestRenderUnknownChars(t *testing.T) {
	// unknown chars should fall back to space glyph
	fonts := AllFonts()
	if len(fonts) == 0 {
		t.Fatal("no fonts")
	}
	result := Render("~@#", fonts[0].Chars)
	if result == "" {
		t.Error("should produce output even for unknown chars")
	}
}

func TestRenderLowercase(t *testing.T) {
	fonts := AllFonts()
	if len(fonts) == 0 {
		t.Fatal("no fonts")
	}
	upper := Render("ABC", fonts[0].Chars)
	lower := Render("abc", fonts[0].Chars)
	if upper != lower {
		t.Error("lowercase should render same as uppercase")
	}
}

func TestFontSpaceGlyph(t *testing.T) {
	for _, f := range AllFonts() {
		if _, ok := f.Chars[' ']; !ok {
			t.Errorf("font %q missing space glyph", f.Name)
		}
	}
}
