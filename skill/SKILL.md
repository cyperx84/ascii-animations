---
name: asciifx
description: Add beautiful, terminal-safe ASCII animations to CLI and TUI programs with the asciifx toolkit — intro banners, text reveals, spinners, ambient backgrounds (fire, matrix, plasma, aurora). Use when adding or polishing any terminal animation, splash screen, loading indicator or text effect, when an agent needs to SEE an animation frame as text, or when linting ASCII art for alignment and unsafe glyphs. Triggers: "ascii animation", "terminal animation", "splash screen", "intro banner", "spinner", "loading animation", "text effect", "matrix rain", "tui polish", "asciifx".
---

# asciifx

asciifx is a deterministic terminal animation engine plus a CLI built for agents. You cannot watch motion, so the workflow is: **pick an effect, render frames as text, inspect, tune params, embed**.

Two parts of it are worth reaching for even when nothing animates:

- `asciifx check` lints ASCII art for width problems. Run it on any art you write or generate, before committing it.
- `asciifx render --format luma` turns a colour-only effect into a brightness map you can read.

Install: `go install github.com/cyperx84/ascii-animations/cmd/asciifx@latest`

## 1. Decide whether to animate at all

Reach for an effect only when it earns its place. Most of the catalogue — fire, plasma, aurora — is
for screensavers and CLI intros, not for a dashboard, where it would own the whole screen.

| Situation | Use | Budget |
|---|---|---|
| Waiting on work of unknown length | `spinner` (Kind spinner) with a label | loop until done |
| Waiting with known progress | a progress bar, not an animation | — |
| First run / splash / `--version` flair | a transition (`reveal`, `decrypt`, `beams`, `typewriter`) on a banner | under 3 s, opt-out flag |
| Highlight a result or title | `shine` or `rainbow` loop, or a one-shot `reveal` | 1–2 s |
| Screensaver, idle screen, demo | ambient (`fire`, `matrix`, `plasma`, `aurora`, `starfield`) | until keypress |
| Output is piped, CI, `TERM=dumb`, screen reader | no motion: one static frame (the engine does this automatically) | 0 |

Never block the user's real work on an animation. Intros must be skippable and short.

## 2. Discover

```sh
asciifx list                  # every effect, kind, loop/duration, glyph classes
asciifx list --json           # machine-readable specs with params
asciifx info reveal --json    # params: type, default, range, options, docs
```

`asciifx catalog --out .` writes `catalog.json` and `llms.txt` if you want the whole surface in context.

## 3. See it (the critical step)

```sh
asciifx render reveal --banner "ACME" --frames 0,12,24,-1 --format plain
asciifx render fire --w 60 --h 14 --frame 45 --format luma
asciifx render decrypt --text "ACCESS GRANTED" --at 0.8 --format json
```

- `plain`: glyphs only. Good for text effects.
- `luma`: brightness map. **Use for colour-driven effects** (fire, plasma, aurora, half-block art), where plain text shows only `▀`.
- `json`: lines, luma rows and per-row colour runs, so you can check gradients numerically.
- Negative `--frame` counts from the end; `-1` is the final frame. For transitions the final frame must equal the content: check it.
- Output is deterministic for a given `--seed`, so frame diffs are meaningful and you can paste frames into golden tests.

Tune with repeatable `-p key=value`. Unknown keys and bad values fail with the list of valid options, so read the error and fix it rather than guessing.

### Patterns compose

Every `pattern` param takes an expression, not just a name:

```sh
asciifx render reveal --banner ACME -p 'pattern=invert(center)' --frame -1
asciifx render reveal --banner ACME -p 'pattern=min(invert(center),wave)' --frame -1
asciifx render reveal --banner ACME -p 'pattern=blend(dissolve,spiral,0.25)' --frame -1
```

A pattern is a spatial ordering in [0,1] saying where cells animate first and last, so
composing orderings mixes *how* an effect spreads without adding an effect. `blend`
takes a constant weight because an ordering has no time axis; to make a mix move, change
the effect's own easing instead. `asciifx catalog` publishes the grammar as
`pattern_syntax`.

### Chain effects

`--then` plays another effect after the previous one. `-p` and `--for` apply to the effect
named last, so flags read in the order you write them:

```sh
asciifx render reveal --banner ACME -p pattern=center --then fire --for 1s --frame -1
asciifx play reveal --text HELLO --inline --then shine --for 3s
asciifx render reveal --text HI --for 500ms --then shine --for 1s --json  # frames_total: 46
```

An effect that loops needs `--for`, because a chain can only advance past an effect that
finishes; `--for` also overrides a finite effect's own length. Content is shared: every
transition in a chain transforms the same text, so a chain cannot change its subject
part-way through. `render --format json` reports a chain as `"steps": ["reveal","fire"]`
with params keyed `"1.pattern"`, `"2.palette"`.

### Dither, don't guess

In the 16- and 256-colour profiles, gradients are quantised with ordered (Bayer)
dithering by default: the encode picks between the two nearest palette entries so the
local average lands near the true colour. It is a pure function of colour and cell, so a
still region stipples rather than shimmering. `--dither none|bayer4|bayer8` changes it;
it is ignored for truecolor (nothing to dither onto).

## 4. Lint any art you write by hand

LLMs misalign 2D ASCII. Before committing banners, frames or spinner sets:

```sh
asciifx check art.txt              # file:frame:line:col: message ; exit 3 on problems
asciifx check --json spinners.json # cli-spinners {interval, frames} shape also accepted
```

