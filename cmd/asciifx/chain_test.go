package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
)

// frameOf runs a render invocation and decodes the single JSON frame.
func frameOf(t *testing.T, args ...string) frameJSON {
	t.Helper()
	code, stdout, stderr := runCLI(t, "", append(args, "--json")...)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, stdout, stderr)
	}
	var f frameJSON
	if err := json.Unmarshal([]byte(stdout), &f); err != nil {
		t.Fatalf("decode %q: %v", stdout, err)
	}
	return f
}

func TestChainReportsItsSteps(t *testing.T) {
	f := frameOf(t, "render", "reveal", "--text", "HI", "--w", "24", "--h", "6",
		"-p", "pattern=center", "--then", "shine", "--for", "1s", "-p", "palette=matrix",
		"--frame", "0", "--seed", "1")
	if f.Effect != "reveal+shine" {
		t.Errorf("effect = %q, want the joined step names", f.Effect)
	}
	if len(f.Steps) != 2 || f.Steps[0] != "reveal" || f.Steps[1] != "shine" {
		t.Fatalf("steps = %v", f.Steps)
	}
	// -p is scoped to the effect named last, so the first keeps its default.
	if got := f.Params["1.pattern"]; got != "center" {
		t.Errorf("1.pattern = %q, want center", got)
	}
	if got := f.Params["1.palette"]; got != "aurora" {
		t.Errorf("1.palette = %q, want reveal's default (aurora)", got)
	}
	if got := f.Params["2.palette"]; got != "matrix" {
		t.Errorf("2.palette = %q, want matrix", got)
	}
	// reveal is 1.6s and shine 1s, both at 30fps: 2.6s rounds up to 79 frames.
	if f.FramesTotal != 79 {
		t.Errorf("frames_total = %d, want 79", f.FramesTotal)
	}
}

func TestChainOverridesAFiniteDuration(t *testing.T) {
	f := frameOf(t, "render", "reveal", "--text", "HI", "--w", "24", "--h", "6",
		"--for", "500ms", "--then", "shine", "--for", "1s", "--frame", "0")
	// 0.5s + 1s at 30fps rounds up to 46 frames.
	if f.FramesTotal != 46 {
		t.Errorf("frames_total = %d, want 46 (--for should shorten reveal)", f.FramesTotal)
	}
}

func TestSingleEffectIsUnchanged(t *testing.T) {
	f := frameOf(t, "render", "reveal", "--text", "HI", "--w", "24", "--h", "6", "-p", "pattern=center", "--frame", "0")
	if f.Effect != "reveal" {
		t.Errorf("effect = %q", f.Effect)
	}
	if len(f.Steps) != 0 {
		t.Errorf("a single effect must not report steps: %v", f.Steps)
	}
	if _, prefixed := f.Params["1.pattern"]; prefixed {
		t.Errorf("a single effect must report unprefixed params: %v", f.Params)
	}
	if got := f.Params["pattern"]; got != "center" {
		t.Errorf("pattern = %q, want center", got)
	}
}

func TestSingleEffectKeepsItsDefaultsResolved(t *testing.T) {
	// The JSON contract is that params are resolved, not just echoed, so the
	// refactor onto the chain must not have made them raw.
	f := frameOf(t, "render", "fire", "--w", "24", "--h", "6", "--frame", "0")
	spec, err := fx.Lookup("fire")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range spec.Params {
		if _, ok := f.Params[p.Name]; !ok {
			t.Errorf("param %q missing from the resolved report", p.Name)
		}
	}
}

func TestChainErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"looping step needs a duration",
			[]string{"render", "reveal", "--text", "HI", "--then", "shine"},
			"--for"},
		{"unknown step name",
			[]string{"render", "reveal", "--text", "HI", "--then", "shnie"},
			"did you mean \"shine\""},
		{"unknown param on a later step",
			[]string{"render", "reveal", "--text", "HI", "--then", "shine", "--for", "1s", "-p", "pattern=center"},
			"step 2"},
		{"empty --then",
			[]string{"render", "reveal", "--text", "HI", "--then", ""},
			"--then needs an effect name"},
		{"bad --for",
			[]string{"render", "reveal", "--text", "HI", "--for", "never"},
			"positive duration"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, "", c.args...)
			if code != exitUsage {
				t.Fatalf("exit %d, want %d (stdout %q stderr %q)", code, exitUsage, stdout, stderr)
			}
			if !strings.Contains(stderr, c.want) {
				t.Fatalf("stderr %q lacks %q", stderr, c.want)
			}
		})
	}
}

