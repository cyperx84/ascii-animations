package main

import (
	"bufio"
	"encoding/json"
	"strconv"

	"github.com/cyperx84/ascii-animations/asciifx/term"
)

func cmdCast(e *env, args []string) error {
	c := findCommand("cast")
	fs := newFlags(c.name)
	rf := addRunFlags(fs)
	seconds := fs.Float64("seconds", 0, "length to record (default: full duration for finite effects, 3s for loops)")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	name, err := oneEffect(c, pos)
	if err != nil {
		return err
	}
	// 0 means "default length"; anything else must be a sane finite span.
	// Casts stream frame by frame, so this bounds time, not memory.
	if *seconds != 0 {
		if err := checkSeconds(*seconds); err != nil {
			return err
		}
	}
	profile := term.TrueColor
	if rf.profile != "" {
		if profile, err = term.ParseProfile(rf.profile); err != nil {
			return usageErr("valid profiles: truecolor, 256, 16, none", "%v", err)
		}
	}
	b, err := rf.build(e, name)
	if err != nil {
		return err
	}
	return writeCast(e, b, profile, *seconds)
}

// writeCast emits asciicast v3: a header line, then [interval, "o", data]
// events where interval is seconds since the previous event. Output is a
// pure function of the run, so casts are reproducible.
func writeCast(e *env, b *built, profile term.Profile, seconds float64) error {
	r := b.run
	fps := r.FPS()
	w, h := r.Size()
	last := r.Frames() - 1
	if seconds > 0 || last < 0 {
		if seconds == 0 {
			seconds = 3
		}
		last = int(seconds*float64(fps) + 0.5)
		if n := r.Frames(); n > 0 {
			last = min(last, n-1)
		}
	}
	out := bufio.NewWriter(e.stdout)
	header := map[string]any{
		"version": 3,
		"term":    map[string]int{"cols": w, "rows": h},
		"title":   "asciifx " + b.spec.Name,
	}
	hb, _ := json.Marshal(header)
	out.Write(hb)
	out.WriteByte('\n')
	event := func(dt float64, data string) {
		eb, _ := json.Marshal([]any{round6(dt), "o", data})
		out.Write(eb)
		out.WriteByte('\n')
	}
	ren := &term.Renderer{Profile: profile}
	prevTick := 0
	for t := 0; t <= last; t++ {
		buf, err := r.Seek(t)
		if err != nil {
			return runtimeErr(err, "")
		}
		frame := ren.Frame(buf)
		if frame == nil {
			continue
		}
		data := string(frame)
		if t == 0 {
			data = "\x1b[?25l" + data
		}
		event(float64(t-prevTick)/float64(fps), data)
		prevTick = t
	}
	// Park the cursor below the art and show it again, one tick after the
	// last frame.
	event(float64(last+1-prevTick)/float64(fps), "\x1b["+strconv.Itoa(h)+";1H\x1b[?25h")
	return out.Flush()
}