It flags emoji, wide/ambiguous runes, tabs, ragged line widths and frames of differing size.

## 5. In a TUI

An effect is a widget. `Model.View()` returns a styled string, so it goes into any
lipgloss v2 layout with no second renderer:

```go
m, err := teafx.New("fire", fx.Options{W: 26, H: 8, Seed: 1})
frame := lipgloss.NewCompositor(
    lipgloss.NewLayer(title).X(0).Y(0),
    lipgloss.NewLayer(m.View()).X(1).Y(1),
).Render()
```

Do **not** mix a string `Layer` onto a lipgloss `Canvas` alongside `teafx.At(...)`: a Layer fills the
whole area it is given, so composing one wipes the drawables beneath it. Use layers and a Compositor
for a layout, or a Canvas of drawables, not both.

**Spinners.** `asciifx/spinner` is a drop-in for `charm.land/bubbles/v2/spinner`: the same types, the
same twelve predefined spinners, and the same `New`/`WithSpinner`/`WithStyle`/`TickMsg`/`Update`/
`View`/`Tick`/`ID`, so migrating is the import line and nothing else.

```go
s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
// Init: return m.s.Tick (a method value). Update: m.s, cmd = m.s.Update(msg).
```

Extras, all opt-in: `WithLabel`, `WithPalette`, `WithShimmer`, and this project's 14 single-width
frame sets via `Named("dots")` or the package vars.

Prefer this over the spinner *effect*: the frame set is a value rather than a validated param, and the
effect defaults to 15fps because it is drawn inside someone else's view — a TUI re-renders its whole
screen on every tick, so the rate is a cost the parent pays. The effect's label highlight is opt-in
(`-p shimmer=1.4`) for the same reason: it is the only part that wants more ticks, so it should not
be what sets the default.

**Cell-precise composition.** `Model` implements `uv.Drawable`, so `teafx.At(model, uv.Rect(x,y,w,h)`
composes it into a lipgloss v2 `Canvas` at a rectangle, and `teafx.Content(screen, area)` seeds a
transition with content that is already on screen. That is how you animate a panel you have already
drawn instead of replacing it.

## 6. Embed

**Go, standalone** (handles alt screen, frame pacing, resize, static fallback and terminal restore on Ctrl-C and panic):

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects" // registers every effect
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func main() {
	if err := intro(); err != nil {
		fmt.Fprintln(os.Stderr, "intro:", err) // an intro must never block the real program
	}
}

func intro() error {
	banner, err := fx.Banner("ACME", "block")
	if err != nil {
		return err
	}
	spec, err := fx.Lookup("reveal")
	if err != nil {
		return err
	}
	run, err := fx.NewRun(spec, fx.Options{
		W: 60, H: 9, Seed: 1,
		Params:  map[string]string{"pattern": "center", "palette": "synthwave"},
		Content: fx.Text(banner, tint.None),
	})
	if err != nil {
		return err // unknown or invalid params are reported here
	}
	err = term.Play(context.Background(), run, term.PlayOptions{Caps: term.Detect(os.Stdout), Inline: true})
	if errors.Is(err, term.ErrInterrupted) {
		return nil // the user skipped the intro
	}
	return err
}
```

**Bubble Tea v2**: `asciifx/teafx` is a component. `teafx.New("spinner", fx.Options{...})`, call its `Init`/`Update`/`View` from your model, and check `Done()` to move on from an intro. See `examples/bubbletea`.

**Any other language**: shell out to the CLI.
- `asciifx play <effect> --inline` for a one-shot intro.
- `asciifx cast <effect> > intro.cast` for a pre-baked asciicast v3 stream you replay.
- `asciifx render ... --format ansi --frames ...` for pre-rendered frames you embed as data.

## Non-negotiable rules (the engine enforces these; hand-rolled code must too)

- **Glyphs:** single-width glyphs only in animated regions. No emoji; widths vary 2–6 cells across terminals.
- **Restore on exit:** always restore cursor, colours and the alt screen, including on SIGINT and panic.
- **Static fallback:** one static frame when not a TTY, under `CI`, with `TERM=dumb`, or with `ASCIIFX_REDUCED_MOTION=1`. `NO_COLOR` removes colour but not motion.
- **Drawing:** diff cells, write once per frame, and wrap frames in synchronized output (`CSI ?2026h/l`). Never clear the screen each frame. Run `asciifx play --probe` to ask the terminal with DECRQM instead of trusting `TERM`; `ASCIIFX_SYNC=0` overrides.
- **Frame rate:** about 30 fps locally, 10–15 fps over SSH or tmux. `Detect` caps it automatically (`ASCIIFX_FPS` overrides). Drop late frames; never queue them.
- **Colour:** gradients in OKLab. Fade to the terminal background (unset colour), not black. Real tools should look right in 16 colours; check with `--profile 16`. Palette profiles dither by default (`--dither none` to disable).
- **Determinism:** seeded randomness and injected time, so frames can be snapshot-tested.

## References

- `docs/research.md`: ecosystem survey, terminal support matrix, formats and licensing cautions, and an addendum listing every borrowed idea with the file that carries it.
- [references/libraries.md](references/libraries.md): other-language libraries (tachyonfx, TerminalTextEffects, Ink/ora, Textual) when asciifx is not an option.
- [references/tools.md](references/tools.md): editors, converters (chafa), recorders (VHS, asciinema).
- [references/templates.md](references/templates.md): art and font collections (check licences).
