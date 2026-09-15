package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update rewrites the golden files instead of comparing against them:
//
//	go test ./cmd/asciifx -run Golden -update
//
// Regenerate only when a change to an effect is intended, and read the diff.
var update = flag.Bool("update", false, "rewrite testdata/*.golden")

// goldenCase is one deterministic CLI invocation whose stdout is pinned.
// dense marks a case whose output is a full-screen field rather than a
// banner, so the shape check can insist on a real spread of glyphs. want is
// an optional substring the output must contain.
type goldenCase struct {
	name  string
	args  []string
	dense bool
	want  string
}

// goldenCases deliberately covers each glyph class (ascii, box, block,
// halfblock, braille), each effect kind (ambient, transition, spinner) and
// each text format, so a regression in one renderer cannot hide behind
// another. Every case fixes --seed, --w, --h and the frame, because runs are
// pure functions of exactly those.
var goldenCases = []goldenCase{
	{name: "fire-luma", dense: true, args: []string{"render", "fire", "--w", "48", "--h", "12", "--frame", "45", "--format", "luma", "--seed", "1"}},
	{name: "plasma-luma", dense: true, args: []string{"render", "plasma", "--w", "40", "--h", "10", "--frame", "20", "--format", "luma", "--seed", "3"}},
	{name: "aurora-luma", dense: true, args: []string{"render", "aurora", "--w", "40", "--h", "10", "--frame", "60", "--format", "luma", "--seed", "5"}},
	{name: "starfield-luma", dense: true, args: []string{"render", "starfield", "--w", "40", "--h", "12", "--frame", "90", "--format", "luma", "--seed", "2"}},
	{name: "matrix-plain", dense: true, args: []string{"render", "matrix", "--w", "40", "--h", "10", "--frame", "30", "--format", "plain", "--seed", "7"}},
	{name: "dna-plain", dense: true, want: "⠨", args: []string{"render", "dna", "--w", "32", "--h", "10", "--frame", "24", "--format", "plain", "--seed", "4"}},
	{name: "pipes-luma", dense: true, args: []string{"render", "pipes", "--w", "40", "--h", "10", "--frame", "120", "--format", "luma", "--seed", "6"}},
	{name: "reveal-plain", want: "███", args: []string{"render", "reveal", "--banner", "ACME", "--w", "40", "--h", "10", "--frame", "24", "--format", "plain", "--seed", "1"}},
	{name: "reveal-final-plain", want: "███", args: []string{"render", "reveal", "--banner", "ACME", "--w", "40", "--h", "10", "--frame", "-1", "--format", "plain", "--seed", "1"}},
	{name: "decrypt-plain", dense: true, args: []string{"render", "decrypt", "--text", "ACCESS GRANTED", "--w", "32", "--h", "6", "--frame", "40", "--format", "plain", "--seed", "1"}},
	{name: "decrypt-final-plain", want: "ACCESS GRANTED", args: []string{"render", "decrypt", "--text", "ACCESS GRANTED", "--w", "32", "--h", "6", "--frame", "-1", "--format", "plain", "--seed", "1"}},
	{name: "rainbow-plain", want: "HELLO", args: []string{"render", "rainbow", "--text", "HELLO", "--w", "24", "--h", "4", "--frame", "20", "--format", "plain", "--seed", "1"}},
	{name: "spinner-plain", args: []string{"render", "spinner", "--w", "8", "--h", "2", "--frame", "12", "--format", "plain", "--seed", "1", "-p", "style=dots"}},
	{name: "fire-ansi-256", want: "38;5;", args: []string{"render", "fire", "--w", "24", "--h", "6", "--frame", "45", "--format", "ansi", "--profile", "256", "--seed", "1"}},
	{name: "fire-ansi-16", want: "\x1b[0;", args: []string{"render", "fire", "--w", "24", "--h", "6", "--frame", "45", "--format", "ansi", "--profile", "16", "--seed", "1"}},
	{name: "reveal-pattern-combinator", want: "███", args: []string{"render", "reveal", "--banner", "HI", "--w", "24", "--h", "8", "--frame", "18", "--format", "plain", "--seed", "1", "-p", "pattern=min(invert(center),wave)"}},
	{name: "chain-mid", want: "███", args: []string{"render", "reveal", "--banner", "HI", "--w", "24", "--h", "8", "--frame", "30", "--format", "plain", "--seed", "1", "--then", "fire", "--for", "1s"}},
	{name: "chain-final", want: "▀", args: []string{"render", "reveal", "--banner", "HI", "--w", "24", "--h", "8", "--frame", "-1", "--format", "plain", "--seed", "1", "--then", "fire", "--for", "1s"}},
	{name: "chain-ansi-256", want: "38;5;", args: []string{"render", "reveal", "--banner", "HI", "--w", "20", "--h", "6", "--for", "500ms", "--then", "shine", "--for", "500ms", "--frame", "-1", "--format", "ansi", "--profile", "256", "--seed", "1"}},
	{name: "check-bad-art", want: "exit 3", args: []string{"check", "-"}},
}

