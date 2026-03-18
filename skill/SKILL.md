---
name: ascii-animations
description: Add polished ASCII animations to CLI/TUI applications. Use when adding spinners, loading animations, splash screens, text effects, or any terminal animation to a command-line tool. Covers Go, Node.js, Rust, and Python with library recommendations, drop-in templates, generators, and performance best practices. Trigger on phrases like "add animation", "ASCII animation", "spinner", "loading animation", "splash screen", "CLI animation", "terminal animation".
---

# ASCII Animations for CLIs & TUIs

Add polished terminal animations to any CLI app. This skill covers library selection, templates, generators, and engineering best practices.

## Quick Picks by Language

| Language | Animation Library | TUI Framework | Spinner |
|---|---|---|---|
| **Go** | `sysc-Go` (12+ effects, themes, fonts) | Bubble Tea + Lip Gloss | `briandowns/spinner` (90+) |
| **Node.js** | `node-cli-frames` (frame-by-frame) | Ink (React for CLI) | `ora` + `cli-spinners` (100+) |
| **Rust** | `tachyonfx` (shader effects + visual editor) | Ratatui + Crossterm | hand-roll or `indicatif` |
| **Python** | `asciimatics` (sprites, particles, fire) | Textual (16.7M colors, animation) | `yaspin` or `halo` |

For detailed library info, installation, and code examples per language, see [references/libraries.md](references/libraries.md).

## Animation Types & Templates

### Spinners (loading indicators)
- Use `cli-spinners` JSON format — portable across all languages
- 100+ presets: dots, arrows, bouncing bars, emoji, braille patterns
- Install: `npm i cli-spinners` (JSON format usable anywhere)
- Go: `briandowns/spinner` · Node: `ora` wraps cli-spinners · Python: `yaspin`

### Splash Screens & Banners
- Text logos: `figlet` (300+ fonts) or `toilet` (with color)
- Font collections: `cmatsuoka/figlet-fonts` (largest), `xero/figlet-fonts` (hacker aesthetic)
- Animated text effects: `sysc-Go` (fire-text, blackhole, ring-text, pour, print)
- Gold standard: GitHub Copilot CLI animated banner (open source, engineering blog)

### Full-Screen Effects
- Matrix rain, fire, rain, fireworks, aquarium: `sysc-Go` (Go, 12+ effects)
- TerminalTextEffects (Python, 20+ effects): `pip install TerminalTextEffects`
- Particle systems, sprites: `asciimatics` (Python)

For template collections and curated art resources, see [references/templates.md](references/templates.md).

## Generators & Creation Tools

### Design Custom Animations
- **durdraw** — Terminal-based ASCII animation editor (frame-by-frame, 256 colors, Unicode)
- **ASCII Motion** (ascii-motion.com) — Browser-based editor, export for CLI/web
- **Playscii** — Open-source ASCII/ANSI art + animation editor

### Convert Media to ASCII
- **chafa** — Best image→ASCII converter (truecolor, auto-detects terminal capabilities)
- **gif-for-cli** (Go) — Convert GIFs directly to terminal animations
- **ascii.life** — AI-powered, 12 retro themes, export GIF/MP4
- **Carbonyl** — Full Chromium browser rendered in ASCII
- **Video pipeline**: `ffmpeg -i video.mp4 frame_%03d.jpg && chafa frame_*.jpg`

### Declarative Terminal Animations
- **VHS** (by Charm) — Write terminal animations as tape files, replay them

For full tool listings with links, see [references/tools.md](references/tools.md).

## Engineering Best Practices

### Terminal Compatibility
- Use 4-bit ANSI colors (16 colors) for maximum compatibility
- Provide fallback chain: truecolor → 256 → 16 → no color
- Test across: iTerm2, Ghostty, Kitty, Alacritty, Windows Terminal, xterm
- Never assume cursor position — always use ANSI save/restore cursor (`\x1b[s` / `\x1b[u`)

### Performance
- Use real-time frame timing (`Date.now()` delta), not `sleep` intervals
- Overwrite in place instead of clear-screen (`\x1b[A` to move up, not `\x1b[2J`)
- Batch writes — double buffer in memory, write once per frame
- Hide cursor during animation: `\x1b[?25l` → restore: `\x1b[?25h`
- Minimize escape sequences — count them, they add up at 60fps
- Ghostty is currently the fastest terminal for heavy animation

### Accessibility
- Respect `NO_COLOR` environment variable
- Respect `TERM=dumb` (disable all formatting)
- Screen readers interpret rapid character changes as noise — provide `--no-animation` flag
- Don't rely solely on animation to convey information

### Frame Storage Format
For custom animations, use this simple format (widely compatible):

```
---FRAME---
<frame content here>
---END---
---FRAME---
<next frame content>
---END---
```

Or JSON (portable, parseable):
```json
{"frames": ["frame1", "frame2", "frame3"], "interval": 100}
```

### Must-Read Reference
GitHub Copilot CLI engineering blog — how they built their animated ASCII banner (6,000+ lines of TypeScript for a 3-second animation, semantic color roles, terminal fragmentation handling):
https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/
