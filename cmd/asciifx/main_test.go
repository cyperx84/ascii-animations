package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &env{stdin: strings.NewReader(stdin), stdout: &out, stderr: &errb})
	return code, out.String(), errb.String()
}

func TestRenderJSONDeterministic(t *testing.T) {
	args := []string{"render", "reveal", "--text", `HI\nTHERE`, "--w", "12", "--h", "4", "--frames", "0,20,-1", "--format", "json", "-p", "palette=synthwave"}
	code, first, stderr := runCLI(t, "", args...)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, first, stderr)
	}
	for i := 0; i < 3; i++ {
		if _, again, _ := runCLI(t, "", args...); again != first {
			t.Fatal("render --format json output differs between runs")
		}
	}
	var frames []frameJSON
	if err := json.Unmarshal([]byte(first), &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Fatalf("want 3 frames, got %d", len(frames))
	}
	last := frames[2]
	if last.Tick != last.FramesTotal-1 || last.Params["palette"] != "synthwave" || last.W != 12 || len(last.Lines) != 4 || len(last.Styles) != 4 {
		t.Fatalf("unexpected last frame: %+v", last)
	}
	if !strings.Contains(strings.Join(last.Lines, "\n"), "THERE") {
		t.Fatalf("final frame lacks content:\n%s", strings.Join(last.Lines, "\n"))
	}
	if len(last.Styles[1]) == 0 || last.Styles[1][0].FG == "none" {
		t.Fatal("final frame has no colour runs")
	}
	// A different seed must still be deterministic, and ambient too.
	a := []string{"render", "fire", "--w", "16", "--h", "4", "--at", "0.5", "--json", "--seed", "7"}
	_, x, _ := runCLI(t, "", a...)
	_, y, _ := runCLI(t, "", a...)
	if x != y || !strings.HasPrefix(x, "{") {
		t.Fatal("ambient json render not deterministic")
	}
}

func TestRenderErrors(t *testing.T) {
	cases := []struct {
		args []string
		hint string
	}{
		{[]string{"render", "fier", "--json"}, `did you mean \"fire\"`},
		{[]string{"render", "fire", "-p", "cooler=1", "--json"}, "valid params for fire"},
		{[]string{"render", "fire", "-p", "palette=nope", "--json"}, "info fire"},
		{[]string{"render", "fire", "--frame", "-1", "--json"}, "loops forever"},
		{[]string{"render", "reveal", "--frame", "999", "--json"}, "0.."},
		{[]string{"render", "fire", "--nope", "--json"}, "help render"},
	}
	for _, c := range cases {
		code, out, _ := runCLI(t, "", c.args...)
		if code != exitUsage {
			t.Errorf("%v: exit %d, want 2", c.args, code)
		}
		var env struct{ Error, Hint string }
		if err := json.Unmarshal([]byte(out), &env); err != nil || env.Error == "" || env.Hint == "" {
			t.Errorf("%v: bad error envelope %q", c.args, out)
		}
		if !strings.Contains(out, c.hint) {
			t.Errorf("%v: hint lacks %q: %s", c.args, c.hint, out)
		}
	}
}

