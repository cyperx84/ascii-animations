# Dogfood findings

What broke, chafed or was missing when this repo's own demo programs were written against
its public API. Every entry names the file and line that caused it, so the next person can
start at the code rather than at the story.

Status: round 1 (engine fixes) and round 2 (five demos) landed on
`feat/fxdash-dogfood`. Findings 2, 3, 4, 6 and 7 are fixed; 1 and 5 are
documented, not fixed.

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

### 5. A lipgloss Layer composed onto a Canvas ignores its own position

This is the one that cost the most time, because nothing fails: the layout
just comes out wrong.

`lipgloss.Layer.Draw` (`layer.go:153` in lipgloss v2.0.6) is

	func (l *Layer) Draw(scr uv.Screen, area uv.Rectangle) {
		uv.NewStyledString(l.content).Draw(scr, area)
	}

It never reads its own X and Y — those are `Compositor`'s business.
`Canvas.Compose` hands every drawable the whole canvas, so a pile of
positioned Layers composed onto a Canvas all paint at the origin, each one
clearing the canvas as it goes, and only the last survives. In
`examples/05-dashboard` that looked like "the ambient panel never renders";
the real cause was the node table, composed last, wiping everything.

The README warned about mixing the two paths but gave the wrong reason. The
reason is not that a Layer is opaque — it is that a Layer has no position
until something gives it one. `teafx.At` is that something, and it pins any
drawable, not only an effect:

	l := lipgloss.NewLayer(s)
	canvas.Compose(teafx.At(l, uv.Rect(x, y, l.Width(), l.Height())))

Both demos that use a Canvas now pin everything, and
`examples/04-panels` has a test that the two paths paint identical glyphs, so
a regression in either one shows up as a diff.

Not fixed in code — it is upstream behaviour and the pin is the right answer.
Fixed in the README and demonstrated in two demos.

### 6. A Caps could not be rendered with from outside `term`

`Caps.dither()` was unexported, and it holds a real rule: Detect's answer is
re-resolved against the current profile, because a caller may have replaced
the profile that answer was made for. A consumer holding a `Caps` therefore
could not encode a buffer the way that Caps says to — `Caps.Dither` alone is
the wrong answer often enough to matter.

Fixed: `term.ANSICaps(b, caps)`, used by `teafx.Model.View` itself.

### 7. Alt screen moved in Bubble Tea v2 and nothing here said so

`tea.WithAltScreen()` is gone; it is `View.AltScreen` now. Every asciifx doc
comment that mentions a Bubble Tea program predates that. Only
`examples/05-dashboard` needs the alternate screen, and it now shows the
current shape.

## What the demos cover

| Demo | Surface it exercises |
|---|---|
| `examples/01-spinner` | the drop-in spinner, `WithLabel`, `WithPalette`, `WithShimmer` |
| `examples/02-intro` | `teafx.Model`, `Done`, `Restart`, `SetCaps` |
| `examples/03-chain` | `fx.Compose`, `Step.Filter`, `fx.SelNot`/`fx.SelInk`, `teafx.FromRun`, `Loop` |
| `examples/04-panels` | both composition paths, `teafx.At`, `Canvas.Compose`, and a test that they agree |
| `examples/05-dashboard` | `teafx.Content`, `Snapshot`, several models at once, `SetSize`, `fx.Hash01` |

Every demo has a headless test, and three of them lint their own frames with
`asciifx/lint` — which is the loop the README sells, closed on the project
itself.
