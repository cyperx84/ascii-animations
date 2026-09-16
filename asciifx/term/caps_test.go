package term

import (
	"context"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
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
		// An ASCIIFX_FPS that is not a usable answer must leave the transport
		// cap standing, never uncap playback.
		{"fps above the ceiling", env("ASCIIFX_FPS", "999", "TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
		{"fps of zero", env("ASCIIFX_FPS", "0", "TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
		{"negative fps", env("ASCIIFX_FPS", "-1", "TERM", "xterm"), 30, tint.Bayer8, false, false},
		{"fps that is not a number", env("ASCIIFX_FPS", "abc", "TERM", "xterm"), 30, tint.Bayer8, false, false},
		{"fps with trailing junk", env("ASCIIFX_FPS", "60fps", "TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
		{"fps in scientific notation", env("ASCIIFX_FPS", "1e3", "TERM", "xterm"), 30, tint.Bayer8, false, false},
		{"fps with whitespace", env("ASCIIFX_FPS", " 60", "TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
		{"empty fps", env("ASCIIFX_FPS", "", "TERM", "xterm"), 30, tint.Bayer8, false, false},
		{"the ceiling itself is allowed", env("ASCIIFX_FPS", "240", "TERM", "tmux-256color"), 240, tint.Bayer8, false, false},
		{"one past the ceiling is not", env("ASCIIFX_FPS", "241", "TERM", "tmux-256color"), 15, tint.Bayer8, false, false},
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

// TestZeroCapsIsConservative pins the zero value field by field, and then
// pins what Play actually does with it. The comment on Caps used to claim a
// zero literal behaved like an animated truecolour terminal; it does not, and
// the test that was here hid that by building its Renderer with a hardcoded
// TrueColor instead of reading c.Profile.
func TestZeroCapsIsConservative(t *testing.T) {
	var c Caps
	for _, f := range []struct {
		name      string
		got, want any
	}{
		{"Profile", c.Profile, NoColor},
		{"TTY", c.TTY, false},
		{"Animate", c.Animate, false},
		{"Reason", c.Reason, ""},
		{"NoSync", c.NoSync, false},
		{"SyncKnown", c.SyncKnown, false},
		{"FPS", c.FPS, 0},
		{"Dither", c.Dither, tint.NoDither},
		{"DitherPref", c.DitherPref, tint.NoDither},
	} {
		if f.got != f.want {
			t.Errorf("zero Caps: %s = %v, want %v", f.name, f.got, f.want)
		}
	}

	// Play is handed the zero Caps untouched: Animate's zero is what sends it
	// down the static branch, and Profile's zero is why that frame carries no
	// colour at all.
	out := playStatic(t, c)
	if strings.Contains(out, "\x1b") {
		t.Fatalf("a zero Caps produced escape sequences: %q", out)
	}
	if !strings.Contains(out, "\u2588") {
		t.Fatalf("a zero Caps produced no frame: %q", out)
	}
}

// TestZeroValuedFieldsImposeNothing is the half of the old claim that is true:
// once a caller has set the two fields whose zero is a restriction, the rest
// impose nothing until they are set.
func TestZeroValuedFieldsImposeNothing(t *testing.T) {
	c := Caps{Profile: TrueColor, Animate: true}

	// NoSync's zero leaves synchronized output on.
	ren := &Renderer{Profile: c.Profile, Sync: !c.NoSync, Dither: c.dither()}
	b := cell.New(4, 1)
	b.WriteString(0, 0, "test", tint.RGB(255, 0, 0))
	first := ren.Frame(b)
	if first == nil {
		t.Fatal("first frame wrote nothing")
	}
	if string(first[:len(syncStart)]) != syncStart {
		t.Fatal("a zero NoSync lost synchronized output")
	}
	// FPS's zero caps nothing, and Dither's zero adds no stipple.
	if got := playFPS(60, c.FPS); got != 60 {
		t.Errorf("a zero FPS capped a 60 fps run to %d", got)
	}
	if got := c.dither(); got != tint.NoDither {
		t.Errorf("a zero Dither resolved to %v", got)
	}
}

// TestNoColorAnyNonEmptyValueDisablesColour pins the NO_COLOR contract, which
// has one reader and one meaning. no-color.org asks for colour to be dropped
// when the variable is present and not an empty string, regardless of its
// value, so NO_COLOR=0 disables colour exactly as NO_COLOR=1 does and
// NO_COLOR= counts as unset. It decides the profile and nothing else, and
// DitherPref survives it so that an explicit profile afterwards gets the
// dither that profile implies.
func TestNoColorAnyNonEmptyValueDisablesColour(t *testing.T) {
	for _, v := range []string{"1", "0", "false", "no", " ", "anything"} {
		c := detect(true, env("NO_COLOR", v, "COLORTERM", "truecolor", "TERM", "xterm-256color"))
		if c.Profile != NoColor {
			t.Errorf("NO_COLOR=%q: Profile = %v, want none", v, c.Profile)
		}
		if c.Dither != tint.NoDither {
			t.Errorf("NO_COLOR=%q: Dither = %v, want none", v, c.Dither)
		}
		if c.DitherPref != tint.Bayer8 {
			t.Errorf("NO_COLOR=%q: DitherPref = %v, want bayer8 to survive for a later override", v, c.DitherPref)
		}
		c.Profile = ANSI256
		if got := c.dither(); got != tint.Bayer8 {
			t.Errorf("NO_COLOR=%q then Profile = ANSI256: dither = %v, want bayer8", v, got)
		}
	}
	// An empty value is not a set one, and neither is an absent variable.
	for _, c := range []Caps{
		detect(true, env("NO_COLOR", "", "TERM", "xterm-256color")),
		detect(true, env("TERM", "xterm-256color")),
	} {
		if c.Profile != ANSI256 {
			t.Errorf("without NO_COLOR: Profile = %v, want 256", c.Profile)
		}
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

// flat paints every cell the one colour flatBuffer uses, so a dithered encode
// has to stipple across two palette entries and an undithered one cannot.
type flat struct{}

func (flat) Duration() float64 { return 0.1 }
func (flat) Step(f *fx.Frame) {
	c := tint.MustHex("#3a7bd5")
	for y := 0; y < f.Buf.H; y++ {
		for x := 0; x < f.Buf.W; x++ {
			f.Buf.Set(x, y, cell.Cell{Rune: '\u2588', FG: c})
		}
	}
}

func flatRun(t *testing.T) *fx.Run {
	t.Helper()
	spec := &fx.Spec{
		Name: "flat", Kind: fx.Transition, FPS: 10, DefW: 8, DefH: 8,
		New: func(fx.Values, int, int, *rand.Rand) (fx.Effect, error) { return flat{}, nil },
	}
	r, err := fx.NewRun(spec, fx.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// playStatic runs the non-animated path, which encodes one frame with exactly
// the profile and dither Play resolved.
//
// It refuses rather than clears an Animate it was not given: forcing the field
// here would mean no caller could ever show that Animate is what sends Play
// down this branch in the first place.
func playStatic(t *testing.T, caps Caps) string {
	t.Helper()
	if caps.Animate {
		t.Fatal("playStatic reads Play's static branch; pass a Caps with Animate false")
	}
	f, err := os.CreateTemp(t.TempDir(), "static")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := Play(context.Background(), flatRun(t), PlayOptions{Out: f, Caps: caps}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f.Name())
	return string(b)
}

// TestDetectKeepsDitherPreferenceUnderNoColor pins the NO_COLOR half of the
// stale-dither bug: NO_COLOR still means no colour, but it must not also erase
// the dither the environment asked for, or an explicit profile has nothing to
// fall back on.
func TestDetectKeepsDitherPreferenceUnderNoColor(t *testing.T) {
	c := detect(true, env("NO_COLOR", "1", "TERM", "xterm-256color"))
	if c.Profile != NoColor || c.Dither != tint.NoDither {
		t.Fatalf("NO_COLOR must still disable colour and dither: Profile=%v Dither=%v", c.Profile, c.Dither)
	}
	if c.DitherPref != tint.Bayer8 {
		t.Fatalf("DitherPref = %v, want bayer8: the preference outlives the profile", c.DitherPref)
	}
	c.Profile = ANSI256
	if got := c.dither(); got != tint.Bayer8 {
		t.Fatalf("after overriding the profile under NO_COLOR, dither = %v, want bayer8", got)
	}
	// An explicit ASCIIFX_DITHER survives NO_COLOR too.
	d := detect(true, env("NO_COLOR", "1", "ASCIIFX_DITHER", "bayer4", "TERM", "xterm-256color"))
	d.Profile = ANSI256
	if got := d.dither(); got != tint.Bayer4 {
		t.Fatalf("ASCIIFX_DITHER=bayer4 under NO_COLOR: dither = %v, want bayer4", got)
	}
}

// TestProfileOverrideReresolvesDetectedDither is the bug 4d20d7d fixed in the
// CLI, pinned at the layer every library caller goes through: Detect resolves
// Dither for the profile it saw, so replacing Profile has to re-resolve it.
func TestProfileOverrideReresolvesDetectedDither(t *testing.T) {
	for _, c := range []struct {
		name string
		env  func(string) string
		to   Profile
		want tint.Dither
	}{
		{"truecolor terminal, 256 asked for", env("COLORTERM", "truecolor", "TERM", "xterm-256color"), ANSI256, tint.Bayer8},
		{"truecolor terminal, 16 asked for", env("COLORTERM", "truecolor", "TERM", "xterm-256color"), ANSI16, tint.Bayer8},
		{"256 terminal, truecolor asked for", env("TERM", "xterm-256color"), TrueColor, tint.NoDither},
		{"256 terminal, none asked for", env("TERM", "xterm-256color"), NoColor, tint.NoDither},
	} {
		caps := detect(true, c.env)
		caps.Profile = c.to
		if got := caps.dither(); got != c.want {
			t.Errorf("%s: dither = %v, want %v", c.name, got, c.want)
		}
	}

	// End to end through Play: a flat colour must be stippled once the caller
	// has asked for a palette profile.
	caps := detect(true, env("COLORTERM", "truecolor", "TERM", "xterm-256color"))
	caps.Profile = ANSI256
	caps.Animate = false // one frame is enough to see the stipple
	if n := len(paletteIndices(playStatic(t, caps))); n < 2 {
		t.Fatalf("Play used %d palette entries after Caps.Profile = ANSI256; the dither did not come back", n)
	}
}

// TestHandBuiltCapsDitherIsUsedAsGiven is the other side of the same fix: a
// Caps the caller assembled has no detected preference to re-resolve from, so
// its Dither must be taken literally. Re-resolving unconditionally would read
// the zero DitherPref and silently drop the caller's bayer8.
func TestHandBuiltCapsDitherIsUsedAsGiven(t *testing.T) {
	on := Caps{Profile: ANSI256, Dither: tint.Bayer8}
	if got := on.dither(); got != tint.Bayer8 {
		t.Fatalf("hand-built Caps: dither = %v, want bayer8 (DitherPref is %v and must not be consulted)", got, on.DitherPref)
	}
	if n := len(paletteIndices(playStatic(t, on))); n < 2 {
		t.Fatalf("Play used %d palette entries for a hand-built Caps{Dither: bayer8}, want a stipple", n)
	}
	off := Caps{Profile: ANSI256}
	if got := off.dither(); got != tint.NoDither {
		t.Fatalf("hand-built Caps with no dither: dither = %v, want none", got)
	}
	if n := len(paletteIndices(playStatic(t, off))); n != 1 {
		t.Fatalf("Play used %d palette entries for a hand-built Caps with no dither, want 1", n)
	}
}

// TestSetDitherOutranksReresolution pins the escape hatch: once the caller has
// chosen, a later Profile change must not undo the choice.
func TestSetDitherOutranksReresolution(t *testing.T) {
	caps := detect(true, env("TERM", "xterm-256color"))
	caps.SetDither(tint.NoDither)
	caps.Profile = ANSI256
	if got := caps.dither(); got != tint.NoDither {
		t.Fatalf("dither = %v after SetDither(none), want none", got)
	}
	caps.Animate = false
	if n := len(paletteIndices(playStatic(t, caps))); n != 1 {
		t.Fatalf("Play used %d palette entries after SetDither(none), want 1", n)
	}
	caps = detect(true, env("COLORTERM", "truecolor", "TERM", "xterm-256color"))
	caps.SetDither(tint.Bayer4)
	caps.Profile = ANSI256
	if got := caps.dither(); got != tint.Bayer4 {
		t.Fatalf("dither = %v after SetDither(bayer4), want bayer4", got)
	}
}

// TestDirectDitherAssignmentAfterDetectIsHonoured pins the public API: Dither
// is an exported field on a Caps that Detect filled in, and assigning to it
// has always been how a caller overrides the detected answer. Re-resolution
// must not swallow that.
func TestDirectDitherAssignmentAfterDetectIsHonoured(t *testing.T) {
	// Turning a detected dither off.
	off := detect(true, env("TERM", "xterm-256color"))
	if off.Dither != tint.Bayer8 {
		t.Fatalf("precondition: a 256 terminal detects %v, want bayer8", off.Dither)
	}
	off.Dither = tint.NoDither
	if got := off.dither(); got != tint.NoDither {
		t.Fatalf("Dither = none assigned directly: dither = %v, want none", got)
	}
	off.Animate = false
	if n := len(paletteIndices(playStatic(t, off))); n != 1 {
		t.Fatalf("Play used %d palette entries after Dither = none, want 1", n)
	}

	// Opting in where detection had nothing to dither onto, alongside the
	// profile override that is the whole reason re-resolution exists.
	on := detect(true, env("COLORTERM", "truecolor", "TERM", "xterm-256color"))
	if on.Dither != tint.NoDither {
		t.Fatalf("precondition: a truecolour terminal detects %v, want none", on.Dither)
	}
	on.Profile = ANSI256
	on.Dither = tint.Bayer4
	if got := on.dither(); got != tint.Bayer4 {
		t.Fatalf("Dither = bayer4 assigned directly: dither = %v, want bayer4", got)
	}
	on.Animate = false
	if n := len(paletteIndices(playStatic(t, on))); n < 2 {
		t.Fatalf("Play used %d palette entries after Dither = bayer4, want a stipple", n)
	}

	// Opting in without touching Profile at all: the environment asked for no
	// dither, the caller wants one, and the profile already has a palette.
	back := detect(true, env("TERM", "xterm-256color", "ASCIIFX_DITHER", "none"))
	if back.Profile != ANSI256 || back.Dither != tint.NoDither {
		t.Fatalf("precondition: Profile=%v Dither=%v", back.Profile, back.Dither)
	}
	back.Dither = tint.Bayer8
	if got := back.dither(); got != tint.Bayer8 {
		t.Fatalf("Dither = bayer8 assigned over ASCIIFX_DITHER=none: dither = %v, want bayer8", got)
	}
	back.Animate = false
	if n := len(paletteIndices(playStatic(t, back))); n < 2 {
		t.Fatalf("Play used %d palette entries after Dither = bayer8, want a stipple", n)
	}
}

// TestSameValueDitherAssignmentNeedsSetDither pins the one case the value
// comparison cannot decide, and the escape hatch for it. Assigning the value
// Detect had already chosen leaves no trace, so it is still treated as
// Detect's answer and re-resolved.
func TestSameValueDitherAssignmentNeedsSetDither(t *testing.T) {
	e := env("COLORTERM", "truecolor", "TERM", "xterm-256color")

	ambiguous := detect(true, e)
	ambiguous.Dither = tint.NoDither // exactly what Detect chose
	ambiguous.Profile = ANSI256
	if got := ambiguous.dither(); got != tint.Bayer8 {
		t.Fatalf("assigning the detected value is indistinguishable from leaving it: dither = %v, want the re-resolved bayer8", got)
	}

	explicit := detect(true, e)
	explicit.SetDither(tint.NoDither)
	explicit.Profile = ANSI256
	if got := explicit.dither(); got != tint.NoDither {
		t.Fatalf("SetDither(none): dither = %v, want none", got)
	}
	explicit.Animate = false
	if n := len(paletteIndices(playStatic(t, explicit))); n != 1 {
		t.Fatalf("Play used %d palette entries after SetDither(none), want 1", n)
	}
}

func TestParseFPS(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
		ok   bool
	}{
		{"60", 60, true},
		{"1", 1, true},
		{"240", 240, true},
		{"241", 0, false},
		{"0", 0, false},
		{"-1", 0, false},
		{"", 0, false},
		{"abc", 0, false},
		{"60fps", 0, false},
		{"1e3", 0, false},
		{" 60", 0, false},
		{"60 ", 0, false},
		{"+60", 60, true}, // strconv accepts a sign; it is still one integer
		{"6.0", 0, false},
	} {
		got, ok := parseFPS(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("parseFPS(%q) = %d, %v; want %d, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestDetectFPSIsUncappedOffATerminal pins the one place a zero FPS is right:
// a pipe has no transport to cap, so nothing is imposed unless ASCIIFX_FPS
// asks for it.
func TestDetectFPSIsUncappedOffATerminal(t *testing.T) {
	if got := detect(false, env("TERM", "tmux-256color")).FPS; got != 0 {
		t.Errorf("off a terminal: FPS = %d, want 0", got)
	}
	if got := detect(false, env("TERM", "tmux-256color", "ASCIIFX_FPS", "999")).FPS; got != 0 {
		t.Errorf("off a terminal with a bad ASCIIFX_FPS: FPS = %d, want 0", got)
	}
	if got := detect(false, env("TERM", "tmux-256color", "ASCIIFX_FPS", "48")).FPS; got != 48 {
		t.Errorf("off a terminal with ASCIIFX_FPS=48: FPS = %d, want 48", got)
	}
}
