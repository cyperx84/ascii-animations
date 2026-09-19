# Dogfood findings

What broke, chafed or was missing when this repo's own demo programs were written against
its public API. Every entry names the file and line that caused it, so the next person can
start at the code rather than at the story.

Status: rounds 1-5 landed on `feat/fxdash-dogfood` — engine fixes, five demos,
an SVG exporter, a spinner that honours the terminal, and the review's own
findings. Everything except 1 and 5 is fixed; those two are
documented, not fixed. "What to do next" at the end is the ranked list.

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

### 8. There was no way to show an animation to anyone not at a terminal

A cast needs a player, a GIF needs a rasteriser and a font, and `vhs` — the
obvious tool — is broken on this machine: 0.12.0 with ffmpeg 9.0.1 prints
"Creating out.gif...", exits 0 and writes nothing, in a sandbox and in a real
TTY alike. So the README of an animation library showed no animation.

Fixed by `asciifx svg` and `asciifx/svg`: every frame in one document, one CSS
keyframe per frame, no script, no font, no external reference, so it plays
inside an `<img>`. Two details earn their code:

- Block glyphs are drawn as rectangles, not text. A terminal scales `█` and
  `▀` to fill the cell; a font does not, so text leaves a seam on every row.
- `--quantize` rounds colours before the runs are cut, which merges
  neighbouring cells a viewer cannot tell apart. A gradient is where the bytes
  go.

### 9. The spinner kept spinning under reduced motion

Found by running `ASCIIFX_REDUCED_MOTION=1 go run ./examples/02-intro` in a
real terminal rather than in a test: the intro froze on its last frame exactly
as intended, and the spinner underneath it kept turning. Two reads a second
apart, from the same pane:

	⠋ Warming up the pixels
	⠸ Warming up the pixels

`teafx.Model` honoured the contract; `asciifx/spinner` had no idea it existed.
A spinner is the one widget that animates for as long as the program runs, so
it is the worst one to miss.

Fixed: `spinner.WithCaps(term.Caps)`, opt-in like the package's other three
extras, so upstream's call shape still behaves exactly as upstream's does. A
Caps with `Animate` false stops the chain rather than hiding it, so a program
under CI does no per-frame work at all.

The lesson is about method, not about spinners: this is the bug the five
headless tests could not have caught, because nothing in a test has an opinion
about the terminal. Run the thing.

### 10. What the review found that the demos did not

A high-effort review of the whole diff, plus the tests written to confirm each
finding. Every one is fixed on this branch.

| | |
|---|---|
| `teafx.go` `SetSize` | a still model resized to frame 0, and an ambient effect's frame 0 is often blank — `matrix` renders nothing at all there. Under `ASCIIFX_REDUCED_MOTION`, resizing `examples/05-dashboard` blanked the panel for good. `Restart` already guarded this; `SetSize` did not. |
| `teafx.go` `SetCaps` | bumped the tick tag unconditionally, so calling it mid-chain with `Animate` true killed the chain and returned no command to restart it: a silent permanent freeze for anyone who sets caps from the first `WindowSizeMsg`. It now invalidates only when it is stopping motion. |
| `05-dashboard` refresh | every manual refresh started a second poll timer without retiring the first, so five keypresses left six timers polling at six times the rate. Tagged with the generation, the same trick `teafx.TickMsg` uses. |
| `svg.Options` strings | `Background`, `Foreground` and `FontFamily` were interpolated into a `<style>` element unescaped. `--foreground '</style><script>…'` produced a document carrying a script. They are now validated and rejected, because escaping is not enough for text that will be read as CSS. |
| `svg` font size | emitted at `%.0f` while the cell grid was derived from the unrounded value, so `--font-size 13.5` drew 14px glyphs on an 8.1px grid and every row drifted. |
| `svg` dim cells | `opacity` on a `<tspan>`, which the spec leaves renderers free to ignore. `fill-opacity` is the attribute that applies. |

And one the review did not find, which the test written for its finding did:
**`examples/05-dashboard` matched the space key as `" "`, but Bubble Tea v2
names it `"space"`** — the manual refresh key had never worked, and only the
6-second timer made it look like it did.

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

## What to do next

Ranked by what a consumer hits first, not by what is interesting to build.

1. **Make `SetCaps` hard to forget.** It is the whole terminal-safety contract
   and it is opt-in, which means every program that does not know about it is
   the program that needed it. Options, roughly in order of how much they
   change: document it at the top of `teafx`; add `teafx.NewDetected` that
   folds `term.Detect(os.Stdout)` in; or make `Model` consult a package-level
   default set once at startup. Do not make `New` read the environment
   silently — the CLI's own habit of detecting explicitly is the right one.
2. **`fx.Compose` deserves a param-scoped chain builder.** `Step` is fine in
   Go, but the CLI's `--then`/`--for`/`--filter` scoping has no Go equivalent,
   so the two ways of saying the same thing look nothing alike. A
   `fx.Chain().Then("fire").For(1.2).Filter(sel)` builder would close that.
3. **Resize deserves a transition-aware answer.** `examples/05-dashboard`
   drops a transition mid-flight on resize, because the run is sized to the
   panel it started in. `Run.Resize` resumes pure transitions exactly, so a
   `Model.SetSize` that kept a transition alive is possible — it just needs
   the content re-snapshotted at the new size, which only the caller can do.
   An explicit `Model.Reseed(Content)` would make it expressible.
4. **The SVG exporter could carry the terminal's own colours.** It quantises
   but never dithers, so a 16- or 256-colour profile exports as flat bands
   where the terminal would stipple. `term.ANSIWith` already knows how; the
   encoder would need the same `tint.Dither` path.
5. **`showcase` and `pkg/` still share no code with the engine** (finding 1).
   Either port the showcase onto `asciifx` — it is the best remaining
   dogfood, and it would delete most of `pkg/` — or retire it now that
   `examples/` covers the same ground with less code.
