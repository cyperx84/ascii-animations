# ASCII Animations

Interactive TUI showcase of terminal ASCII animations, effects, and text banners. Built with Go, Bubble Tea, and Lip Gloss.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

```
┌─────────────────────────────────────┐
│                                     │
│   ▸ 🎯 Spinners (28)              │
│     🔥 Full-Screen Effects (8)     │
│     📝 Text Banners (60)           │
│     🎨 Splash Screens (6)          │
│     🌈 Color Showcase (3)          │
│     📦 Export (1)                   │
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

### Speed Control

Use `+`/`-` while viewing any animation to speed up or slow down (0.25x to 4x).

### Random

Press `r` while in any category to jump to a random animation.

## Features

### 🎯 Spinners — 28 animations across 7 categories

| Category | Spinners |
|----------|----------|
| **Braille** | Dots, Bounce, Dots2, Dots3, BrailleSnake |
| **Classic** | Line, Grow, Toggle, Pipe |
| **Arrows** | Arrow, Arrow2, Bounce2 |
| **Blocks** | Bar, Pulse, BarH, BlockScroll |
| **Shapes** | Circle, Square, Star, Diamond, Triangle |
| **Points** | BouncingBall, Dots4, Flip |
| **Emoji** | Moon, Clock, Earth, Weather |

### 🔥 Full-Screen Effects — 8 effects

| Effect | Description |
|--------|-------------|
| **Matrix Rain** | Green falling characters inspired by The Matrix |
| **Fire** | Rising flame simulation with heat propagation |
| **Rain** | Falling droplets with trails and splash effects |
| **Starfield** | Warp-speed stars zooming outward from center |
| **Snow** | Falling snowflakes with wind drift and accumulation |
| **DNA Helix** | Rotating double helix with A-T/G-C base pairs |
| **Wave** | Colorful sine waves scrolling across the screen |
| **Plasma** | Organic patterns with layered sine functions |

### 📝 Text Banners — 12 fonts × 5 sample texts

Fonts: **Block**, **Shadow**, **Slim**, **Dot**, **Double**, **Slant**, **Star Wars**, **ANSI Shadow**, **Doom**, **Speed**, **Thick**, **Ghost**

Press `t` to type custom text rendered live in all fonts.

### 🎨 Splash Screens — 6 demos

- **Typing Effect** — Typewriter with blinking cursor
- **Expanding Border** — Line-by-line logo reveal
- **Fade In** — Density character progression (░▒▓█)
- **Spinner Splash** — Loading spinner into logo reveal
- **Glitch Reveal** — Text un-corrupts from random characters
- **Scan Line** — Bright bar sweeps top-to-bottom revealing content

### 🌈 Color Showcase — 3 tiers

- **16 Colors** — Standard ANSI palette
- **256 Colors** — Full 256-color terminal palette
- **Truecolor** — Animated 24-bit RGB gradient

### 📦 Export

Press `e` on any animation to export it as a standalone Go program (no dependencies).

## Keyboard

| Key | Action |
|-----|--------|
| `j/k` `↑↓` | Navigate menu |
| `Enter` | Select category |
| `h/l` `←→` | Cycle animations |
| `r` | Random animation |
| `+`/`-` | Speed up / slow down |
| `s` | Toggle source code |
| `e` | Export as standalone Go file |
| `t` | Custom text (banners only) |
| `q` `Esc` | Back / Quit |

## Theme

Dracula dark theme throughout — purple accents, green selections, pink highlights.

## Build from Source

```bash
git clone https://github.com/cyperx84/ascii-animations.git
cd ascii-animations
make build
./showcase
```

## Run Tests

```bash
go test ./...
```

## Project Structure

```
cmd/showcase/main.go        — entry point
pkg/
  theme/                    — Dracula palette + Lip Gloss styles
  ui/                       — TUI model, menu, animation views
  spinners/                 — 28 spinner definitions (7 categories)
  effects/                  — 8 full-screen effects
  banners/                  — block-letter font rendering (12 fonts)
  splash/                   — 6 splash screen demos
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
| [cli-spinners](https://github.com/sindresorhus/cli-spinners) | Portable spinner JSON format (100+ patterns) |
| [go-figure](https://github.com/common-nighthawk/go-figure) | Figlet text rendering (referenced) |
| [figlet](http://www.figlet.org/) | 300+ ASCII art fonts |
| [figlet-fonts](https://github.com/cmatsuoka/figlet-fonts) | Largest figlet font collection |
| [asciimatics](https://github.com/peterbrittain/asciimatics) | Python terminal effects (inspiration) |
| [TachyonFX](https://github.com/junkdog/tachyonfx) | Ratatui shader effects (inspiration) |

Research powered by [cyperx84/openclaw](https://github.com/cyperx84/openclaw).

## License

MIT
