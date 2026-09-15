package term

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func TestDetectTransportAndRenderingPrefs(t *testing.T) {
	cases := []struct {
		name    string
		env     func(string) string
		fps     int
		dither  tint.Dither
		noSync  bool
		syncSet bool
	}{
		{"local truecolor", env("COLORTERM", "truecolor", "TERM", "xterm-256color"), 30, tint.NoDither, false, false},
		{"256 dithers", env("TERM", "xterm-256color"), 30, tint.Bayer8, false, false},
		{"16 dithers", env("TERM", "xterm"), 30, tint.Bayer8, false, false},
		{"ssh is capped", env("SSH_CONNECTION", "10.0.0.1 1 2 3", "TERM", "xterm-256color"), 15, tint.Bayer8, false, false},
		{"tmux is capped", env("TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
		{"screen is capped", env("TERM", "screen.xterm-256color"), 15, tint.Bayer8, false, false},
		{"explicit fps wins", env("SSH_CONNECTION", "x", "ASCIIFX_FPS", "60", "TERM", "xterm"), 60, tint.Bayer8, false, false},
		{"sync off", env("ASCIIFX_SYNC", "0", "TERM", "xterm"), 30, tint.Bayer8, true, true},
		{"sync on is still recorded", env("ASCIIFX_SYNC", "1", "TERM", "xterm"), 30, tint.Bayer8, false, true},
		{"dither off", env("ASCIIFX_DITHER", "none", "TERM", "xterm"), 30, tint.NoDither, false, false},
		{"no colour disables dither", env("NO_COLOR", "1", "TERM", "xterm"), 30, tint.NoDither, false, false},
	}
	for _, c := range cases {
		got := detect(true, c.env)
		if got.FPS != c.fps {
			t.Errorf("%s: FPS = %d, want %d", c.name, got.FPS, c.fps)
		}
		if got.Dither != c.dither {
			t.Errorf("%s: Dither = %v, want %v", c.name, got.Dither, c.dither)
		}
		if got.NoSync != c.noSync {
			t.Errorf("%s: NoSync = %v, want %v", c.name, got.NoSync, c.noSync)
		}
		if got.syncSet != c.syncSet {
			t.Errorf("%s: syncSet = %v, want %v", c.name, got.syncSet, c.syncSet)
		}
	}
}

// TestDetectZeroCapsKeepsTodayBehaviour pins the compatibility contract: a
// caller that builds a Caps literal instead of calling Detect keeps animated
// synchronized truecolour output.
func TestDetectZeroCapsKeepsTodayBehaviour(t *testing.T) {
	var c Caps
	if c.NoSync || c.SyncKnown || c.FPS != 0 || c.Dither != tint.NoDither {
		t.Fatalf("zero Caps is not inert: %+v", c)
	}
	ren := &Renderer{Profile: TrueColor, Sync: !c.NoSync, Dither: c.Dither}
	b := cell.New(4, 1)
	b.WriteString(0, 0, "test", tint.RGB(255, 0, 0))
	first := ren.Frame(b)
	if first == nil {
		t.Fatal("first frame wrote nothing")
	}
	if string(first[:len(syncStart)]) != syncStart {
		t.Fatal("a zero Caps lost synchronized output")
	}
}

func TestDitherFor(t *testing.T) {
	for _, c := range []struct {
		p    Profile
		in   tint.Dither
		want tint.Dither
	}{
		{TrueColor, tint.Bayer8, tint.NoDither},
		{NoColor, tint.Bayer8, tint.NoDither},
		{ANSI256, tint.Bayer8, tint.Bayer8},
		{ANSI16, tint.Bayer8, tint.Bayer8},
		{ANSI256, tint.NoDither, tint.NoDither},
	} {
		if got := DitherFor(c.p, c.in); got != c.want {
			t.Errorf("DitherFor(%v, %v) = %v, want %v", c.p, c.in, got, c.want)
		}
	}
}

// flatBuffer is a block of one colour that sits between two palette entries,
// so a dithered encode has to choose both of them.
func flatBuffer() *cell.Buffer {
	b := cell.New(8, 8)
	c := tint.MustHex("#3a7bd5")
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			b.Set(x, y, cell.Cell{Rune: '\u2588', FG: c})
		}
	}
	return b
}

// paletteIndices returns the distinct xterm-256 indices in an encoded frame.
func paletteIndices(s string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ";38;5;")[1:] {
		i := 0
		for i < len(part) && part[i] >= '0' && part[i] <= '9' {
			i++
		}
		out[part[:i]] = true
	}
	return out
}

// TestRendererDitherIsDeterministicAndPositional checks the property the diff
// encoder depends on: the same buffer always encodes identically, and a flat
// colour is stippled across more than one palette entry only when dithering
// is on.
func TestRendererDitherIsDeterministicAndPositional(t *testing.T) {
	b := flatBuffer()
	first := string((&Renderer{Profile: ANSI256, Dither: tint.Bayer8}).Frame(b))
	again := string((&Renderer{Profile: ANSI256, Dither: tint.Bayer8}).Frame(b))
	if first != again {
		t.Fatal("dithered frames are not reproducible")
	}
	plain := string((&Renderer{Profile: ANSI256}).Frame(b))
	if n := len(paletteIndices(plain)); n != 1 {
		t.Fatalf("undithered flat colour used %d palette entries, want 1", n)
	}
	dithered := paletteIndices(first)
	if len(dithered) < 2 {
		t.Fatalf("dithered flat colour used only %d palette entries; the stipple is missing", len(dithered))
	}
}
