package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/ascii-animations/asciifx/ease"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

type catalogJSON struct {
	Name     string              `json:"name"`
	Version  string              `json:"version"`
	Install  string              `json:"install"`
	Effects  []specInfo          `json:"effects"`
	Palettes map[string][]string `json:"palettes"`
	Easings  []string            `json:"easings"`
	Patterns []string            `json:"patterns"`
	// PatternSyntax is the grammar for pattern expressions, which go beyond
	// the bare names in Patterns: combinators nest arbitrarily.
	PatternSyntax string            `json:"pattern_syntax"`
	Dithers       []string          `json:"dithers"`
	Directions    []string          `json:"directions"`
	BannerFonts   []string          `json:"banner_fonts"`
	ExitCodes     map[string]string `json:"exit_codes"`
}

const installCmd = "go install github.com/cyperx84/ascii-animations/cmd/asciifx@latest"

func buildCatalog() catalogJSON {
	c := catalogJSON{
		Name:          "asciifx",
		Version:       versionString(),
		Install:       installCmd,
		Effects:       []specInfo{},
		Palettes:      map[string][]string{},
		Easings:       ease.Names(),
		Patterns:      fx.PatternNames(),
		PatternSyntax: fx.PatternGrammar(),
		Dithers:       tint.DitherNames(),
		Directions:    tint.Directions,
		BannerFonts:   bannerFonts(),
		ExitCodes:     map[string]string{"0": "ok", "1": "runtime error", "2": "usage error (unknown effect, param or flag)", "3": "check found problems"},
	}
	for _, s := range fx.All() {
		frames := 0
		if r, err := fx.NewRun(s, fx.Options{Seed: 1}); err == nil {
			frames = r.Frames()
		}
		c.Effects = append(c.Effects, specInfo{Spec: s, Frames: frames})
	}
	for _, name := range tint.PaletteNames() {
		g := tint.Palettes[name]
		stops := make([]string, len(g))
		for i, col := range g {
			stops[i] = col.String()
		}
		c.Palettes[name] = stops
	}
	return c
}

func cmdCatalog(e *env, args []string) error {
	c := findCommand("catalog")
	fs := newFlags(c.name)
	out := fs.String("out", "", "directory for catalog.json and llms.txt (default: catalog.json to stdout)")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return usageErr("usage: "+c.usage, "unexpected arguments: %s", strings.Join(pos, " "))
	}
	cat := buildCatalog()
	if *out == "" {
		return writeJSON(e.stdout, cat)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return runtimeErr(err, "check --out is a writable directory")
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, cat); err != nil {
		return err
	}
	jsonPath := filepath.Join(*out, "catalog.json")
	llmsPath := filepath.Join(*out, "llms.txt")
	if err := os.WriteFile(jsonPath, buf.Bytes(), 0o644); err != nil {
		return runtimeErr(err, "check --out is a writable directory")
	}
	if err := os.WriteFile(llmsPath, []byte(llmsTxt(cat)), 0o644); err != nil {
		return runtimeErr(err, "check --out is a writable directory")
	}
	fmt.Fprintf(e.stdout, "wrote %s (%d effects)\nwrote %s\n", jsonPath, len(cat.Effects), llmsPath)
	return nil
}

func llmsTxt(cat catalogJSON) string {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	p("# asciifx\n\n")
	p("> Deterministic terminal animation toolkit for Go: a catalog of procedural effects (ambient backgrounds, text transitions, spinners) with typed JSON params, a headless renderer that prints any frame as text, an ASCII-art width linter, and a terminal player that always restores the terminal.\n\n")
	p("Every run is a pure function of (effect, params, size, seed, tick), so an agent can inspect frame N as text instead of watching motion. Version %s.\n\n", cat.Version)
	p("## Install\n\n```sh\n%s\n```\n\n", installCmd)
	p("## Commands\n\n")
	p("Exit codes: 0 ok, 1 runtime error, 2 usage error, 3 check found problems. With `--json`, errors print `{\"error\": \"...\", \"hint\": \"...\"}` to stdout.\n\n")
	for _, c := range commands {
		if c.name == "help" || c.name == "version" {
			continue
		}
		p("- `%s` — %s Example: `%s`\n", c.usage, c.summary, firstExample(c))
	}
	p("\nShared flags: `-p key=value` (repeatable, scoped to the effect named last), `--then EFFECT` to chain another effect after the previous one, `--for 2s` to give a looping effect a duration (required before anything can follow it), `--seed N`, `--w/--h`, `--fps`, `--profile`, `--dither`, and for transitions `--text \"A\\nB\"`, `--banner TEXT --font block|slim|mini`, `--file path|-`. Chained output reports `steps` and params keyed `<step>.<name>`.\n\n")
	p("## Effects\n\n")
	for _, s := range cat.Effects {
		p("- %s (%s): %s — `%s`\n", s.Name, s.Kind, firstSentence(s.Description), s.Example)
	}
	p("\nParams, types, ranges and defaults: `asciifx info <effect> --json`.\n\n")
	p("## Palettes\n\n%s. Any palette param also takes hex stops: `-p palette=\"#ff0000,#0000ff\"`.\n\n", strings.Join(tint.PaletteNames(), ", "))
	p("## Patterns\n\n%s\n\nA pattern is a spatial ordering in [0,1] deciding where an effect starts and ends. Params taking a pattern accept an expression: %s. For example `-p pattern=\"min(invert(center),wave)\"`.\n\n", strings.Join(cat.Patterns, ", "), cat.PatternSyntax)
	p("## Dithers\n\n%s. `--dither` stipples gradients in the 16- and 256-colour profiles; it is ordered per cell, so still frames never shimmer.\n\n", strings.Join(cat.Dithers, ", "))
	p("## Easings\n\n%s\n\n", strings.Join(cat.Easings, ", "))
	if len(cat.BannerFonts) > 0 {
		p("## Banner fonts\n\n%s\n\n", strings.Join(cat.BannerFonts, ", "))
	}
	p("## Embedding in Go\n\n")
	p("Headless frame:\n\n```go\nimport (\n\t\"github.com/cyperx84/ascii-animations/asciifx/fx\"\n\t_ \"github.com/cyperx84/ascii-animations/asciifx/effects\"\n)\n\nbuf, _, err := fx.Render(\"fire\", fx.Options{W: 60, H: 16, Seed: 1}, 30)\nfmt.Println(buf.Plain())\n```\n\n")
	p("Play in the terminal (static frame when not a TTY, CI, or reduced motion):\n\n```go\nspec, _ := fx.Lookup(\"reveal\")\nrun, _ := fx.NewRun(spec, fx.Options{Content: fx.Text(\"HELLO\", tint.None), Params: map[string]string{\"palette\": \"synthwave\"}})\nerr := term.Play(ctx, run, term.PlayOptions{Caps: term.Detect(os.Stdout), Inline: true})\n```\n\n")
	p("Bubble Tea v2 component:\n\n```go\nimport \"github.com/cyperx84/ascii-animations/asciifx/teafx\"\n\nm, _ := teafx.New(\"fire\", fx.Options{W: 40, H: 10})\n// Init() starts ticking; forward msgs to m.Update; embed m.View() in your view.\n```\n")
	return b.String()
}

func firstExample(c *command) string {
	for _, l := range strings.Split(c.details, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "asciifx ") {
			return l
		}
	}
	return "asciifx " + c.name
}
