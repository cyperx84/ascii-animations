package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// frameJSON is the machine-readable form of one rendered frame.
type frameJSON struct {
	Effect      string            `json:"effect"`
	Tick        int               `json:"tick"`
	Time        float64           `json:"time"`
	W           int               `json:"w"`
	H           int               `json:"h"`
	FPS         int               `json:"fps"`
	Seed        uint64            `json:"seed"`
	Params      map[string]string `json:"params"`
	FramesTotal int               `json:"frames_total"`
	Lines       []string          `json:"lines"`
	Luma        []string          `json:"luma"`
	Styles      [][]styleRun      `json:"styles"`
}

// styleRun is a horizontal run of cells sharing colours and attributes.
// Runs with no colour and no attributes are omitted.
type styleRun struct {
	X     int      `json:"x"`
	Len   int      `json:"len"`
	FG    string   `json:"fg"`
	BG    string   `json:"bg"`
	Attrs []string `json:"attrs,omitempty"`
}

func cmdRender(e *env, args []string) error {
	c := findCommand("render")
	fs := newFlags(c.name)
	rf := addRunFlags(fs)
	frameS := fs.String("frame", "", "frame index; negative counts from the end (-1 = last)")
	framesS := fs.String("frames", "", "comma-separated frame indices, e.g. 0,10,-1")
	every := fs.Int("every", 0, "every Kth frame (finite: through the last frame; loops: over --seconds)")
	atS := fs.String("at", "", "time in seconds; negative counts from the end")
	seconds := fs.Float64("seconds", 3, "span covered by --every for looping effects")
	format := fs.String("format", "plain", "plain, luma, ansi or json")
	asJSON := fs.Bool("json", false, "same as --format json")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	if *asJSON {
		*format = "json"
	}
	name, err := oneEffect(c, pos)
	if err != nil {
		return err
	}
	switch *format {
	case "plain", "luma", "ansi", "json":
	default:
		return usageErr("formats: plain, luma, ansi, json", "unknown format %q", *format)
	}
	selectors := 0
	for _, set := range []bool{*frameS != "", *framesS != "", *every != 0, *atS != ""} {
		if set {
			selectors++
		}
	}
	if selectors > 1 {
		return usageErr("pick one of --frame, --frames, --every, --at", "frame selectors are mutually exclusive")
	}
	profile := term.TrueColor
	if *format == "ansi" {
		if profile, err = rf.colorProfile(os.Stdout); err != nil {
			return err
		}
	}
	b, err := rf.build(e, name)
	if err != nil {
		return err
	}
	r := b.run
	total := r.Frames()
	fps := r.FPS()
	rangeHint := func() string {
		if total == 0 {
			return fmt.Sprintf("%s loops forever: use a tick >= 0 (at %d fps, tick %d is 1s)", name, fps, fps)
		}
		return fmt.Sprintf("%s has %d frames (0..%d, or -1 for the last) at %d fps", name, total, total-1, fps)
	}
	resolve := func(n int) (int, error) {
		if n < 0 {
			if total == 0 {
				return 0, usageErr(rangeHint(), "negative frame %d needs a finite effect", n)
			}
			n += total
		}
		if n < 0 || (total > 0 && n >= total) {
			return 0, usageErr(rangeHint(), "frame %d out of range", n)
		}
		if total == 0 && n > maxTick(fps) {
			return 0, usageErr(fmt.Sprintf("looping effects render at most %gs (tick %d at %d fps); frames are deterministic, so a later tick shows nothing new", maxSeconds, maxTick(fps), fps), "frame %d too far", n)
		}
		return n, nil
	}

	var ticks []int
	switch {
	case *frameS != "":
		n, err := strconv.Atoi(*frameS)
		if err != nil {
			return usageErr("--frame takes an integer such as 0, 12 or -1", "bad --frame %q", *frameS)
		}
		t, err := resolve(n)
		if err != nil {
			return err
		}
		ticks = []int{t}
	case *framesS != "":
		list, err := parseIntList(*framesS)
		if err != nil {
			return usageErr("--frames takes integers like 0,10,20,-1", "bad --frames: %v", err)
		}
		if len(list) > maxFrames {
			return usageErr("split the request into smaller batches", "%d frames requested; limit is %d", len(list), maxFrames)
		}
		for _, n := range list {
			t, err := resolve(n)
			if err != nil {
				return err
			}
			ticks = append(ticks, t)
		}
	case *every != 0:
		if *every < 0 {
			return usageErr("--every takes a positive step", "bad --every %d", *every)
		}
		last := total - 1
		if total == 0 {
			if err := checkSeconds(*seconds); err != nil {
				return err
			}
			last = int(*seconds * float64(fps))
		}
		// Size the list arithmetically and check the cap before allocating;
		// count*every never exceeds last, so the loop cannot overflow.
		count := last / *every + 1
		tail := total > 0 && last%*every != 0
		if tail {
			count++
		}
		if count > maxFrames {
			return usageErr("raise --every or lower --seconds", "%d frames requested; limit is %d", count, maxFrames)
		}
		ticks = make([]int, 0, count)
		for i := 0; i <= last / *every; i++ {
			ticks = append(ticks, i**every)
		}
		if tail {
			ticks = append(ticks, last)
		}
	case *atS != "":
		at, err := strconv.ParseFloat(*atS, 64)
		if err != nil || math.IsNaN(at) || math.IsInf(at, 0) {
			return usageErr("--at takes finite seconds, e.g. 0.5 or -0.1", "bad --at %q", *atS)
		}
		if math.Abs(at) > maxSeconds {
			return usageErr(fmt.Sprintf("--at must be within ±%gs", maxSeconds), "--at %g too far", at)
		}
		if at < 0 {
			if total == 0 {
				return usageErr(rangeHint(), "negative --at needs a finite effect")
			}
			at += r.Duration()
		}
		t, err := resolve(int(at*float64(fps) + 0.5))
		if err != nil {
			return err
		}
		ticks = []int{t}
	default:
		ticks = []int{term.StaticTick(r)}
	}

	w := e.stdout
	var jsonFrames []frameJSON
	for i, t := range ticks {
		buf, err := r.Seek(t)
		if err != nil {
			return runtimeErr(err, "")
		}
		secs := round6(float64(t) / float64(fps))
		if *format == "json" {
			jsonFrames = append(jsonFrames, frameToJSON(b, buf, t, secs))
			continue
		}
		if len(ticks) > 1 {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "--- frame %d (t=%.2fs)\n", t, secs)
		}
		switch *format {
		case "plain":
			fmt.Fprint(w, buf.Plain())
		case "luma":
			fmt.Fprint(w, term.Luma(buf))
		case "ansi":
			fmt.Fprint(w, term.ANSI(buf, profile))
		}
	}
	if *format == "json" {
		if len(jsonFrames) == 1 {
			return writeJSON(w, jsonFrames[0])
		}
		return writeJSON(w, jsonFrames)
	}
	fmt.Fprintln(w)
	return nil
}

