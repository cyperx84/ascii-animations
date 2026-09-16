package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func cmdPlay(e *env, args []string) error {
	c := findCommand("play")
	fs := newFlags(c.name)
	rf := addRunFlags(fs)
	inline := fs.Bool("inline", false, "draw below the cursor instead of the alternate screen")
	loop := fs.Bool("loop", false, "restart finite effects")
	limit := fs.Duration("limit", 0, "stop after this long, e.g. 5s")
	hold := fs.Duration("hold", time.Second, "keep a finished fullscreen effect on screen")
	probe := fs.Bool("probe", false, "ask the terminal whether it supports synchronized output (mode 2026) before playing")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	name, err := oneEffect(c, pos)
	if err != nil {
		return err
	}
	if *limit < 0 || *hold < 0 {
		return usageErr("durations look like 500ms, 2s or 1m", "--limit and --hold must not be negative")
	}
	caps := term.Detect(os.Stdout)
	if rf.profile != "" || rf.dither != "" {
		profile, dither, err := rf.colorPrefs(os.Stdout)
		if err != nil {
			return err
		}
		// SetDither, not a plain assignment: it also marks the dither as the
		// caller's, so Play does not re-resolve it from the environment.
		caps.Profile = profile
		caps.SetDither(dither)
	}
	b, err := rf.build(e, name)
	if err != nil {
		return err
	}
	err = term.Play(context.Background(), b.run, term.PlayOptions{
		Out:    os.Stdout,
		Caps:   caps,
		Inline: *inline,
		Fit:    !*inline && rf.w == 0 && rf.h == 0,
		Limit:  *limit,
		Loop:   *loop,
		Hold:   *hold,
		Probe:  *probe,
	})
	if errors.Is(err, term.ErrInterrupted) || errors.Is(err, context.Canceled) {
		return nil
	}
	if err != nil {
		return runtimeErr(err, "")
	}
	return nil
}
