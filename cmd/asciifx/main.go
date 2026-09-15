// Command asciifx renders, previews, lints and exports asciifx terminal
// animations. It is built for two audiences: humans at a terminal, and
// coding agents that cannot watch motion and need frames as text, JSON
// envelopes and stable exit codes.
//
// Exit codes: 0 ok, 1 runtime error, 2 usage error, 3 check found problems.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
)

const (
	exitOK     = 0
	exitError  = 1
	exitUsage  = 2
	exitIssues = 3
)

var version = ""

func versionString() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

// env carries the process streams so tests can drive run directly.
type env struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// cliError is an error with an exit code and a separate actionable hint.
type cliError struct {
	code int
	msg  string
	hint string
}

func (e *cliError) Error() string { return e.msg }

func usageErr(hint, format string, args ...any) *cliError {
	return &cliError{code: exitUsage, msg: fmt.Sprintf(format, args...), hint: hint}
}

func runtimeErr(err error, hint string) *cliError {
	return &cliError{code: exitError, msg: err.Error(), hint: hint}
}

type command struct {
	name    string
	summary string
	usage   string
	details string
	run     func(e *env, args []string) error
}

var commands []*command

func findCommand(name string) *command {
	for _, c := range commands {
		if c.name == name {
			return c
		}
	}
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], &env{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}))
}

func run(args []string, e *env) int {
	if len(args) == 0 {
		fmt.Fprint(e.stderr, mainHelp())
		return exitUsage
	}
	name := args[0]
	switch name {
	case "-h", "-help", "--help":
		fmt.Fprint(e.stdout, mainHelp())
		return exitOK
	case "-v", "-version", "--version":
		name = "version"
	}
	c := findCommand(name)
	if c == nil {
		names := make([]string, len(commands))
		for i, c := range commands {
			names[i] = c.name
		}
		hint := "commands: " + strings.Join(names, ", ") + "; run `asciifx help` for usage"
		if s := suggest(name, names); s != "" {
			hint = fmt.Sprintf("did you mean `asciifx %s`? %s", s, hint)
		}
		return report(e, wantsJSON(args), usageErr(hint, "unknown command %q", name))
	}
	if err := c.run(e, args[1:]); err != nil {
		return report(e, wantsJSON(args), err)
	}
	return exitOK
}

// wantsJSON scans raw args so even flag-parse failures honour --json.
func wantsJSON(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
		switch a {
		case "--json", "-json", "--json=true", "-json=true":
			return true
		}
		if strings.HasPrefix(a, "--format=json") || strings.HasPrefix(a, "-format=json") {
			return true
		}
	}
	for i, a := range args {
		if (a == "--format" || a == "-format") && i+1 < len(args) && args[i+1] == "json" {
			return true
		}
	}
	return false
}

var errHelpShown = errors.New("help shown")

// report prints err in the requested shape and returns the exit code.
func report(e *env, asJSON bool, err error) int {
	if errors.Is(err, errHelpShown) {
		return exitOK
	}
	var ce *cliError
	if !errors.As(err, &ce) {
		ce = &cliError{code: exitError, msg: err.Error()}
	}
	if ce.msg == "" {
		// The command already reported (e.g. check listing its issues).
		return ce.code
	}
	if asJSON {
		out := struct {
			Error string `json:"error"`
			Hint  string `json:"hint"`
		}{ce.msg, ce.hint}
		writeJSON(e.stdout, out)
	} else {
		fmt.Fprintf(e.stderr, "asciifx: %s\n", ce.msg)
		if ce.hint != "" {
			fmt.Fprintf(e.stderr, "hint: %s\n", ce.hint)
		}
	}
	return ce.code
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newFlags returns a silent flag set; parse errors are reported by parse.
func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// parse handles flags and positionals in any order, so both
// `render fire --frame 3` and `render --frame 3 fire` work.
func parse(e *env, c *command, fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Fprint(e.stdout, commandHelp(c))
				return nil, errHelpShown
			}
			return nil, usageErr(fmt.Sprintf("run `asciifx help %s` for flags and examples", c.name), "%s: %v", c.name, err)
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		// flag stops at "--"; everything after it is positional.
		if len(args) > len(rest) && args[len(args)-len(rest)-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// suggest returns the closest name within a small edit distance, or "".
func suggest(name string, names []string) string {
	best, bestD := "", 1<<30
	for _, n := range names {
		d := levenshtein(strings.ToLower(name), strings.ToLower(n))
		if strings.HasPrefix(n, name) && len(name) >= 2 {
			d = min(d, 1)
		}
		if d < bestD {
			best, bestD = n, d
		}
	}
	if bestD <= max(2, len(name)/3) {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
