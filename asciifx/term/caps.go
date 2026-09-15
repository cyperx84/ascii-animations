// Package term turns asciifx cell buffers into terminal output: capability
// detection, colour downsampling, diffed frame encoding and a player that
// always restores the terminal.
package term

import (
	"fmt"
	"os"
	"strings"

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

// Caps is what the output terminal can do and whether animating is wanted.
type Caps struct {
	Profile Profile
	// TTY reports whether output is an interactive terminal.
	TTY bool
	// Animate is false when motion should be replaced by one static frame.
	Animate bool
	// Reason explains why Animate is false, for logs and --json output.
	Reason string
}

// Detect inspects the environment and output file. Every decision has an
// override: ASCIIFX_COLOR sets the profile, ASCIIFX_REDUCED_MOTION=1 forces a
// static frame, and ASCIIFX_FORCE_ANIMATION=1 animates even in CI or a pipe.
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
	return c
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
