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
	"time"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// runFlags are the flags shared by every command that builds a run.
type runFlags struct {
	chain   *chain
	text    string
	banner  string
	font    string
	file    string
	seed    uint64
	w, h    int
	fps     int
	profile string
	dither  string
}

// step is one link in a chain: an effect plus the flags scoped to it.
type step struct {
	name   string
	params map[string]string
	// forSeconds gives a looping effect a duration so the chain can advance.
	forSeconds float64
}

// chain collects the effect chain in command-line order. Index 0 is the
// positional effect and each --then appends one. -p and --for apply to the
// step declared most recently, so flags read in the order they are written:
//
//	asciifx render reveal -p pattern=center --then shine -p palette=aurora
//
// The flag package calls Set in command-line order, so tracking "the step
// named last" is enough; no separate index syntax is needed.
type chain struct {
	steps []step
	cur   int
}

func newChain() *chain {
	return &chain{steps: []step{{params: map[string]string{}}}}
}

// chained reports whether any --then step was given, which is what decides
// between the plain single-effect path and a composition.
func (c *chain) chained() bool { return len(c.steps) > 1 }

func (c *chain) add(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("--then needs an effect name, e.g. --then shine")
	}
	c.steps = append(c.steps, step{name: name, params: map[string]string{}})
	c.cur = len(c.steps) - 1
	return nil
}

func (c *chain) setParam(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || strings.TrimSpace(k) == "" {
		return fmt.Errorf("param %q must be key=value, e.g. -p palette=synthwave", s)
	}
	c.steps[c.cur].params[strings.TrimSpace(k)] = v
	return nil
}

func (c *chain) setFor(s string) error {
	s = strings.TrimSpace(s)
	d, err := time.ParseDuration(s)
	if err != nil {
		// A bare number is seconds, which is what people write.
		if secs, ferr := strconv.ParseFloat(s, 64); ferr == nil {
			d, err = time.Duration(secs*float64(time.Second)), nil
		}
	}
	if err != nil || d <= 0 {
		return fmt.Errorf("--for wants a positive duration such as 2s or 500ms, got %q", s)
	}
	if d > maxForDuration {
		return fmt.Errorf("--for %s is longer than the %s limit", d, maxForDuration)
	}
	c.steps[c.cur].forSeconds = d.Seconds()
	return nil
}

// maxForDuration bounds one step of a chain, matching the render and cast
// limits so a seek cannot be made arbitrarily expensive from the CLI.
const maxForDuration = time.Duration(maxSeconds) * time.Second

// flag Values. They share one chain so ordering across the flags is
// meaningful.
type chainParamValue struct{ c *chain }

func (v *chainParamValue) String() string { return "" }
func (v *chainParamValue) Set(s string) error {
	return v.c.setParam(s)
}

type chainThenValue struct{ c *chain }

func (v *chainThenValue) String() string { return "" }
func (v *chainThenValue) Set(s string) error {
	return v.c.add(s)
}

type chainForValue struct{ c *chain }

func (v *chainForValue) String() string { return "" }
func (v *chainForValue) Set(s string) error {
	return v.c.setFor(s)
}

func addRunFlags(fs *flag.FlagSet) *runFlags {
	rf := &runFlags{chain: newChain()}
	fs.Var(&chainParamValue{rf.chain}, "p", "effect param key=value (repeatable); applies to the effect named last")
	fs.Var(&chainThenValue{rf.chain}, "then", "play another effect after the previous one (repeatable)")
	fs.Var(&chainForValue{rf.chain}, "for", "give the effect named last a duration, e.g. 2s (required to chain a looping effect)")
	fs.StringVar(&rf.text, "text", "", `content text for transitions; "\n" starts a new line`)
	fs.StringVar(&rf.banner, "banner", "", "content rendered as a banner font")
	fs.StringVar(&rf.font, "font", "block", "banner font")
	fs.StringVar(&rf.file, "file", "", "content file, or - for stdin")
	fs.Uint64Var(&rf.seed, "seed", 1, "random seed")
	fs.IntVar(&rf.w, "w", 0, "width in cells (default: effect default, grown to fit content)")
	fs.IntVar(&rf.h, "h", 0, "height in cells")
	fs.IntVar(&rf.fps, "fps", 0, "tick rate (default: effect's recommended fps)")
	fs.StringVar(&rf.profile, "profile", "", "colour profile: truecolor, 256, 16, none (default: detected)")
	fs.StringVar(&rf.dither, "dither", "", "ordered dither for 16/256 colour: bayer8 (default), bayer4, none")
	return rf
}