func TestCheckExitCodes(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	bad := filepath.Join(dir, "bad.txt")
	os.WriteFile(good, []byte("+--+\n|  |\n+--+\n---\n+--+\n|##|\n+--+\n"), 0o644)
	os.WriteFile(bad, []byte("ab\U0001F525cd\n\tx\nshort\n---\nab\n"), 0o644)

	if code, out, errs := runCLI(t, "", "check", good); code != 0 {
		t.Fatalf("good file: exit %d\n%s%s", code, out, errs)
	}
	code, out, _ := runCLI(t, "", "check", bad)
	if code != exitIssues {
		t.Fatalf("bad file: exit %d, want 3", code)
	}
	for _, want := range []string{":0:1:3:", "U+1F525", "emoji", "tab", "columns wide", "lines tall"} {
		if !strings.Contains(out, want) {
			t.Errorf("check output lacks %q:\n%s", want, out)
		}
	}
	code, out, _ = runCLI(t, "", "check", "--json", bad)
	var res checkResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || code != exitIssues || res.OK || res.Frames != 2 {
		t.Fatalf("json check: exit %d, %+v, %v", code, res, err)
	}
	// cli-spinners JSON on stdin.
	if code, out, _ := runCLI(t, `{"interval":80,"frames":["-","\\","|","/"]}`, "check", "--json", "-"); code != 0 || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("spinner json: exit %d %s", code, out)
	}
	// render output round-trips through check.
	_, frames, _ := runCLI(t, "", "render", "reveal", "--text", "OK", "--w", "6", "--h", "3", "--every", "10")
	if code, out, _ := runCLI(t, frames, "check", "-"); code != 0 {
		t.Fatalf("render output failed check: %s", out)
	}
}

func TestCastDeterministic(t *testing.T) {
	args := []string{"cast", "fire", "--w", "10", "--h", "3", "--seconds", "0.5"}
	_, a, _ := runCLI(t, "", args...)
	_, b, _ := runCLI(t, "", args...)
	if a != b {
		t.Fatal("cast output differs between runs")
	}
	lines := strings.Split(strings.TrimSpace(a), "\n")
	if !strings.Contains(lines[0], `"version":3`) || len(lines) < 3 {
		t.Fatalf("bad cast: %s", a)
	}
	var ev []any
	if err := json.Unmarshal([]byte(lines[1]), &ev); err != nil || len(ev) != 3 || ev[1] != "o" {
		t.Fatalf("bad event %s", lines[1])
	}
}

func TestRenderFrameCapsBeforeAllocating(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"render", "fire", "--every", "1", "--seconds", "1000000000", "--json"}, "--seconds"},
		{[]string{"render", "fire", "--every", "1", "--seconds", "NaN", "--json"}, "finite"},
		{[]string{"render", "fire", "--every", "1", "--seconds", "Inf", "--json"}, "finite"},
		{[]string{"render", "fire", "--every", "1", "--seconds", "600", "--fps", "240", "--json"}, "limit is 5000"},
		{[]string{"render", "fire", "--every", "9223372036854775807", "--seconds", "-1", "--json"}, "finite"},
		{[]string{"render", "fire", "--at", "NaN", "--json"}, "finite"},
		{[]string{"render", "fire", "--at", "1e300", "--json"}, "within"},
		{[]string{"render", "fire", "--frame", "9223372036854775807", "--json"}, "at most"},
		{[]string{"cast", "fire", "--seconds", "NaN"}, "finite"},
		{[]string{"cast", "fire", "--seconds", "1e12"}, "finite"},
	}
	for _, c := range cases {
		start := time.Now()
		code, out, errs := runCLI(t, "", c.args...)
		out += errs
		if code != exitUsage {
			t.Errorf("%v: exit %d, want 2: %s", c.args, code, out)
		}
		if !strings.Contains(out, c.want) || !strings.Contains(out, "hint") {
			t.Errorf("%v: output lacks %q: %s", c.args, c.want, out)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%v: took %v; caps must apply before work", c.args, d)
		}
	}
	// Huge step on a finite effect is fine: first and last frame only.
	code, out, _ := runCLI(t, "", "render", "reveal", "--text", "X", "--w", "3", "--h", "1", "--every", "9223372036854775807")
	if code != 0 || strings.Count(out, "--- frame") != 2 {
		t.Fatalf("huge --every on finite effect: exit %d\n%s", code, out)
	}
}

