# Libraries by Language

## Go

### sysc-Go
Terminal animation library with 12+ effects and TUI browser.
```bash
go install github.com/Nomadcxx/sysc-Go/cmd/syscgo@latest
go install github.com/Nomadcxx/sysc-Go/cmd/syscgo-tui@latest
```
Effects: fire, matrix-rain, rain, fireworks, beams, aquarium, fire-text, matrix-art, rain-art, pour, print, beam-text, ring-text, blackhole.
Features: 174 block-style fonts, theme system (Dracula, Nord, Catppuccin, Tokyo Night), interactive TUI for browsing/previewing.
https://github.com/Nomadcxx/sysc-Go

### briandowns/spinner
90+ configurable terminal spinners/progress indicators.
```bash
go get github.com/briandowns/spinner
```
https://github.com/briandowns/spinner

### Charm Ecosystem (Bubble Tea + Bubbles + Lip Gloss + Harmonica)
- **Bubble Tea** — Elm-architecture TUI framework
- **Bubbles** — Pre-built widgets (spinner, progress, list, text input)
- **Lip Gloss** — Terminal styling (colors, borders, margins)
- **Harmonica** — Physics-based spring animations
- **VHS** — Declarative terminal animation tapes
https://github.com/charmbracelet

### Spinix
Highly customizable loading animations (spinners + progress bars).
https://github.com/topics/progress-bar?l=go

## Node.js

### cli-spinners
100+ spinner patterns as JSON. Used by 2,297+ packages. Cross-language portable format.
```bash
npm i cli-spinners
```
https://github.com/sindresorhus/cli-spinners

### ora
Elegant terminal spinner wrapping cli-spinners.
```bash
npm i ora
```
https://github.com/sindresorhus/ora

### node-cli-frames
Create ASCII frame-by-frame animations in the terminal.
https://github.com/IonicaBizau/node-cli-frames

### Ink
React for the terminal. Build TUIs with components.
```bash
npm i ink react
```
Pair with `ink-spinner`, `ink-big-text` for animations.

### figlet
Text banner generator, 300+ fonts.
```bash
npm i figlet
```

## Rust

### Ratatui + Crossterm
The standard TUI stack. Sub-millisecond rendering, widgets, layouts.
```toml
[dependencies]
ratatui = "0.29"
crossterm = "0.28"
```
https://ratatui.rs

### TachyonFX
Shader-like terminal effects with FTL (Function Transformation Language) browser editor for visual effect creation.
https://github.com/blazing/bufferface

### indicatif
Progress bars and spinners for Rust CLIs.
```toml
[dependencies]
indicatif = "0.17"
```

## Python

### asciimatics
Cross-platform curses replacement + ASCII art animations (sprites, particles, fire effects, screen rendering). 4.3k stars.
```bash
pip install asciimatics
```
https://pypi.org/project/asciimatics/

### Textual
Modern TUI framework. 16.7M colors, mouse support, smooth flicker-free animation, CSS-like styling.
```bash
pip install textual
```
https://textual.textualize.io

### TerminalTextEffects
20+ terminal effects (matrix, rain, bouncy, bubbles, expand, fire, etc.).
```bash
pip install TerminalTextEffects
```

### yaspin / halo
Minimal terminal spinners.
```bash
pip install yaspin
pip install halo
```
