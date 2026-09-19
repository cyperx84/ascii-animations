package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cyperx84/ascii-animations/asciifx/lint"
)

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
	frames, err := lint.Split(data, *sep)
	if err != nil {
		if errors.Is(err, lint.ErrNoFrames) {
			return usageErr(err.Error(), "JSON input has no frames")
		}
		return runtimeErr(err, "check the input format")
	}
	res := lint.Result{File: path, Frames: len(frames), Issues: lint.Check(path, frames)}
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
