package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// runFlags are the flags shared by every command that builds a run.
type runFlags struct {
	params  paramFlag
	text    string
	banner  string
	font    string
	file    string
	seed    uint64
	w, h    int
	fps     int
	profile string
}

func addRunFlags(fs *flag.FlagSet) *runFlags {
	rf := &runFlags{params: paramFlag{}}
	fs.Var(rf.params, "p", "effect param key=value (repeatable)")
	fs.StringVar(&rf.text, "text", "", `content text for transitions; "\n" starts a new line`)
	fs.StringVar(&rf.banner, "banner", "", "content rendered as a banner font")
	fs.StringVar(&rf.font, "font", "block", "banner font")
	fs.StringVar(&rf.file, "file", "", "content file, or - for stdin")
	fs.Uint64Var(&rf.seed, "seed", 1, "random seed")
	fs.IntVar(&rf.w, "w", 0, "width in cells (default: effect default, grown to fit content)")
	fs.IntVar(&rf.h, "h", 0, "height in cells")
	fs.IntVar(&rf.fps, "fps", 0, "tick rate (default: effect's recommended fps)")
	fs.StringVar(&rf.profile, "profile", "", "colour profile: truecolor, 256, 16, none (default: detected)")
	return rf
}

// colorProfile resolves --profile, falling back to environment detection.
func (rf *runFlags) colorProfile(out *os.File) (term.Profile, error) {
	if rf.profile == "" {
		return term.Detect(out).Profile, nil
	}
	p, err := term.ParseProfile(rf.profile)
	if err != nil {
		return 0, usageErr("valid profiles: truecolor, 256, 16, none", "%v", err)
	}
	return p, nil
}

// lookupSpec finds an effect with a did-you-mean hint.
func lookupSpec(name string) (*fx.Spec, error) {
	if name == "" {
		return nil, usageErr("run `asciifx list` to see available effects", "missing effect name")
	}
	spec, err := fx.Lookup(name)
	if err == nil {
		return spec, nil
	}
	all := fx.All()
	names := make([]string, len(all))
	for i, s := range all {
		names[i] = s.Name
	}
	hint := "available effects: " + strings.Join(names, ", ")
	if s := suggest(name, names); s != "" {
		hint = fmt.Sprintf("did you mean %q? %s", s, hint)
	}
	return nil, usageErr(hint, "unknown effect %q", name)
}

// validateParams checks params up front so errors carry precise hints.
func validateParams(spec *fx.Spec, params map[string]string) error {
	names := make([]string, len(spec.Params))
	for i, p := range spec.Params {
		names[i] = p.Name
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := params[k]
		var p *fx.Param
		for i := range spec.Params {
			if spec.Params[i].Name == k {
				p = &spec.Params[i]
			}
		}
		if p == nil {
			hint := fmt.Sprintf("valid params for %s: %s; run `asciifx info %s` for types and ranges", spec.Name, strings.Join(names, ", "), spec.Name)
			if len(names) == 0 {
				hint = fmt.Sprintf("%s takes no params", spec.Name)
			}
			if s := suggest(k, names); s != "" {
				hint = fmt.Sprintf("did you mean %q? %s", s, hint)
			}
			return usageErr(hint, "effect %s has no param %q", spec.Name, k)
		}
		if _, err := spec.Resolve(map[string]string{k: v}); err != nil {
			return usageErr(paramHint(spec, p, err.Error()), "%v", err)
		}
	}
	return nil
}

func paramHint(spec *fx.Spec, p *fx.Param, msg string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "-p %s takes %s", p.Name, p.Type)
	listed := len(p.Options) > 0 && strings.Contains(msg, p.Options[0]) && strings.Contains(msg, p.Options[len(p.Options)-1])
	switch {
	case listed:
	case len(p.Options) > 0:
		fmt.Fprintf(&sb, ", one of: %s", strings.Join(p.Options, ", "))
	case p.Min != nil && p.Max != nil:
		fmt.Fprintf(&sb, " in [%g, %g]", *p.Min, *p.Max)
	}
	switch {
	case listed:
	case p.Type == fx.Palette:
		sb.WriteString(`, or hex stops like "#ff0000,#0000ff"`)
	case p.Type == fx.ColorParam:
		sb.WriteString(`, as "#rrggbb" or "none"`)
	}
	fmt.Fprintf(&sb, " (default %q); see `asciifx info %s`", p.Default, spec.Name)
	return sb.String()
}