func TestChainPlayAndCastBuild(t *testing.T) {
	// Both commands share the run builder, so a chain must at least construct
	// and produce a stream rather than failing on the synthetic spec.
	code, stdout, stderr := runCLI(t, "", "cast", "reveal", "--text", "HI", "--w", "20", "--h", "4",
		"--then", "shine", "--for", "1s", "--seconds", "0.2", "--profile", "256")
	if code != 0 {
		t.Fatalf("cast exit %d: %s", code, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 3 {
		t.Fatalf("cast produced %d lines", len(lines))
	}
	var header map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &header); err != nil {
		t.Fatalf("first line is not the cast header: %v", err)
	}
	if title, _ := header["title"].(string); title != "asciifx reveal+shine" {
		t.Errorf("cast title = %q", title)
	}
}

// TestFilterBindsToTheLastNamedEffect checks --filter is step-scoped like --for,
// and that the binding is visible in the JSON an agent reads.
func TestFilterBindsToTheLastNamedEffect(t *testing.T) {
	f := frameOf(t, "render", "reveal", "--text", "HI", "--w", "24", "--h", "6",
		"--then", "fire", "--for", "1s", "--filter", "not(ink)", "--frame", "0", "--seed", "1")
	if got := f.Params["2.filter"]; got != "not(ink)" {
		t.Errorf("2.filter = %q, want not(ink)", got)
	}
	if _, ok := f.Params["1.filter"]; ok {
		t.Error("the filter leaked onto step 1")
	}

	// On a single effect it is reported unprefixed, like the params are.
	one := frameOf(t, "render", "glitch", "--text", "HI", "--w", "20", "--h", "4", "--frame", "3", "--filter", "outer(1)")
	if got := one.Params["filter"]; got != "outer(1)" {
		t.Errorf("filter = %q, want outer(1)", got)
	}
	// The last-named effect is the one that gets it, so --filter before --then
	// applies to the first effect; on a chain that is step 1.
	two := frameOf(t, "render", "reveal", "--text", "HI", "--w", "24", "--h", "6", "--filter", "inner(1)",
		"--then", "reveal", "--for", "500ms", "--frame", "0", "--seed", "1")
	if got := two.Params["1.filter"]; got != "inner(1)" {
		t.Errorf("1.filter = %q, want inner(1)", got)
	}
}

func TestFilterErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"a content selector needs content",
			[]string{"render", "fire", "--w", "20", "--h", "5", "--filter", "ink"},
			"no content"},
		{"...and says how to fix it",
			[]string{"render", "fire", "--w", "20", "--h", "5", "--filter", "not(ink)"},
			"--then"},
		{"a typo gets a suggestion",
			[]string{"render", "reveal", "--text", "HI", "--filter", "nto(ink)"},
			`did you mean "not"`},
		{"an unknown name gets a suggestion",
			[]string{"render", "reveal", "--text", "HI", "--filter", "nke"},
			`did you mean "ink"`},
		{"a bad expression is reported with its step",
			[]string{"render", "reveal", "--text", "HI", "--then", "fire", "--for", "1s", "--filter", "all(ink"},
			"step 2"},
		{"an empty filter is a usage error",
			[]string{"render", "reveal", "--text", "HI", "--filter", ""},
			"--filter needs a selector"},
		{"a geometry filter is fine without content",
			[]string{"render", "fire", "--w", "20", "--h", "5", "--filter", "outer(1)", "--frame", "3"},
			""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, "", c.args...)
			if c.want == "" {
				if code != exitOK {
					t.Fatalf("exit %d, stderr %q", code, stderr)
				}
				return
			}
			if code != exitUsage {
				t.Fatalf("exit %d, want %d (stderr %q)", code, exitUsage, stderr)
			}
			if !strings.Contains(stderr, c.want) {
				t.Fatalf("stderr %q lacks %q", stderr, c.want)
			}
		})
	}
}

// TestFilterChangesTheFrame is the CLI-level proof that a filter does something:
// the same invocation with and without it must differ, and the filtered one must
// keep the banner fire would otherwise consume.
func TestFilterChangesTheFrame(t *testing.T) {
	base := []string{"render", "reveal", "--banner", "HI", "--w", "24", "--h", "8",
		"--frame", "-1", "--format", "plain", "--seed", "1", "--then", "fire", "--for", "1s"}
	_, unfiltered, _ := runCLI(t, "", base...)
	_, filtered, _ := runCLI(t, "", append(append([]string{}, base...), "--filter", "not(ink)")...)
	if unfiltered == filtered {
		t.Fatal("the filter changed nothing")
	}
	if strings.Contains(unfiltered, "███") {
		t.Fatal("fire was supposed to consume the banner without the filter")
	}
	if !strings.Contains(filtered, "███") {
		t.Fatalf("the filtered frame lost the banner:\n%s", filtered)
	}
}
