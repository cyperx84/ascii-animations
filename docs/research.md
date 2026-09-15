# Research: terminal animation toolkit

Compiled 2026-09-15. Stars, versions and licenses were checked against GitHub, npm and the Go module proxy on that date.

## 1. The gap

- GitHub's Copilot CLI team, in their animated banner write-up (2026-01-28): "there's no framework for terminal animations". Twenty frames took about 6k lines of TypeScript. They capped the animation under 3 s, ran it at 75 ms/frame, used 4-bit semantic colour roles with light and dark palettes, and made it opt-in, skipped in screen-reader mode. https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/
- Agent tooling today is static art only:
  - Figlet MCP servers: textarttools-mcp, ascii_banner_mcp, taag.fun, artscii.
  - The one animation tool is the GUI-driven ascii-motion-mcp, which has 69 tools and drives a web editor.
  - The TUI design skill (gfargo/tui-design-skill) has essentially nothing on animation.
- termcn (shadcn-labs, about 1.1k stars) proves the vendored-registry model for Ink and OpenTUI, but has only about 7 motion items. glyph (truffle-dev) does the same for Go.
- **Nobody offers:**
  - An effect catalog with JSON params.
  - A headless "render frame N as plain text" CLI.
  - Procedural effects you can vendor.
  - Enforced terminal safety: SIGINT restore, 2026 sync, NO_COLOR, reduced motion, non-TTY fallback.
  - A width linter for art.
  - A motion decision-tree skill.
  - Golden-frame test generation.
- Agents can't watch motion, and LLMs read 2D ASCII badly (ASCIIEval, https://arxiv.org/html/2410.01733v2). The frame-as-text preview, plus the linter, is what closes that loop.

## 2. Engines: what makes output gorgeous

| Project | Lang | Stars | License | Core abstraction |
|---|---|---|---|---|
| tachyonfx (ratatui/tachyonfx) v0.25.2 | Rust | ~1.3k | MIT | `Shader.process(dt, &mut Buffer, Rect)`: a post-process on the finished cell buffer. EffectTimer + Interpolation, spatial Patterns, CellFilter, a DSL with `to_dsl()` round-trip, the FTL web editor |
| TerminalTextEffects 0.15.0 | Python | ~4.2k | MIT | Per-character actor: a Path of bezier Waypoints (speed, ease, hold), Scenes of Frames synced to path distance, and an EventHandler chaining steps. 37 effects |
| ttfx (omacom) | Rust | ~250 | MIT | Byte-identical port of TTE, single binary |
| sysc-Go (Nomadcxx) | Go | 150 | MIT | Structs with `Update/Render/Resize/UpdatePalette`, a registry, 12 themes |
| termflix | Rust | 72 | - | 60 procedural animations; each picks braille, half-block or ASCII |
| asciimatics 1.15 | Python | ~4.3k | Apache-2.0 | Screen, Scene, Effect; particle emitters |
| Textual | Python | ~37k | MIT | `widget.animate(attr, value, duration, easing)`, with 30 Penner curves |
| harmonica | Go | ~1.6k | MIT | Springs: `NewSpring(FPS, freq, damping)` |

Go platform facts, verified with `go list` and `go doc`:
- `charm.land/bubbletea/v2` is at v2.0.9 and uses the ncurses-style "cursed" cell-diff renderer.
- `charm.land/lipgloss/v2` is at v2.0.6. It provides `NewCanvas(w,h)` with `SetCell/CellAt/Render` over `uv.Cell` (ultraviolet), plus `Layer` compositing.
- `Blend1D(steps, stops...)` and `Blend2D(w,h,angle,stops...)` blend in CIELAB.
- This repo is still on bubbletea v1.3.10 / lipgloss v1.1.0.

### Design patterns to steal
- **Effects as post-process shaders over a cell buffer**, run after the view renders.
- **Timer and easing kept separate:**
  - The 30 Penner curves.
  - Cubic-bezier `make_easing(x1,y1,x2,y2)`.
  - Springs.
- **Spatial patterns** map (cell, global t) to a local t with a soft `transition_width`:
  - Shapes: radial, diamond, spiral, diagonal, sweep, wave, checkerboard, random dissolve.
  - Combinators: min, max, blend, invert.
- **Composition:** sequence, parallel, repeat, ping_pong, delay, prolong, freeze_at, reverse, remap_alpha.
- **CellFilter selectors:** text only, by colour, outer margin, AllOf/AnyOf.
- **TTE actor model** for "text assembles itself" effects: paths with bezier waypoints, and scenes synced to distance.
- **Symbol and colour envelopes:** a flash to bright, then decay through the palette to the resting colour, while the glyph cycles random, then `░▒▓`, then the final character.
- **Gradients over space and time** in a perceptual space (OKLab/CIELAB), in horizontal, vertical, radial, diagonal or angle directions. Fade to the *terminal background*, not black.
- **Seeded RNG and fixed-dt stepping**, which gives snapshot tests.
- **Serializable config or DSL, a typed config that also generates CLI flags, and a named registry.** This is the agent enabler.
- **Palettes and themes as data.**
- **Cheap classics:**
  - DOOM fire: a heat buffer plus a 37-colour palette, pulling from the pixel below at x±rand.
  - The lolcat sine rainbow.
  - The nms decrypt resolve.
  - The donut.c luminance ramp `.,-~:;=!*#$@`.
  - Matrix head and trail.
  - Pipes box-corner walkers.
  - CRT look: dim rows, ghost trail, line jitter.

## 3. Rendering: non-negotiable rules

