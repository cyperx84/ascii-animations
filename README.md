# ASCII Animations

Interactive TUI showcase of terminal ASCII animations. Built with Go, Bubble Tea, and Lip Gloss.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

```
┌─────────────────────────────────────┐
│                                     │
│   ▸ 🎯 Spinners                    │
│     🔥 Full-Screen Effects          │
│     📝 Text Banners                 │
│     🎨 Splash Screens               │
│     🌈 Color Showcase               │
│     📦 Export                        │
│                                     │
│   j/k ↑↓ navigate · enter select   │
└─────────────────────────────────────┘
```

## Install

```bash
go install github.com/cyperx84/ascii-animations/cmd/showcase@latest
```

## Usage

```bash
showcase
```

Navigate with `j/k` or arrow keys, `Enter` to select, `q/Esc` to go back.

## Features

- **🎯 Spinners** — 12 spinner types with live previews (dots, braille, arrows, moon phases, clocks, bars)
- **🔥 Full-Screen Effects** — Matrix rain, fire simulation, rain, starfield warp
- **📝 Text Banners** — 5 fonts (Block, Shadow, Slim, Dot, Double) with custom text input
- **🎨 Splash Screens** — Typing effect, expanding border, fade-in demos
- **🌈 Color Showcase** — 16-color, 256-color, and truecolor gradient comparison
- **📦 Export** — Export any animation as a standalone Go code snippet

### Keyboard

| Key | Action |
|-----|--------|
| `j/k` `↑↓` | Navigate menu |
| `Enter` | Select category |
| `h/l` `←→` | Cycle animations |
| `s` | Toggle source code |
| `e` | Export as standalone Go file |
| `t` | Custom text (banners) |
| `q` `Esc` | Back / Quit |

### Theme

Dracula dark theme throughout — purple accents, green selections, pink highlights.

## Build from Source

```bash
git clone https://github.com/cyperx84/ascii-animations.git
cd ascii-animations
make build
./showcase
```

## Project Structure

```
cmd/showcase/main.go        — entry point
pkg/
  theme/                    — Dracula palette + Lip Gloss styles
  ui/                       — TUI model, menu, animation views
  spinners/                 — 12 spinner definitions
  effects/                  — matrix rain, fire, rain, starfield
  banners/                  — block-letter font rendering (5 fonts)
  splash/                   — splash screen demos
  colors/                   — 16/256/truecolor showcase
  export/                   — standalone Go snippet exporter
skill/                      — OpenClaw skill reference
```

## Credits

| Library | Use |
|---------|-----|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Terminal styling & layout |
| [briandowns/spinner](https://github.com/briandowns/spinner) | Spinner presets (referenced) |
| [cli-spinners](https://github.com/sindresorhus/cli-spinners) | Portable spinner JSON format |
| [go-figure](https://github.com/common-nighthawk/go-figure) | Figlet text rendering (referenced) |
| [figlet](http://www.figlet.org/) | 300+ ASCII art fonts |

## License

MIT
