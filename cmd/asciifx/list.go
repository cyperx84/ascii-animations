package main

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
)

func cmdList(e *env, args []string) error {
	c := findCommand("list")
	fs := newFlags(c.name)
	asJSON := fs.Bool("json", false, "emit JSON")
	kind := fs.String("kind", "", "filter: ambient, transition or spinner")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return usageErr("usage: "+c.usage, "unexpected arguments: %s", strings.Join(pos, " "))
	}
	switch fx.Kind(*kind) {
	case "", fx.Ambient, fx.Transition, fx.Spinner:
	default:
		return usageErr("kinds: ambient, transition, spinner", "unknown kind %q", *kind)
	}
	specs := []*fx.Spec{}
	for _, s := range fx.All() {
		if *kind == "" || s.Kind == fx.Kind(*kind) {
			specs = append(specs, s)
		}
	}
	if *asJSON {
		return writeJSON(e.stdout, specs)
	}
	tw := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tKIND\tLENGTH\tGLYPHS\tDESCRIPTION")
	for _, s := range specs {
		length := "loop"
		if s.Duration > 0 {
			length = fmt.Sprintf("%gs", s.Duration)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Kind, length, strings.Join(s.Glyphs, ","), truncate(firstSentence(s.Description), 60))
	}
	tw.Flush()
	if len(specs) == 0 {
		fmt.Fprintln(e.stderr, "no effects match")
	}
	return nil
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n-1]), " ") + "…"
}

type specInfo struct {
	*fx.Spec
	// Frames is the frame count at the default fps with default params; 0
	// when the effect loops.
	Frames int `json:"frames"`
}

func cmdInfo(e *env, args []string) error {
	c := findCommand("info")
	fs := newFlags(c.name)
	asJSON := fs.Bool("json", false, "emit JSON")
	pos, err := parse(e, c, fs, args)
	if err != nil {
		return err
	}
	name, err := oneEffect(c, pos)
	if err != nil {
		return err
	}
	spec, err := lookupSpec(name)
	if err != nil {
		return err
	}
	frames := 0
	if r, err := fx.NewRun(spec, fx.Options{Seed: 1}); err == nil {
		frames = r.Frames()
	}
	if *asJSON {
		return writeJSON(e.stdout, specInfo{Spec: spec, Frames: frames})
	}
	w := e.stdout
	fmt.Fprintf(w, "%s (%s) — %s\n\n", spec.Name, spec.Kind, spec.Title)
	fmt.Fprintf(w, "%s\n\n", spec.Description)
	if spec.Duration > 0 {
		fmt.Fprintf(w, "length:  %gs, %d frames at %d fps\n", spec.Duration, frames, spec.FPS)
	} else {
		fmt.Fprintf(w, "length:  loops forever, %d fps\n", spec.FPS)
	}
	fmt.Fprintf(w, "size:    default %dx%d, min %dx%d\n", spec.DefW, spec.DefH, spec.MinW, spec.MinH)
	fmt.Fprintf(w, "glyphs:  %s\n", strings.Join(spec.Glyphs, ", "))
	if spec.Content {
		fmt.Fprintln(w, "content: yes (--text, --banner or --file)")
	}
	if len(spec.Tags) > 0 {
		fmt.Fprintf(w, "tags:    %s\n", strings.Join(spec.Tags, ", "))
	}
	fmt.Fprintln(w)
	if len(spec.Params) == 0 {
		fmt.Fprintln(w, "params:  none")
	} else {
		fmt.Fprintln(w, "params (-p key=value):")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, p := range spec.Params {
			constraint := ""
			switch {
			case p.Min != nil && p.Max != nil:
				constraint = fmt.Sprintf("[%g, %g]", *p.Min, *p.Max)
			case p.Type == fx.Enum && len(p.Options) <= 4:
				constraint = strings.Join(p.Options, "|")
			case len(p.Options) > 0:
				constraint = fmt.Sprintf("%d options", len(p.Options))
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\tdefault %s\t%s\n", p.Name, p.Type, constraint, quoteDefault(p.Default), p.Doc)
		}
		tw.Flush()
		for _, p := range spec.Params {
			if len(p.Options) > 4 || (len(p.Options) > 0 && p.Type != fx.Enum) {
				fmt.Fprintf(w, "  %s options: %s\n", p.Name, strings.Join(p.Options, ", "))
			}
		}
	}
	if spec.Example != "" {
		fmt.Fprintf(w, "\nexample:\n  %s\n", spec.Example)
	}
	fmt.Fprintf(w, "\npreview as text:\n  asciifx render %s --format plain\n", spec.Name)
	return nil
}

func quoteDefault(s string) string {
	if s == "" || strings.ContainsAny(s, " ,") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