func goldenPath(name string) string { return filepath.Join("testdata", name+".golden") }

func TestGoldenFrames(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			stdin := ""
			if tc.name == "check-bad-art" {
				// A frame with an emoji, a tab and a ragged line: every
				// finding kind the linter produces, in a stable order.
				stdin = "hello \U0001F600\tworld\nok\n"
			}
			code, stdout, stderr := runCLI(t, stdin, tc.args...)
			// The exit code is part of the contract, so pin it too.
			got := stdout
			if code != 0 {
				got = fmt.Sprintf("exit %d\n%s", code, stdout)
			}
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath(tc.name), []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath(tc.name))
			if err != nil {
				t.Fatalf("read golden: %v\nrun `go test ./cmd/asciifx -run Golden -update` to create it", err)
			}
			if got != string(want) {
				t.Errorf("%s changed.\n--- want ---\n%s\n--- got ---\n%s\nstderr: %s", tc.name, string(want), got, stderr)
			}
		})
	}
}

// TestGoldenFramesAreReproducible is the property the golden files rest on:
// two runs of the same invocation are byte-identical. If this fails the
// golden files are meaningless, so it fails loudly and names the case.
func TestGoldenFramesAreReproducible(t *testing.T) {
	for _, tc := range goldenCases {
		stdin := ""
		if tc.name == "check-bad-art" {
			stdin = "hello \U0001F600\tworld\nok\n"
		}
		_, first, _ := runCLI(t, stdin, tc.args...)
		for i := 0; i < 3; i++ {
			if _, again, _ := runCLI(t, stdin, tc.args...); again != first {
				t.Fatalf("%s: run %d differs from the first", tc.name, i+2)
			}
		}
	}
}

// TestGoldenFramesHaveShape guards against the failure mode where every
// golden file is regenerated from an effect that silently renders nothing.
func TestGoldenFramesHaveShape(t *testing.T) {
	for _, tc := range goldenCases {
		stdin := ""
		if tc.name == "check-bad-art" {
			stdin = "hello \U0001F600\tworld\nok\n"
		}
		_, stdout, stderr := runCLI(t, stdin, tc.args...)
		got := stdout
		if code, _, _ := runCLI(t, stdin, tc.args...); code != 0 {
			got = fmt.Sprintf("exit %d\n%s", code, stdout)
		}
		if strings.TrimSpace(got) == "" {
			t.Errorf("%s: rendered nothing (stderr: %s)", tc.name, stderr)
			continue
		}
		if tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: output lacks %q", tc.name, tc.want)
		}
		if !tc.dense {
			continue
		}
		distinct := map[rune]bool{}
		for _, r := range got {
			distinct[r] = true
		}
		if len(distinct) < 4 {
			t.Errorf("%s: output has only %d distinct runes; looks degenerate", tc.name, len(distinct))
		}
	}
}
