# Dogfood findings

What broke, chafed or was missing when this repo's own demo programs were written against
its public API. Every entry names the file and line that caused it, so the next person can
start at the code rather than at the story.

Status: round 1 (engine fixes) in progress on `feat/fxdash-dogfood`.

## Method

Five demo programs, simple to hard, each written as an outside consumer would write it:
only the exported surface, no reaching into internals. `examples/01-spinner` through
`examples/05-dashboard`. Friction found while writing them is logged here; the fixes land
before the demos so the demos show the API as it should be, not as it was.

## Findings

### 1. No program in the repo used the engine

`cmd/showcase/main.go:8` and `pkg/ui/ui.go:10-16` import `pkg/*` only. The showcase TUI
predates `asciifx/` and shares no code with it, so before this work nothing in the repo
exercised the embedding surface the README sells to TUI builders. The two `examples/`
programs are ~60 lines each and cover `Model.View` and a hand-driven `fx.Run`; they touch
neither `teafx.At`, `teafx.Content`, `fx.Compose`, `fx.Filter`, `Restart` nor `Caps`.

Not fixed by this work: `showcase` and `pkg/` are left alone. Logged because it explains
why the gaps below survived to now.

### 2. `teafx.Model` ignores the terminal-safety contract

`asciifx/teafx/teafx.go:92` ticks at `run.FPS()` unconditionally and
`asciifx/teafx/teafx.go:141` encodes every frame as `term.TrueColor`. Nothing consults
`term.Caps`, so a Bubble Tea consumer gets none of the guarantees the README lists under
"Terminal safety":

- no 15 fps cap over SSH, tmux or screen
- no `ASCIIFX_REDUCED_MOTION` / `CI` / `TERM=dumb` static frame
- no `ASCIIFX_FPS` override

Bubble Tea v2's renderer downsamples colour, so the profile half is mostly covered by the
framework; the motion half is not covered by anything.

Fix: `Model.SetCaps(term.Caps)`, mirroring `term.Play` — `term.StaticTick` picks the same
frame `Play` shows, and the FPS cap uses the same `Caps.FPS` field.

### 3. A composed chain cannot be a Bubble Tea component

`teafx.New` (`asciifx/teafx/teafx.go:61`) takes an effect *name* and calls `fx.Lookup`.
`fx.Compose` (`asciifx/fx/compose.go:150`) returns a synthetic `*fx.Spec` that is never
registered. So the chain the CLI builds from `--then` had no way into a `Model` except
`fx.Register`, which panics on a duplicate name (`asciifx/fx/registry.go:138`) and would
publish a private chain into `asciifx list` for every consumer of the process.

The workaround was to drive `fx.NewRun` by hand with a private `tea.Tick` — re-implementing
`Model.Update`'s tick chain, tag invalidation included, in every program that wants a chain.

Fix: `teafx.FromRun(*fx.Run) Model`, which is what `Model.Run()` already implies exists.

### 4. The ASCII-art linter is not importable

`cmd/asciifx/check.go:1` is `package main`. The linter is the first of the two things the
README calls hard to get elsewhere, and a consumer who wants it in their own test — "my
banner is still single-width after I edited it" — cannot call it. Shelling out to the
binary is the only path, which means their test depends on a build of this module being on
`PATH`.

Fix: move the rules to `asciifx/lint`; `cmd/asciifx/check.go` keeps the flags, the
formatting and the exit code. `cmd/asciifx/testdata/check-bad-art.golden` pins the CLI
output byte for byte, so it is the regression guard for the move.

## Round 2 findings

Filled in as the demos are written.
