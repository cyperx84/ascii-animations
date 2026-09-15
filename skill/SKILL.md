---
name: asciifx
description: Add beautiful, terminal-safe ASCII animations to CLI and TUI programs with the asciifx toolkit — intro banners, text reveals, spinners, ambient backgrounds (fire, matrix, plasma, aurora). Use when adding or polishing any terminal animation, splash screen, loading indicator or text effect, when an agent needs to SEE an animation frame as text, or when linting ASCII art for alignment and unsafe glyphs. Triggers: "ascii animation", "terminal animation", "splash screen", "intro banner", "spinner", "loading animation", "text effect", "matrix rain", "tui polish", "asciifx".
---

# asciifx

asciifx is a deterministic terminal animation engine plus a CLI built for agents. You cannot watch motion, so the workflow is: **pick an effect, render frames as text, inspect, tune params, embed**.

Install: `go install github.com/cyperx84/ascii-animations/cmd/asciifx@latest`

## 1. Decide whether to animate at all

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

## 4. Lint any art you write by hand

LLMs misalign 2D ASCII. Before committing banners, frames or spinner sets:

```sh
asciifx check art.txt              # file:frame:line:col: message ; exit 3 on problems
asciifx check --json spinners.json # cli-spinners {interval, frames} shape also accepted
```

It flags emoji, wide/ambiguous runes, tabs, ragged line widths and frames of differing size.

## 5. Embed

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
- **Drawing:** diff cells, write once per frame, and wrap frames in synchronized output (`CSI ?2026h/l`). Never clear the screen each frame.
- **Frame rate:** about 30 fps locally, 10–15 fps over SSH or tmux. Drop late frames; never queue them.
- **Colour:** gradients in OKLab. Fade to the terminal background (unset colour), not black. Real tools should look right in 16 colours; check with `--profile 16`.
- **Determinism:** seeded randomness and injected time, so frames can be snapshot-tested.

## References

- `docs/research.md`: ecosystem survey, terminal support matrix, formats and licensing cautions.
- [references/libraries.md](references/libraries.md): other-language libraries (tachyonfx, TerminalTextEffects, Ink/ora, Textual) when asciifx is not an option.
- [references/tools.md](references/tools.md): editors, converters (chafa), recorders (VHS, asciinema).
- [references/templates.md](references/templates.md): art and font collections (check licences).
