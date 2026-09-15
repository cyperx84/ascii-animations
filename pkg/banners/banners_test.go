package banners

import (
	"strings"
	"testing"
	"unicode"
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

// TestRenderableMatchesWhatRenderCanDraw is the contract the banner text field
// depends on: a rune is worth accepting exactly when Render turns it into
// something other than a blank.
func TestRenderableMatchesWhatRenderCanDraw(t *testing.T) {
	font := AllFonts()[0].Chars
	for r := rune(32); r < 0x2500; r++ {
		if !Renderable(r) {
			continue
		}
		if _, ok := font[unicode.ToUpper(r)]; !ok {
			t.Errorf("Renderable(%q) is true but the first font has no glyph", r)
		}
	}
	// Everything the fonts cover is accepted.
	for _, r := range []rune(" !0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		if !Renderable(r) {
			t.Errorf("Renderable(%q) is false for a glyph every font has", r)
		}
	}
	// Lowercase is drawable because Render uppercases first.
	for _, r := range []rune("az09") {
		if !Renderable(r) {
			t.Errorf("Renderable(%q) is false, but Render uppercases it", r)
		}
	}
	// A rune with no glyph renders as a blank, so it must be refused.
	for _, r := range []rune{'é', '#', '?', '@', '-', '_', '.', '\n'} {
		if Renderable(r) {
			t.Errorf("Renderable(%q) is true but Render would draw a blank", r)
		}
	}
	// The fallback really is blank, which is what makes accepting an
	// unrenderable rune invisible rather than obviously wrong: the field looks
	// like it did nothing while the rune sits in the string.
	if got, want := Render("é", font), Render(" ", font); got != want {
		t.Errorf("an unrenderable rune does not fall back to a blank:\n%q\n%q", got, want)
	}
	// Uppercasing is what makes lowercase acceptable, so prove the link rather
	// than assuming it.
	if Render("az", font) != Render("AZ", font) {
		t.Error("Render does not uppercase, so Renderable must not either")
	}
}
