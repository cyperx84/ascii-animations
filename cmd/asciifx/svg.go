package main

import (
	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/svg"
)

func cmdSVG(e *env, args []string) error {
	c := findCommand("svg")
	fs := newFlags(c.name)
	rf := addRunFlags(fs)
	seconds := fs.Float64("seconds", 0, "length to export (default: the whole effect if finite, 3s if it loops)")
	fontSize := fs.Float64("font-size", 16, "font size in pixels")
	padding := fs.Float64("padding", 12, "padding in pixels")
	background := fs.String("background", "#0d1117", `background colour, or "none" for transparent`)
	foreground := fs.String("foreground", "#c9d1d9", "colour of cells the effect left uncoloured")
	quantize := fs.Int("quantize", 32, "colour levels per channel; fewer means a smaller file (0 keeps every colour)")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	name, err := oneEffect(c, pos)
	if err != nil {
		return err
	}
	if *seconds != 0 {
		if err := checkSeconds(*seconds); err != nil {
			return err
		}
	}
	b, err := rf.build(e, name)
	if err != nil {
		return err
	}
	r := b.run
	last := r.Frames() - 1
	if *seconds > 0 || last < 0 {
		s := *seconds
		if s == 0 {
			s = 3
		}
		last = int(s*float64(r.FPS()) + 0.5)
		if n := r.Frames(); n > 0 {
			last = min(last, n-1)
		}
	}
	// Frames are collected rather than streamed because the document's
	// keyframes need the count before the first frame is written. Seek reuses
	// one buffer, so each frame is cloned out of it.
	frames := make([]*cell.Buffer, 0, last+1)
	for t := 0; t <= last; t++ {
		buf, err := r.Seek(t)
		if err != nil {
			return runtimeErr(err, "")
		}
		frames = append(frames, buf.Clone())
	}
	err = svg.Encode(e.stdout, frames, svg.Options{
		FontSize:   *fontSize,
		Padding:    *padding,
		Background: *background,
		Foreground: *foreground,
		FPS:        r.FPS(),
		Quantize:   *quantize,
		Title:      "asciifx " + b.spec.Name,
	})
	if err != nil {
		return runtimeErr(err, "")
	}
	return nil
}
