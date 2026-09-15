package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// issue is one lint finding. Frame is 0-based (like render ticks); line and
// col are 1-based (editor convention), col counting runes.
type issue struct {
	File    string `json:"file"`
	Set     string `json:"set,omitempty"`
	Frame   int    `json:"frame"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Kind    string `json:"kind"`
	Rune    string `json:"rune,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type checkFrame struct {
	set   string
	index int
	lines []string
	// base is the 1-based file line of the frame's first line, 0 when
	// positions are relative to the frame (JSON input).
	base int
}

type checkResult struct {
	OK     bool    `json:"ok"`
	File   string  `json:"file"`
	Frames int     `json:"frames"`
	Issues []issue `json:"issues"`
}

func cmdCheck(e *env, args []string) error {
	c := findCommand("check")
	fs := newFlags(c.name)
	asJSON := fs.Bool("json", false, "emit JSON")
	sep := fs.String("frames-sep", "---", "line separating frames in text input")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageErr("usage: "+c.usage+" (use - for stdin)", "check needs exactly one file")
	}
	path := pos[0]
	var data []byte
	if path == "-" {
		data, err = io.ReadAll(e.stdin)
		path = "<stdin>"
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return runtimeErr(err, "check the path, or pass - to read stdin")
	}
	frames, err := splitFrames(data, *sep)
	if err != nil {
		return err
	}
	res := checkResult{File: path, Frames: len(frames), Issues: lint(path, frames)}
	res.OK = len(res.Issues) == 0
	if *asJSON {
		if err := writeJSON(e.stdout, res); err != nil {
			return err
		}
	} else {
		for _, is := range res.Issues {
			loc := is.File
			if is.Set != "" {
				loc += "#" + is.Set
			}
			fmt.Fprintf(e.stdout, "%s:%d:%d:%d: %s\n", loc, is.Frame, is.Line, is.Col, is.Message)
		}
		if res.OK {
			fmt.Fprintf(e.stderr, "%s: ok (%d frames)\n", path, res.Frames)
		} else {
			fmt.Fprintf(e.stderr, "%s: %d issues in %d frames\n", path, len(res.Issues), res.Frames)
		}
	}
	if !res.OK {
		return &cliError{code: exitIssues, msg: "", hint: ""}
	}
	return nil
}

type spinnerJSON struct {
	Interval int      `json:"interval"`
	Frames   []string `json:"frames"`
}

// splitFrames reads cli-spinners JSON (one spinner or a map of them) or text
// frames separated by sep lines. A line is a separator when it equals sep or
// starts with sep+" ", so `asciifx render --every` output checks directly.
func splitFrames(data []byte, sep string) ([]checkFrame, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '{' {
		var one struct {
			Frames *[]string `json:"frames"`
		}
		if json.Unmarshal(trim, &one) == nil && one.Frames != nil {
			return spinnerFrames("", *one.Frames), nil
		}
		var many map[string]spinnerJSON
		if err := json.Unmarshal(trim, &many); err == nil && len(many) > 0 {
			names := make([]string, 0, len(many))
			for n := range many {
				names = append(names, n)
			}
			sort.Strings(names)
			var out []checkFrame
			for _, n := range names {
				out = append(out, spinnerFrames(n, many[n].Frames)...)
			}
			return out, nil
		}
		if json.Valid(trim) {
			return nil, usageErr(`expected cli-spinners shape {"interval":80,"frames":["..."]} or a map of name to that`, "JSON input has no frames")
		}
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	var frames []checkFrame
	cur := checkFrame{base: 1}
	for i, l := range lines {
		if sep != "" && (l == sep || strings.HasPrefix(l, sep+" ")) {
			if len(cur.lines) > 0 || i > 0 {
				frames = append(frames, cur)
			}
			cur = checkFrame{index: len(frames), base: i + 2}
			continue
		}
		cur.lines = append(cur.lines, l)
	}
	frames = append(frames, cur)
	// Drop empty frames produced by a leading separator or blank spacers
	// between frames (render output puts a blank line before each header).
	var out []checkFrame
	for _, f := range frames {
		for len(f.lines) > 0 && f.lines[len(f.lines)-1] == "" {
			f.lines = f.lines[:len(f.lines)-1]
		}
		if len(f.lines) == 0 {
			continue
		}
		f.index = len(out)
		out = append(out, f)
	}
	return out, nil
}

func spinnerFrames(set string, frames []string) []checkFrame {
	out := make([]checkFrame, len(frames))
	for i, f := range frames {
		out[i] = checkFrame{set: set, index: i, lines: strings.Split(f, "\n")}
	}
	return out
}

// displayWidth sums cell widths, counting unsafe runes at their likely width.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch cw := cell.Width(r); {
		case r == '\t':
			w += 8 - w%8
		case cw < 0:
			w += 2
		default:
			w += cw
		}
	}
	return w
}

