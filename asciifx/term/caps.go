// Package term turns asciifx cell buffers into terminal output: capability
// detection, colour downsampling, diffed frame encoding and a player that
// always restores the terminal.
package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/tint"
	xterm "golang.org/x/term"
)

// Profile is the colour depth to encode for.
type Profile int

const (
	NoColor Profile = iota
	ANSI16
	ANSI256
	TrueColor
)

func (p Profile) String() string {
	return [...]string{"none", "16", "256", "truecolor"}[p]
}

// ParseProfile accepts none, 16, 256 or truecolor.
func ParseProfile(s string) (Profile, error) {
	switch strings.ToLower(s) {
	case "none", "nocolor", "no-color", "ascii":
		return NoColor, nil
	case "16", "ansi", "ansi16":
		return ANSI16, nil
	case "256", "ansi256":
		return ANSI256, nil
	case "truecolor", "24bit", "rgb":
		return TrueColor, nil
	}
	return NoColor, fmt.Errorf("unknown colour profile %q: use none, 16, 256 or truecolor", s)
}

// DitherFor returns the ordered-dither pattern worth using at a profile.
// Ordered dithering is only meaningful when the output is a palette; a
// truecolour or colourless terminal has nothing to dither onto.
func DitherFor(p Profile, d tint.Dither) tint.Dither {
	if p == ANSI16 || p == ANSI256 {
		return d
	}
	return tint.NoDither
}

// Caps is what the output terminal can do and whether animating is wanted.
// Every field's zero value means "no restriction", so a caller that builds a
// Caps literal gets the same behaviour as an animated truecolour terminal.
type Caps struct {
	Profile Profile
	// TTY reports whether output is an interactive terminal.
	TTY bool
	// Animate is false when motion should be replaced by one static frame.
	Animate bool
	// Reason explains why Animate is false, for logs and --json output.
	Reason string
	// NoSync disables synchronized output (mode 2026). Terminals ignore a
	// mode they do not know, so this is only set when a probe got an explicit
	// negative answer, or by ASCIIFX_SYNC=0.
	NoSync bool
	// SyncKnown is true when a probe got a definite answer about mode 2026,
	// so a caller can tell "not supported" from "never asked".
	SyncKnown bool
	// FPS caps the tick rate. Nothing is capped when it is zero. Slower
	// transports (SSH, tmux) get a lower cap because dropped frames look
	// better than queued ones.
	FPS int
	// Dither stipples gradients when Profile is a palette. Zero is off.
	//
	// Detect resolves it for the profile it detected, so Play re-resolves it
	// from DitherPref whenever Profile has since been changed; otherwise an
	// explicit profile inherits a decision made for one that is no longer in
	// play. Assigning any other value here is your answer and is used as
	// given. The one case that cannot be told apart is assigning exactly the
	// value Detect had already chosen — say so with SetDither.
	Dither tint.Dither

	// DitherPref is the dither the environment asked for, before Profile had
	// its say: ASCIIFX_DITHER, or bayer8. It outlives the profile on purpose,
	// so overriding Profile has something to resolve against. Setting it on a
	// Caps you built yourself opts into the same resolution.
	DitherPref tint.Dither

	// ditherAuto is the value Detect wrote into Dither. Dither is re-resolved
	// only while it still holds that value, so assigning a different one is
	// honoured without needing SetDither. A Caps built by hand leaves both
	// zero; re-resolving its zero DitherPref yields NoDither either way, so
	// the comparison costs it nothing.
	ditherAuto tint.Dither

	// ditherSet settles the one case the comparison cannot: SetDither called
	// with exactly the value Detect had already chosen.
	ditherSet bool

	// syncSet records that ASCIIFX_SYNC was given explicitly, so Probe cannot
	// overwrite the user's answer.
	syncSet bool
}

// SetDither fixes the dither to render with. Assigning Dither directly does
// the same thing, except when the value assigned is the one Detect had already
// chosen; use this to say you mean it.
func (c *Caps) SetDither(d tint.Dither) {
	c.Dither, c.ditherSet = d, true
}

