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
