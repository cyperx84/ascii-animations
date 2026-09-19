// Package lint checks ASCII art frames for the things that make terminal
// animation break: runes whose width the terminal decides, lines of unequal
// width, and frames that change size between ticks.
//
// It is the engine behind `asciifx check`, exported so the check can live in
// your own test suite rather than in a build step:
//
//	func TestBannerStaysSingleWidth(t *testing.T) {
//		art, err := fx.Banner("ACME", "block")
//		if err != nil {
//			t.Fatal(err)
//		}
//		for _, is := range lint.String(art) {
//			t.Errorf("%d:%d: %s", is.Line, is.Col, is.Message)
//		}
//	}
//
// Every rule exists because of a way a terminal draws something differently
// from the buffer that describes it, so a clean run means the art redraws the
// same in every terminal, not that it looks good.
package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

// Issue is one finding. Frame is 0-based, like a render tick; Line and Col
// are 1-based, the editor convention, and Col counts runes.
type Issue struct {
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

// Frame is one frame of art to check.
type Frame struct {
	// Set names the frame's group, for input that holds several animations.
	Set string
	// Index is the frame's position in its set.
	Index int
	// Lines is the frame, one string per row, tabs and all.
	Lines []string
	// Base is the 1-based file line of the frame's first line, so an issue
	// can point at the file. Zero means positions are relative to the frame.
	Base int
}

// Result is a whole check, shaped for `asciifx check --json`.
type Result struct {
	OK     bool    `json:"ok"`
	File   string  `json:"file"`
	Frames int     `json:"frames"`
	Issues []Issue `json:"issues"`
}

// ErrNoFrames is returned by Split for JSON input that parses but holds no
// frames, which is a different mistake from malformed input and usually means
// the wrong file was passed.
var ErrNoFrames = errors.New(`expected cli-spinners shape {"interval":80,"frames":["..."]} or a map of name to that`)

// String checks one frame of text, splitting it on newlines. It is the short
// form for the common case: art held in a Go string.
func String(s string) []Issue {
	return Check("", []Frame{{Lines: strings.Split(strings.TrimSuffix(s, "\n"), "\n")}})
}

// Strings checks a set of frames, each one a multi-line string — the shape a
// spinner or a hand-written animation has in Go.
func Strings(set string, frames []string) []Issue {
	return Check("", FromStrings(set, frames))
}

// FromStrings turns multi-line strings into frames.
func FromStrings(set string, frames []string) []Frame {
	out := make([]Frame, len(frames))
	for i, f := range frames {
		out[i] = Frame{Set: set, Index: i, Lines: strings.Split(f, "\n")}
	}
	return out
}

type spinnerJSON struct {
	Interval int      `json:"interval"`
	Frames   []string `json:"frames"`
}

// Split reads cli-spinners JSON (one spinner or a map of them) or text frames
// separated by sep lines. A line is a separator when it equals sep or starts
// with sep+" ", so `asciifx render --every` output checks directly.
func Split(data []byte, sep string) ([]Frame, error) {
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '{' {
		var one struct {
			Frames *[]string `json:"frames"`
		}
		if json.Unmarshal(trim, &one) == nil && one.Frames != nil {
			return FromStrings("", *one.Frames), nil
		}
		var many map[string]spinnerJSON
		if err := json.Unmarshal(trim, &many); err == nil && len(many) > 0 {
			names := make([]string, 0, len(many))
			for n := range many {
				names = append(names, n)
			}
			sort.Strings(names)
			var out []Frame
			for _, n := range names {
				out = append(out, FromStrings(n, many[n].Frames)...)
			}
			return out, nil
		}
		if json.Valid(trim) {
			return nil, ErrNoFrames
		}
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	var frames []Frame
	cur := Frame{Base: 1}
	for i, l := range lines {
		if sep != "" && (l == sep || strings.HasPrefix(l, sep+" ")) {
			if len(cur.Lines) > 0 || i > 0 {
				frames = append(frames, cur)
			}
			cur = Frame{Index: len(frames), Base: i + 2}
			continue
		}
		cur.Lines = append(cur.Lines, l)
	}
	frames = append(frames, cur)
	// Drop empty frames produced by a leading separator or blank spacers
	// between frames (render output puts a blank line before each header).
	var out []Frame
	for _, f := range frames {
		for len(f.Lines) > 0 && f.Lines[len(f.Lines)-1] == "" {
			f.Lines = f.Lines[:len(f.Lines)-1]
		}
		if len(f.Lines) == 0 {
			continue
		}
		f.Index = len(out)
		out = append(out, f)
	}
	return out, nil
}

// Width sums cell widths, counting unsafe runes at their likely width.
func Width(s string) int {
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

// RuneReason names the problem with a rune in animated output, and says what
// to do instead. An empty kind means the rune is safe.
func RuneReason(r rune) (kind, why string) {
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

// Check runs every rule over frames and returns the findings, sorted by set,
// frame, line and column. file is copied into each issue's File field and is
// not read.
func Check(file string, frames []Frame) []Issue {
	issues := []Issue{}
	add := func(f Frame, line, col int, kind, msg string, r rune) {
		is := Issue{File: file, Set: f.Set, Frame: f.Index, Line: line, Col: col, Kind: kind, Message: msg}
		if f.Base > 0 {
			is.Line = f.Base + line - 1
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
		widths := make([]int, len(f.Lines))
		trailing := 0
		for li, l := range f.Lines {
			col := 1
			for _, r := range l {
				if kind, why := RuneReason(r); kind != "" {
					add(f, li+1, col, kind, fmt.Sprintf("%q U+%04X: %s", string(r), r, why), r)
				}
				col++
			}
			widths[li] = Width(l)
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
		if ragged && trailing > 0 && trailing < len(f.Lines) {
			flagPadded := trailing <= len(f.Lines)-trailing
			for li, l := range f.Lines {
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
		d := dims{w: frameW, h: len(f.Lines)}
		if prev, ok := setDims[f.Set]; !ok {
			setDims[f.Set], setFirst[f.Set] = d, f.Index
		} else {
			if d.h != prev.h {
				add(f, 1, 1, "frame-height", fmt.Sprintf("frame is %d lines tall; frame %d is %d (frames of different heights leave stale rows)", d.h, setFirst[f.Set], prev.h), 0)
			}
			if d.w != prev.w {
				add(f, 1, 1, "frame-width", fmt.Sprintf("frame is %d columns wide; frame %d is %d (frames of different widths leave stale columns)", d.w, setFirst[f.Set], prev.w), 0)
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
