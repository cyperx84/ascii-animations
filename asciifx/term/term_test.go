package term

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name    string
		tty     bool
		env     func(string) string
		profile Profile
		animate bool
	}{
		{"truecolor tty", true, env("COLORTERM", "truecolor", "TERM", "xterm-256color"), TrueColor, true},
		{"256", true, env("TERM", "xterm-256color"), ANSI256, true},
		{"no color keeps motion", true, env("NO_COLOR", "1", "COLORTERM", "truecolor"), NoColor, true},
		{"pipe is static", false, env("COLORTERM", "truecolor"), TrueColor, false},
		{"ci is static", true, env("CI", "true", "TERM", "xterm"), ANSI16, false},
		{"dumb", true, env("TERM", "dumb"), NoColor, false},
		{"reduced motion", true, env("ASCIIFX_REDUCED_MOTION", "1"), ANSI16, false},
		{"forced", false, env("ASCIIFX_FORCE_ANIMATION", "1", "ASCIIFX_COLOR", "256"), ANSI256, true},
	}
	for _, c := range cases {
		got := detect(c.tty, c.env)
		if got.Profile != c.profile || got.Animate != c.animate {
			t.Errorf("%s: got profile=%v animate=%v (%s)", c.name, got.Profile, got.Animate, got.Reason)
		}
		if !got.Animate && got.Reason == "" {
			t.Errorf("%s: static without a reason", c.name)
		}
	}
}

func TestRendererDiffs(t *testing.T) {
	b := cell.New(6, 2)
	b.WriteString(0, 0, "hello", tint.RGB(255, 0, 0))
	r := &Renderer{Profile: TrueColor, Sync: true}
	first := string(r.Frame(b))
	if !strings.HasPrefix(first, syncStart) || !strings.HasSuffix(first, syncEnd) {
		t.Fatal("frame not wrapped in synchronized output")
	}
	if !strings.Contains(first, "38;2;255;0;0") {
		t.Fatal("truecolour SGR missing")
	}
	if out := r.Frame(b); out != nil {
		t.Fatalf("unchanged frame wrote %d bytes", len(out))
	}
	b.SetRune(4, 0, '!', tint.RGB(255, 0, 0))
	second := string(r.Frame(b))
	if strings.Contains(second, "hell") || !strings.Contains(second, "!") {
		t.Fatalf("diff should only rewrite the changed cell: %q", second)
	}
}

func TestANSIProfiles(t *testing.T) {
	b := cell.New(2, 1)
	b.Set(0, 0, cell.Cell{Rune: 'x', FG: tint.RGB(250, 0, 0), BG: tint.RGB(0, 0, 200)})
	if s := ANSI(b, ANSI16); !strings.Contains(s, ";91") || !strings.Contains(s, ";44") {
		t.Errorf("16-colour: %q", s)
	}
	if s := ANSI(b, NoColor); s != "\x1b[0mx\x1b[0m " && strings.Contains(s, "38;") {
		t.Errorf("no-colour output contains colour: %q", s)
	}
	if s := ANSI(b, ANSI256); !strings.Contains(s, "38;5;") {
		t.Errorf("256: %q", s)
	}
}

func TestLumaShowsColourOnlyShapes(t *testing.T) {
	b := cell.New(3, 1)
	b.Set(0, 0, cell.Cell{Rune: '█', FG: tint.RGB(255, 255, 255)})
	b.Set(1, 0, cell.Cell{Rune: '▀', FG: tint.RGB(40, 40, 40), BG: tint.RGB(40, 40, 40)})
	got := Luma(b)
	if got[0] != '@' || got[2] != ' ' || got[1] == '@' {
		t.Fatalf("luma %q", got)
	}
}
