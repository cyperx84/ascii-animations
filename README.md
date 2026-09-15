# ASCII Animations

**asciifx** is a deterministic terminal animation toolkit with two things that are hard to get
elsewhere: a linter for ASCII art, and a headless renderer that prints any animation frame as
text — so an agent that cannot watch motion can still verify it.

Plus 18 procedural effects (ambient, transitions, spinners), a Bubble Tea v2 component, and a
lipgloss v2 integration.

```sh
go install github.com/cyperx84/ascii-animations/cmd/asciifx@latest
```

## Two things worth having

### 1. A linter for ASCII art

Misaligned ASCII art is a constant, boring problem, and LLMs write broken 2D art every time.
Nothing else lints for it:

```sh
$ asciifx check art.txt
art.txt:0:1:7: "😀" U+1F600: emoji width varies between terminals (2-6 cells); use single-width glyphs such as box, block or braille
art.txt:0:1:8: "\t" U+0009: tab renders as 1-8 columns depending on position; expand to spaces
art.txt:0:2:3: line is 2 columns wide, frame is 21 (pad lines to equal width so redraws overwrite every cell)
$ echo $?
3
```

It reads plain frames separated by `---`, or [cli-spinners](https://github.com/sindresorhus/cli-spinners)
JSON, and flags emoji, wide and ambiguous runes, tabs, control and zero-width characters, trailing
whitespace that differs between lines, ragged lines, and frames of drifting size. `--json` for a
machine-readable report; exit status 3 when it finds something, so it drops into CI as-is.

### 2. Frames an agent can read

Every run is a pure function of `(effect, params, size, seed, tick)`, so any frame can be printed
instead of watched:

```sh
asciifx render fire --frame 45 --format luma          # brightness map, for colour-only effects
asciifx render reveal --banner HI --frame -1          # plain glyphs; -1 is the last frame
asciifx render decrypt --text "ACCESS" --at 0.8 --json # lines, luma rows and colour runs
```

That is the loop an agent can actually close: render → inspect as text → tune a param → render
again. `asciifx catalog --out .` writes `catalog.json` and `llms.txt` describing the whole surface,
including the pattern grammar. `skill/SKILL.md` teaches the loop.

## For TUI builders

An effect is a widget, not a takeover. `Model.View()` returns a styled string, so it drops into any
lipgloss v2 layout:

```go
import (
	"charm.land/lipgloss/v2"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
)

m, err := teafx.New("fire", fx.Options{W: 26, H: 8, Seed: 1})

frame := lipgloss.NewCompositor(
	lipgloss.NewLayer(title).X(0).Y(0),
	lipgloss.NewLayer(m.View()).X(1).Y(1),   // the effect, positioned like anything else
	lipgloss.NewLayer(spinner.View()).X(1).Y(10),
).Render()
```

Run [`examples/lipgloss`](examples/lipgloss) to see it.

**A spinner that replaces `bubbles/spinner`.** Same call shape — `New`, `Update`, `View`, `Tick` —
14 styles, and `View` emits no padding so a layout can measure it:

```go
sp, err := teafx.NewSpinner("dots2", teafx.WithLabel("Compiling"))

func (m model) Init() tea.Cmd                 { return sp.Tick }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sp, cmd = m.sp.Update(msg)                // forward every message; it ignores what is not its own
	return m, cmd
}
func (m model) View() tea.View                { return tea.NewView(m.sp.View()) }
```

**Cell-precise composition.** `Model` implements `uv.Drawable`, so it composes into a lipgloss v2
`Canvas` cell by cell, with `Snapshot` and `Content` to animate a region that is already on screen:

```go
canvas.Compose(teafx.At(model, uv.Rect(2, 1, 40, 10)))
run, _ := fx.NewRun(spec, fx.Options{
	W: area.Dx(), H: area.Dy(),
	Content: teafx.Content(screen, area),   // animate the panel you already drew
})
```

`At` is needed because `Canvas.Compose` hands every drawable the whole canvas. And do not mix the two
paths on one canvas: a string `Layer` fills the entire area it is given, so composing one wipes the
drawables under it.

## Effects

| Kind | Count | Names |
|---|---|---|
| Ambient (fills the buffer, loops) | 10 | aurora, dna, donut, fire, matrix, pipes, plasma, rain, snow, starfield |
| Transitions and loops over content | 7 | beams, decrypt, glitch, rainbow, reveal, shine, typewriter |
| Spinner | 1 | 14 styles |

Two things make the output look like something:

- **OKLab gradients over space and time**, with eased per-cell progress along spatial patterns,
  flash-and-settle colour envelopes, and half-block and braille sub-cell resolution.
- **Ordered (Bayer) dithering** in the 16- and 256-colour profiles, which cuts gradient banding by
  26% at 16 colours and 40% at 256 (measured in `TestDitherReducesPerceivedError`, on the local
  average the eye integrates). It is a pure function of colour and cell, so a still frame stipples
  instead of crawling. Error diffusion is deliberately not used: it shimmers.

Patterns compose, so an effect's *spread* can be shaped without changing the effect:

```sh
asciifx render reveal --banner HI -p 'pattern=min(invert(center),wave)' --frame -1
```

Effects chain, with flags scoped to the effect named last:

```sh
asciifx render reveal --text HI -p pattern=center --for 400ms \
    --then fire --for 1s --then shine --for 400ms --frame -1
```

## Commands

| Command | What it does |
|---|---|
| `check` | Lint ASCII art frames. Exit 3 on findings. |
| `render` | Render frame N as `plain`, `luma`, `ansi` or `json`. |
| `list`, `info` | The effect registry, as a table or JSON specs. |
| `catalog` | `catalog.json` + `llms.txt` for agents. |
| `play` | Animate in the terminal; one static frame when not a TTY. |
| `cast` | Deterministic asciicast v3 recording. |

Shared flags: `-p key=value` (scoped to the effect named last), `--then`/`--for` to chain,
`--seed`, `--w/--h`, `--fps`, `--profile`, `--dither`, and for transitions `--text`, `--banner`,
`--font`, `--file`. `asciifx help <command>` has the details.

Terminal safety, because a broken terminal is worse than no animation:

- Single-width glyphs only, in animated regions.
- Diffed frames wrapped in synchronized output (mode 2026). `play --probe` asks the terminal with
  DECRQM instead of assuming; `ASCIIFX_SYNC=0` overrides.
- 256/16/no-colour fallbacks. `NO_COLOR` removes colour but keeps motion.
- One static frame when output is not a TTY, under `CI`, with `TERM=dumb`, or with
  `ASCIIFX_REDUCED_MOTION=1`.
- Frame rate capped to 15 fps over SSH, tmux and screen (`ASCIIFX_FPS` overrides), because dropped
  frames look better than queued ones.
- Terminal restored on exit, key press, SIGINT and panic.

Environments: `ASCIIFX_COLOR`, `ASCIIFX_DITHER`, `ASCIIFX_FPS`, `ASCIIFX_SYNC`,
`ASCIIFX_REDUCED_MOTION`, `ASCIIFX_FORCE_ANIMATION`, `NO_COLOR`.

## Embed it

```go
package main

import (
	"context"
	"errors"
	"os"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects" // registers every effect
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func main() {
	banner, _ := fx.Banner("ACME", "block")
	spec, _ := fx.Lookup("reveal")
	run, err := fx.NewRun(spec, fx.Options{
		W: 60, H: 9, Seed: 1,
		Params:  map[string]string{"pattern": "center", "palette": "synthwave"},
		Content: fx.Text(banner, tint.None),
	})
	if err != nil {
		return // unknown or invalid params are reported here
	}
	err = term.Play(context.Background(), run, term.PlayOptions{
		Caps: term.Detect(os.Stdout), Inline: true,
	})
	if errors.Is(err, term.ErrInterrupted) {
		return // the user skipped the intro
	}
	_ = err
}
```

`term.Play` handles the alternate screen, frame pacing, resize, the static fallback and terminal
restore. For any other language, shell out to `asciifx play`, `cast` or `render`.

## Install and build

```sh
go install github.com/cyperx84/ascii-animations/cmd/asciifx@latest   # the toolkit
go install github.com/cyperx84/ascii-animations/cmd/showcase@latest  # the demo TUI
```

From source:

```sh
git clone https://github.com/cyperx84/ascii-animations
cd ascii-animations
make build && ./showcase
make test    # or: go test ./...
```

**Golden frames.** 20 of them live in `cmd/asciifx/testdata/`, covering every glyph class, every
effect kind and every output format, with the exit code pinned. They are what stops a renderer
regression from shipping silently:

```sh
go test ./cmd/asciifx -run Golden              # compare
go test ./cmd/asciifx -run Golden -update      # regenerate, then read the diff
```

## Showcase

The original interactive TUI, on Bubble Tea v2 and Lip Gloss v2.

```
┌─────────────────────────────────────┐
│   ASCII Animations Showcase         │
│                                     │
│   ▸ 🎯 Spinners (28)                │
│     🔥 Full-Screen Effects (8)      │
│     📝 Text Banners (60)            │
│     🎨 Splash Screens (6)           │
│     🌈 Color Showcase (3)           │
│     📦 Export (1)                   │
│                                     │
│   j/k ↑↓ navigate · enter select    │
└─────────────────────────────────────┘
```

`j/k` or arrows navigate, `enter` selects, `q`/`esc` back. Inside an animation: `h/l` cycle, `r`
random, `+`/`-` speed, `s` source, `e` export as a standalone Go program, `t` custom banner text.

## Project structure

```
asciifx/            the engine and CLI toolkit
  cell/             the W×H cell buffer every effect draws into
  fx/               registry, patterns, composition, the Run driver
  effects/          18 procedural effects
  ease/ tint/       Penner curves and cubic-bezier; OKLab colour, palettes, dithering
  term/             capability detection, diff encoding, the player
  teafx/            Bubble Tea v2 component, spinner, lipgloss/uv bridge
cmd/asciifx/        the CLI, plus cmd/asciifx/testdata/ golden frames
cmd/showcase/       the demo TUI
pkg/                the showcase's own components (spinners, effects, banners, splash)
examples/           bubbletea/ and lipgloss/ integration examples
skill/              the agent skill: the render → inspect → tune → embed loop
docs/research.md    ecosystem survey, terminal support matrix, and what was borrowed from where
```

## Credits

| Library | Use |
|---|---|
| [Bubble Tea v2](https://charm.land) | TUI framework (the showcase and `teafx`) |
| [Lip Gloss v2](https://charm.land) | Styling, layout, Canvas and Compositor |
| [ultraviolet](https://github.com/charmbracelet/ultraviolet) | The cell and screen types the bridge targets |
| [briandowns/spinner](https://github.com/briandowns/spinner) | Spinner presets (referenced) |
| [cli-spinners](https://github.com/sindresorhus/cli-spinners) | Portable spinner JSON format |
| [go-figure](https://github.com/common-nighthawk/go-figure) | Figlet text rendering (referenced) |
| [figlet](http://www.figlet.org/) | 300+ ASCII art fonts |
| [asciimatics](https://github.com/peterbrittain/asciimatics) | Python terminal effects (inspiration) |
| [TachyonFX](https://github.com/junkdog/tachyonfx) | Ratatui shader effects (inspiration) |
| [TerminalTextEffects](https://github.com/ChrisBuilds/terminaltexteffects) | Actor-model effects (inspiration) |

Research behind the design, including what was borrowed from which project and what was declined
with reasons: [docs/research.md](docs/research.md).

## License

MIT