func TestRenderReviewRegressions(t *testing.T) {
	usage := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"emoji in --text", "", []string{"render", "reveal", "--text", "hi \U0001F525", "--json"}, "U+1F525"},
		{"wide rune in --text", "", []string{"render", "reveal", "--text", "漢", "--json"}, "asciifx check"},
		{"emoji via --file -", "ok\n★\n", []string{"render", "decrypt", "--file", "-", "--json"}, "line 2 col 1"},
		{"bad profile with plain", "", []string{"render", "reveal", "--profile", "bogus", "--json"}, "valid profiles"},
		{"bad profile with luma", "", []string{"render", "reveal", "--profile", "bogus", "--format", "luma", "--json"}, "valid profiles"},
		{"--every 0", "", []string{"render", "fire", "--every", "0", "--json"}, "step of 1 or more"},
		{"--every 0 conflicts like any selector", "", []string{"render", "fire", "--every", "0", "--frame", "1", "--json"}, "mutually exclusive"},
		{"out of range echoes typed index", "", []string{"render", "reveal", "--frames", "0,-1,-100", "--json"}, "frame -100 out of range"},
	}
	for _, c := range usage {
		code, out, _ := runCLI(t, c.stdin, c.args...)
		if code != exitUsage || !strings.Contains(out, c.want) {
			t.Errorf("%s: exit %d, want 2 with %q in:\n%s", c.name, code, c.want, out)
		}
	}

	// play shares the content check; it has no --json, so the error is on stderr.
	if code, _, errs := runCLI(t, "", "play", "reveal", "--text", "\u00b7"); code != exitUsage || !strings.Contains(errs, "not single-width") {
		t.Errorf("play with unsafe content: exit %d, stderr %q", code, errs)
	}

	ok := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"tabs and CRLF content still render", "A\tB\r\nC\r\n", []string{"render", "reveal", "--file", "-", "--frame", "-1"}, "C"},
		{"valid profile with plain", "", []string{"render", "reveal", "--text", "OK", "--profile", "16", "--frame", "-1"}, "OK"},
		{"--every 1", "", []string{"render", "reveal", "--text", "OK", "--w", "4", "--h", "1", "--every", "1"}, "--- frame 48"},
	}
	for _, c := range ok {
		code, out, errs := runCLI(t, c.stdin, c.args...)
		if code != 0 || !strings.Contains(out, c.want) {
			t.Errorf("%s: exit %d, want 0 with %q\nstdout:\n%s\nstderr:\n%s", c.name, code, c.want, out, errs)
		}
	}
}

// TestExplicitProfileIgnoresNoColor pins the CLI half of the stale-dither fix.
// NO_COLOR sets the detected profile to none, and the dither that goes with a
// palette used to be discarded with it, so `--profile 256` rendered undithered
// under NO_COLOR and dithered without it — the same flags, different bytes.
func TestExplicitProfileIgnoresNoColor(t *testing.T) {
	for _, k := range []string{"NO_COLOR", "ASCIIFX_COLOR", "ASCIIFX_DITHER"} {
		t.Setenv(k, "")
	}
	args := []string{"render", "fire", "--w", "24", "--h", "6", "--frame", "45", "--format", "ansi", "--profile", "256", "--seed", "1"}
	code, want, stderr := runCLI(t, "", args...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(want, "38;5;") {
		t.Fatal("--profile 256 did not emit 256-colour output")
	}

	t.Setenv("NO_COLOR", "1")
	if _, got, _ := runCLI(t, "", args...); got != want {
		t.Fatal("NO_COLOR changed the output of an explicit --profile 256")
	}
	// ASCIIFX_DITHER is a preference about palettes, not about colour, so it
	// has to survive NO_COLOR as well.
	t.Setenv("ASCIIFX_DITHER", "bayer8")
	if _, got, _ := runCLI(t, "", args...); got != want {
		t.Fatal("NO_COLOR suppressed an explicit ASCIIFX_DITHER=bayer8")
	}
	// ...and turning the dither off explicitly still works under NO_COLOR.
	t.Setenv("ASCIIFX_DITHER", "")
	_, off, _ := runCLI(t, "", append(append([]string{}, args...), "--dither", "none")...)
	if off == want {
		t.Fatal("--dither none had no effect under NO_COLOR")
	}
}