// dither is the dither to render with. Detect's own answer is re-resolved
// against the current Profile, because the caller may have replaced the
// profile that answer was made for. Anything the caller chose is used as-is.
func (c Caps) dither() tint.Dither {
	if c.ditherSet || c.Dither != c.ditherAuto {
		return c.Dither
	}
	return DitherFor(c.Profile, c.DitherPref)
}

// Detect inspects the environment and output file. Every decision has an
// override: ASCIIFX_COLOR sets the profile, ASCIIFX_REDUCED_MOTION=1 forces a
// static frame, ASCIIFX_FORCE_ANIMATION=1 animates even in CI or a pipe,
// ASCIIFX_SYNC=0 disables mode 2026, ASCIIFX_DITHER=none|bayer4|bayer8 and
// ASCIIFX_FPS=N tune rendering.
//
// Detect never touches the terminal: it reads environment variables only, so
// it is safe to call before deciding whether to animate. Use Probe to ask the
// terminal itself for the things the environment cannot answer.
func Detect(out *os.File) Caps {
	return detect(out != nil && xterm.IsTerminal(int(out.Fd())), os.Getenv)
}

func detect(tty bool, env func(string) string) Caps {
	c := Caps{TTY: tty, Profile: detectProfile(env), Animate: true}
	set := func(k string) bool { v := env(k); return v != "" && v != "0" && v != "false" }
	switch {
	case set("ASCIIFX_FORCE_ANIMATION"):
	case set("ASCIIFX_REDUCED_MOTION") || set("REDUCED_MOTION"):
		c.Animate, c.Reason = false, "reduced motion requested"
	case !tty:
		c.Animate, c.Reason = false, "output is not a terminal"
	case env("TERM") == "dumb":
		c.Animate, c.Reason = false, "TERM=dumb"
	case set("CI"):
		c.Animate, c.Reason = false, "CI environment"
	}
	if v := env("ASCIIFX_SYNC"); v != "" {
		c.syncSet = true
		c.NoSync = v == "0" || v == "false"
	}
	if v := env("ASCIIFX_FPS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 && n <= 240 {
			c.FPS = n
		}
	} else if tty {
		c.FPS = transportFPS(env)
	}
	d := tint.Bayer8
	if v := env("ASCIIFX_DITHER"); v != "" {
		if parsed, err := tint.ParseDither(v); err == nil {
			d = parsed
		}
	}
	// NO_COLOR deliberately does not clear d: it already forced Profile to
	// NoColor, which is what DitherFor reads, and zeroing the preference too
	// would leave an explicit `--profile 256` with no dither to fall back on.
	c.DitherPref = d
	c.ditherAuto = DitherFor(c.Profile, d)
	c.Dither = c.ditherAuto
	return c
}

// transportFPS is the tick-rate cap for the link the terminal sits behind.
// A multiplexer or a remote shell cannot keep up with 30 fps of smooth
// updates, and tmux only passes mode 2026 through from 3.7.
func transportFPS(env func(string) string) int {
	if env("SSH_CONNECTION") != "" || env("SSH_TTY") != "" {
		return 15
	}
	t := strings.ToLower(env("TERM"))
	if strings.HasPrefix(t, "tmux") || strings.HasPrefix(t, "screen") {
		return 15
	}
	return 30
}

func detectProfile(env func(string) string) Profile {
	if v := env("ASCIIFX_COLOR"); v != "" {
		if p, err := ParseProfile(v); err == nil {
			return p
		}
	}
	if env("NO_COLOR") != "" || env("TERM") == "dumb" {
		return NoColor
	}
	switch strings.ToLower(env("COLORTERM")) {
	case "truecolor", "24bit":
		return TrueColor
	}
	switch env("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "ghostty", "vscode", "Hyper", "rio":
		return TrueColor
	}
	t := env("TERM")
	switch {
	case strings.Contains(t, "kitty"), strings.Contains(t, "ghostty"), strings.Contains(t, "direct"):
		return TrueColor
	case strings.Contains(t, "256color"):
		return ANSI256
	case env("WT_SESSION") != "":
		return TrueColor
	}
	return ANSI16
}