// colorPrefs resolves --profile and --dither together, because both default
// from the same environment detection pass. Dithering is dropped for profiles
// that have no palette to dither onto.
func (rf *runFlags) colorPrefs(out *os.File) (term.Profile, tint.Dither, error) {
	caps := term.Detect(out)
	if rf.profile != "" {
		p, err := term.ParseProfile(rf.profile)
		if err != nil {
			return 0, tint.NoDither, usageErr("valid profiles: truecolor, 256, 16, none", "%v", err)
		}
		caps.Profile = p
	}
	d := caps.Dither
	if rf.dither != "" {
		parsed, err := tint.ParseDither(rf.dither)
		if err != nil {
			return 0, tint.NoDither, usageErr("valid dithers: none, bayer4, bayer8", "%v", err)
		}
		d = parsed
	}
	return caps.Profile, term.DitherFor(caps.Profile, d), nil
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

// checkContent rejects runes that are not single-width instead of letting
// fx.Text quietly draw them as '?'. Tabs are fine: fx.Text expands them.
func checkContent(s string) error {
	for i, line := range strings.Split(s, "\n") {
		col := 0
		for _, r := range line {
			col++
			if r == '\t' {
				continue
			}
			if cell.Width(r) != 1 {
				return usageErr(
					"content must use single-width glyphs (ASCII, box, block, braille); run `asciifx check` on the text to list every problem",
					"content line %d col %d: %q (U+%04X) is not single-width", i+1, col, r, r)
			}
		}
	}
	return nil
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
	// params is the resolved parameter set to report. For a chain it is
	// prefixed per step, because a chain has no single parameter set.
	params map[string]string
	// steps names the effect of each link, empty for a single effect.
	steps []string
}

// resolve turns the positional effect plus every --then into the spec to run
// and the per-step specs it was built from. With no --then the registered
// spec is used unchanged, so a plain invocation behaves exactly as it always
// has.
func (rf *runFlags) resolve(name string) (*fx.Spec, []*fx.Spec, error) {
	steps := rf.chain.steps
	steps[0].name = name
	for i, st := range steps {
		if st.name == "" {
			return nil, nil, usageErr("give the effect name before any --then", "step %d has no effect", i+1)
		}
	}
	// Validate each step with the flag-aware checker, so errors carry the
	// did-you-mean and range hints the library cannot know about.
	specs := make([]*fx.Spec, len(steps))
	for i, st := range steps {
		s, err := lookupSpec(st.name)
		if err != nil {
			return nil, nil, wrapStep(i, err)
		}
		specs[i] = s
		if err := validateParams(s, st.params); err != nil {
			return nil, nil, wrapStep(i, err)
		}
	}
	if !rf.chain.chained() {
		return specs[0], specs, nil
	}
	fxSteps := make([]fx.Step, len(steps))
	for i, st := range steps {
		fxSteps[i] = fx.Step{Name: st.name, Params: st.params, For: st.forSeconds}
	}
	spec, err := fx.Compose(fxSteps...)
	if err != nil {
		return nil, nil, usageErr("a chain runs each effect in turn, so a looping effect needs --for, e.g. `--then fire --for 2s`", "%v", err)
	}
	return spec, specs, nil
}

// wrapStep names the failing step without losing the error's exit code.
func wrapStep(i int, err error) error {
	var ce *cliError
	if errors.As(err, &ce) {
		return &cliError{code: ce.code, msg: fmt.Sprintf("step %d: %s", i+1, ce.msg), hint: ce.hint}
	}
	return fmt.Errorf("step %d: %w", i+1, err)
}

// build validates everything and constructs the run. fit leaves the size to
// the caller (play resizes to the terminal).
func (rf *runFlags) build(e *env, name string) (*built, error) {
	spec, specs, err := rf.resolve(name)
	if err != nil {
		return nil, err
	}
	// A chain has no single parameter set, so report each step's resolved
	// params under its position and hand the engine none: the composed spec
	// resolves its children from the step table, not from Run.Values. A single
	// effect reports and passes its own params, unprefixed, exactly as before.
	reported := map[string]string{}
	var stepNames []string
	oParams := map[string]string(nil)
	if rf.chain.chained() {
		stepNames = make([]string, len(rf.chain.steps))
		for i, st := range rf.chain.steps {
			stepNames[i] = st.name
			resolved, err := specs[i].Resolve(st.params)
			if err != nil {
				return nil, runtimeErr(err, "")
			}
			for k, v := range resolved.Map() {
				reported[fmt.Sprintf("%d.%s", i+1, k)] = v
			}
		}
	} else {
		oParams = rf.chain.steps[0].params
		resolved, err := specs[0].Resolve(oParams)
		if err != nil {
			return nil, runtimeErr(err, "")
		}
		reported = resolved.Map()
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
	o := fx.Options{Params: oParams, W: rf.w, H: rf.h, Seed: rf.seed, FPS: rf.fps}
	if spec.Content {
		s, _, err := rf.content(e.stdin)
		if err != nil {
			return nil, err
		}
		s = strings.ReplaceAll(s, "\r\n", "\n")
		if err := checkContent(s); err != nil {
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
	return &built{spec: spec, run: r, seed: rf.seed, params: reported, steps: stepNames}, nil
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