const (
	// maxFrames bounds how many frames one render call may emit.
	maxFrames = 5000
	// maxSeconds bounds how far into a looping effect render and cast go;
	// seeking replays every tick, so this also bounds CPU time.
	maxSeconds = 600.0
)

func maxTick(fps int) int { return int(maxSeconds * float64(fps)) }

// checkSeconds rejects NaN, infinite, non-positive and over-long spans.
func checkSeconds(s float64) error {
	if math.IsNaN(s) || math.IsInf(s, 0) || s <= 0 || s > maxSeconds {
		return usageErr(fmt.Sprintf("--seconds takes a finite number in (0, %g]", maxSeconds), "bad --seconds %g", s)
	}
	return nil
}

func round6(v float64) float64 { return math.Round(v*1e6) / 1e6 }

func frameToJSON(b *built, buf *cell.Buffer, tick int, secs float64) frameJSON {
	return frameJSON{
		Effect:      b.spec.Name,
		Tick:        tick,
		Time:        secs,
		W:           buf.W,
		H:           buf.H,
		FPS:         b.run.FPS(),
		Seed:        b.seed,
		Params:      b.run.Values.Map(),
		FramesTotal: b.run.Frames(),
		Lines:       buf.Lines(),
		Luma:        strings.Split(term.Luma(buf), "\n"),
		Styles:      styleRuns(buf),
	}
}

func styleRuns(buf *cell.Buffer) [][]styleRun {
	rows := make([][]styleRun, buf.H)
	type key struct {
		fg, bg tint.Color
		attr   cell.Attr
	}
	for y := 0; y < buf.H; y++ {
		runs := []styleRun{}
		x0 := 0
		var cur key
		flush := func(x int) {
			if x > x0 && cur != (key{}) {
				runs = append(runs, styleRun{X: x0, Len: x - x0, FG: cur.fg.String(), BG: cur.bg.String(), Attrs: attrNames(cur.attr)})
			}
		}
		for x := 0; x < buf.W; x++ {
			c := buf.Cells[y*buf.W+x]
			k := key{c.FG, c.BG, c.Attr}
			if x == 0 {
				cur = k
				continue
			}
			if k != cur {
				flush(x)
				x0, cur = x, k
			}
		}
		flush(buf.W)
		rows[y] = runs
	}
	return rows
}

func attrNames(a cell.Attr) []string {
	var out []string
	for _, n := range []struct {
		a    cell.Attr
		name string
	}{{cell.Bold, "bold"}, {cell.Dim, "dim"}, {cell.Italic, "italic"}, {cell.Underline, "underline"}, {cell.Reverse, "reverse"}} {
		if a&n.a != 0 {
			out = append(out, n.name)
		}
	}
	return out
}