**Sub-cell glyph tiers:** octant (U+1CD00, Unicode 16) → sextant (U+1FB00) → quadrant → half-block ▀▄ → ASCII.
- Half-block is the default for colour work, because it gets 2 real colours per cell.
- Octants and sextants are opt-in. Ghostty draws them all natively. WezTerm draws them by default. Kitty mostly does. Alacritty draws sextants only. Windows Terminal depends on the font (Cascadia has octants).
- Braille gives 2x4 dots but one fg colour per cell.

**The rules:**
- **Detect capabilities once, with a ~100 ms timeout and an override for every probe:**
  - DECRQM `?2026$p` (sync) and `?2027$p` (grapheme clustering).
  - OSC 11 background query followed by DA1, so an unsupported terminal answers DA1 first instead of timing out.
  - `COLORTERM`/`TERM`.
  - DA1 attribute 4 for sixel.
- **Rendering discipline:**
  - One write per frame, wrapped in `CSI ?2026h … ?2026l` when supported.
  - Never clear the screen; diff cells and batch SGR runs.
  - Idle when nothing changed.
- **An animation is a pure function of (tick, seed, size, caps).** The clock and RNG are injected.
- **Frame rate:**
  - 30 fps local with 2026.
  - 10–15 fps without 2026, or under tmux or SSH. tmux only holds 2026 output since 3.7 (tmux#4744).
  - Drop late frames rather than queue them.
- **Width:**
  - Width-1 glyphs only in animated regions.
  - Force width 1 for block, braille and box ranges, whatever the locale. go-runewidth flips ambiguous-width characters to 2 under CJK locales.
  - No emoji. ZWJ emoji take 2 to 6 cells depending on the terminal.
- **Colour:**
  - Interpolate in OKLab/OKLCH and quantize by OKLab distance.
  - Use *ordered* (Bayer) dithering, never error diffusion, which shimmers.
  - Ship a 16-colour semantic-role palette with light and dark variants, picked via OSC 11.
- **Fallbacks:**
  - Honour `NO_COLOR`, which governs colour only.
  - Show a static frame when the output isn't a TTY, `TERM=dumb`, `CI` is set, or reduced motion is set by flag or env.
  - Provide a screen-reader mode with no redraws.
  - There is no standard reduced-motion env var.
- **Always restore terminal state** (cursor, SGR, alt screen, 2026 and 2027) on exit, panic and SIGINT. That means a layered try/finally, a signal handler and atexit. Bugs from getting this wrong: lint-staged#1530, dialoguer#77.
- **Graphics protocols** (kitty, sixel, iTerm) are an enhancement only, and off under tmux. The text renderer is the reference.
- **Testing:**
  - Golden frames at fixed ticks with a forced colour profile, and golden files marked `-text` in `.gitattributes`.
  - Assert cell styles explicitly where snapshots drop colour; ratatui's TestBackend and insta do.
  - Test that no change means zero bytes written.

## 4. Formats and assets

| Format | Shape | Verdict |
|---|---|---|
| cli-spinners JSON (MIT, 90 spinners) | `{"dots":{"interval":80,"frames":[...]}}` | Must be a valid subset of our format |
| ASCII Motion JSON export (core MIT, premium proprietary) | `metadata.canvasSize`, `animation{frameRate,looping}`, `frames[{duration, content, colors{foreground{"x,y":"#hex"}}}]` | Best shape to borrow |
| Durdraw .dur v8 (BSD-3) | gzipped JSON; per-frame `delay`, `contents[]` lines, `colorMap[[fg,bg]]` palette indices, artist metadata | Borrow metadata; import |
| asciicast v3 (asciinema) | NDJSON header + `[dt,"o",data]` events | Export / baked playback |
| .ans + SAUCE, REXPaint .xp, Playscii .psci | static / layered binary / dormant | Import only |
| FIGlet .flf / TOIlet .tlf | text header `flf2a$ h base max ...`, rows end in `@` | Import fonts |

No "Lottie for terminals" or cross-language animation spec exists.

**Converters and recorders:**
- chafa (LGPL-3): symbol classes include sextant and braille, but no octant.
- ascii-image-converter (Apache-2.0, stale).
- VHS (MIT): `.tape` files produce GIF, MP4 and TXT goldens, and there's a CI action.
- freeze (MIT).
- termshot (MIT).

**Node:**
- ink 7.1.1.
- ora 9.4.1 and ink-spinner, both backed by cli-spinners.
- chalk-animation and gradient-string.
- blessed is dead.

### Licensing cautions
- **Fonts:**
  - xero/figlet-fonts has **no repo license**, so check each font's header and keep its comment lines.
  - cfonts is **GPL-3**, so don't bundle it.
  - figlet.js code is MIT.
- **External tools:**
  - chafa is **LGPL-3**, so call it as an external CLI only.
  - The asciinema CLI, agg and jp2a are GPL, so call them as tools but never embed them.
- **Assets:**
  - gunnargray-dev/unicode-animations has no repo license.
  - Scene .ans art belongs to the artists.
- **This repo:** `pkg/banners` fonts are hand-drawn maps. Some carry figlet-font names (Doom, ANSI Shadow, Star Wars) but are not copied .flf data.

## 5. What makes galleries and libraries agent-friendly
- **Discovery:**
  - llms.txt (termcn and terminaltrove have one; TTE, ratatui, Textual and charm.land don't).
  - A machine-readable registry.
  - termcn's registry.json.
  - The TTE Showroom: a GIF, a copyable command, and a flags table per effect.
- **Machine-friendly output:** JSON envelopes and stable exit codes, as in agent-tty.
- **Vendored source beats dependencies** for agents, because they can read and edit what they installed.
- **Skill layout:** a slim router SKILL.md plus on-demand references (gfargo).
- **Agent terminal drivers** (tui-test, agent-tui, agent-tty) capture whole screens and are not effect-aware.