// content resolves --text, --banner and --file into content, its text and
// whether the user supplied it.
func (rf *runFlags) content(stdin io.Reader) (string, bool, error) {
	set := 0
	for _, s := range []string{rf.text, rf.banner, rf.file} {
		if s != "" {
			set++
		}
	}
	if set > 1 {
		return "", false, usageErr("pass exactly one content source", "--text, --banner and --file are mutually exclusive")
	}
	switch {
	case rf.text != "":
		return strings.ReplaceAll(rf.text, `\n`, "\n"), true, nil
	case rf.banner != "":
		s, err := banner(rf.banner, rf.font)
		if err != nil {
			return "", false, usageErr("fonts: "+strings.Join(bannerFonts(), ", "), "%v", err)
		}
		return s, true, nil
	case rf.file != "":
		var data []byte
		var err error
		if rf.file == "-" {
			data, err = io.ReadAll(stdin)
		} else {
			data, err = os.ReadFile(rf.file)
		}
		if err != nil {
			return "", false, runtimeErr(err, "check the --file path, or use --file - to read stdin")
		}
		return string(data), true, nil
	}
	s, err := banner("ASCIIFX", "block")
	if err != nil {
		s = "ASCIIFX"
	}
	return s, false, nil
}

func textSize(s string) (w, h int) {
	s = strings.TrimRight(strings.ReplaceAll(s, "\t", "    "), "\n")
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		w = max(w, len([]rune(l)))
	}
	return w, len(lines)
}

// built is a validated run plus what went into it.
type built struct {
	spec *fx.Spec
	run  *fx.Run
	seed uint64
}

// build validates everything and constructs the run. fit leaves the size to
// the caller (play resizes to the terminal).
func (rf *runFlags) build(e *env, name string) (*built, error) {
	spec, err := lookupSpec(name)
	if err != nil {
		return nil, err
	}
	if err := validateParams(spec, rf.params); err != nil {
		return nil, err
	}
	if (rf.w != 0) != (rf.h != 0) {
		return nil, usageErr(fmt.Sprintf("default size for %s is --w %d --h %d", spec.Name, spec.DefW, spec.DefH), "--w and --h must be given together")
	}
	if rf.w < 0 || rf.h < 0 || rf.fps < 0 {
		return nil, usageErr("sizes and fps must be positive", "negative --w, --h or --fps")
	}
	if rf.fps > 240 {
		return nil, usageErr("typical rates are 10-60", "--fps %d is too high", rf.fps)
	}
	o := fx.Options{Params: map[string]string(rf.params), W: rf.w, H: rf.h, Seed: rf.seed, FPS: rf.fps}
	if spec.Content {
		s, _, err := rf.content(e.stdin)
		if err != nil {
			return nil, err
		}
		o.Content = fx.Text(s, tint.None)
		if rf.w == 0 {
			cw, ch := textSize(s)
			o.W, o.H = max(spec.DefW, cw+2), max(spec.DefH, ch+2)
		}
	} else if rf.text != "" || rf.banner != "" || rf.file != "" {
		fmt.Fprintf(e.stderr, "asciifx: note: %s is %s and ignores content flags\n", spec.Name, spec.Kind)
	}
	w, h := o.W, o.H
	if w == 0 {
		w, h = spec.DefW, spec.DefH
	}
	if w < spec.MinW || h < spec.MinH {
		return nil, usageErr(fmt.Sprintf("use at least --w %d --h %d", spec.MinW, spec.MinH), "effect %s needs at least %dx%d, got %dx%d", spec.Name, spec.MinW, spec.MinH, w, h)
	}
	r, err := fx.NewRun(spec, o)
	if err != nil {
		return nil, runtimeErr(err, fmt.Sprintf("see `asciifx info %s`", spec.Name))
	}
	return &built{spec: spec, run: r, seed: rf.seed}, nil
}

func oneEffect(c *command, pos []string) (string, error) {
	switch len(pos) {
	case 1:
		return pos[0], nil
	case 0:
		return "", usageErr(fmt.Sprintf("usage: %s; run `asciifx list` for effects", c.usage), "%s needs an effect name", c.name)
	}
	return "", usageErr("usage: "+c.usage, "unexpected arguments: %s", strings.Join(pos[1:], " "))
}

func parseIntList(s string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, errors.New("not an integer: " + part)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, errors.New("empty list")
	}
	return out, nil
}