func runeReason(r rune) (kind, why string) {
	switch w := cell.Width(r); {
	case r == '\t':
		return "tab", "tab renders as 1-8 columns depending on position; expand to spaces"
	case unicode.IsControl(r) || r == 0x7F:
		return "control", "control character has no width and can move the cursor; remove it"
	case w == 0:
		return "zero-width", "zero-width rune (combining mark or format character) merges with its neighbour; use a precomposed single-width glyph"
	case w == 2:
		return "wide", "wide rune takes 2 columns and shifts everything after it; use a single-width glyph"
	case r == 0xFE0F || r == 0x200D:
		return "variable", "emoji presentation selector / joiner makes width terminal-dependent; remove it"
	case r >= 0x1F000:
		return "emoji", "emoji width varies between terminals (2-6 cells); use single-width glyphs such as box, block or braille"
	case w < 0:
		return "variable", "symbol's width varies between terminals and fonts (emoji presentation); pick a glyph from box, block or braille ranges"
	}
	return "", ""
}

func lint(file string, frames []checkFrame) []issue {
	issues := []issue{}
	add := func(f checkFrame, line, col int, kind, msg string, r rune) {
		is := issue{File: file, Set: f.set, Frame: f.index, Line: line, Col: col, Kind: kind, Message: msg}
		if f.base > 0 {
			is.Line = f.base + line - 1
		}
		if r != 0 {
			is.Rune = string(r)
			is.Code = fmt.Sprintf("U+%04X", r)
		}
		issues = append(issues, is)
	}
	type dims struct{ w, h int }
	setDims := map[string]dims{}
	setFirst := map[string]int{}
	for _, f := range frames {
		widths := make([]int, len(f.lines))
		trailing := 0
		for li, l := range f.lines {
			col := 1
			for _, r := range l {
				if kind, why := runeReason(r); kind != "" {
					add(f, li+1, col, kind, fmt.Sprintf("%q U+%04X: %s", string(r), r, why), r)
				}
				col++
			}
			widths[li] = displayWidth(l)
			if strings.TrimRight(l, " \t") != l {
				trailing++
			}
		}
		// Trailing whitespace is fine when every line is padded, and fine
		// when none are; mixing is how stale cells get left behind.
		// Equal-width lines redraw cleanly whatever their padding, so only
		// frames that are also ragged are flagged.
		frameW := 0
		for _, w := range widths {
			frameW = max(frameW, w)
		}
		ragged := false
		for _, w := range widths {
			ragged = ragged || w != frameW
		}
		if ragged && trailing > 0 && trailing < len(f.lines) {
			flagPadded := trailing <= len(f.lines)-trailing
			for li, l := range f.lines {
				trimmed := strings.TrimRight(l, " \t")
				if (trimmed != l) != flagPadded {
					continue
				}
				if flagPadded {
					add(f, li+1, len([]rune(trimmed))+1, "trailing-space", "trailing whitespace while most lines in this frame have none", 0)
				} else {
					add(f, li+1, len([]rune(l))+1, "trailing-space", "no trailing padding while most lines in this frame are padded", 0)
				}
			}
		}
		// Ragged lines: compare against the frame's widest line.
		for li, w := range widths {
			if w != frameW {
				add(f, li+1, w+1, "ragged", fmt.Sprintf("line is %d columns wide, frame is %d (pad lines to equal width so redraws overwrite every cell)", w, frameW), 0)
			}
		}
		d := dims{w: frameW, h: len(f.lines)}
		if prev, ok := setDims[f.set]; !ok {
			setDims[f.set], setFirst[f.set] = d, f.index
		} else {
			if d.h != prev.h {
				add(f, 1, 1, "frame-height", fmt.Sprintf("frame is %d lines tall; frame %d is %d (frames of different heights leave stale rows)", d.h, setFirst[f.set], prev.h), 0)
			}
			if d.w != prev.w {
				add(f, 1, 1, "frame-width", fmt.Sprintf("frame is %d columns wide; frame %d is %d (frames of different widths leave stale columns)", d.w, setFirst[f.set], prev.w), 0)
			}
		}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Set != b.Set {
			return a.Set < b.Set
		}
		if a.Frame != b.Frame {
			return a.Frame < b.Frame
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	return issues
}
