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

## 6. Addendum: re-survey and what was borrowed

Re-checked 2026-09-15 against the live sources below. Everything in this section is
implemented in this repo; each item names the file that carries it. This section is the
breadcrumb trail, so a later reader can tell a deliberate borrow from an accident.

### Sources read this round

- **tachyonfx** (ratatui/tachyonfx) — read the repo tree and README. Confirmed: `src/dsl/`
  holds a full effect DSL (tokenizer, parser, completions engine, method chains, a
  `dsl_format` round-trip), `src/pattern/` has 12 pattern types with combined/blend/inverted
  combinators, `src/cell_filter/` has predicate + analyzer + processor, `src/fx/` has ~40
  effects, plus `effect_manager`, `effect_timer`, `interpolation`, `color_space`,
  `simple_rng` and a `widget/effect_span`. Compositions are `sequence`, `parallel`,
  `repeat`, `ping_pong`, `delay`, `prolong_start/end`, `freeze_at`, `remap_alpha`,
  `run_once`, `never_complete`, `with_duration`. Cell targeting is
  `CellFilter::{FgColor, Outer(Margin), AllOf}`. There is a browser editor (FTL) and
  `EffectDsl::compile("fx::dissolve(500)")`. WASM and `no_std` support.
- **TerminalTextEffects** (ChrisBuilds/terminaltexteffects) — 37 effects. The borrowable
  ideas are the ones this repo already had: a typed effect config dataclass that is
  *automatically* exposed as CLI arguments, per-effect `-h`, canvas/anchor options
  (`--canvas-width`, `--anchor-canvas`, `--anchor-text`, `--reuse-canvas`, `--no-eol`),
  `--frame-rate`, `--xterm-colors`, `--terminal-background-color`,
  `--existing-color-handling`, and `--random-effect` with include/exclude lists. Paths with
  bezier waypoints, scenes, and an event handler remain unimplemented here.
- **sysc-Go** (Nomadcxx) — the closest Go peer. `animations/` is flat files plus a
  `registry.go`; 12 themes; a TUI with a built-in BIT art editor and **174 `.bit` fonts**;
  assets ship as GIFs. Its lesson is packaging (installer, AUR, TUI font browser), not
  engine architecture.
- **tui-vfx** (crates.io) — a newer Rust effects crate; page did not extract, so nothing
  is claimed about it here.
- **ASCII Motion** docs (`docs.ascii-motion.com`) — plain-text, structured-JSON and
  session export; JSON import. Session files are JSON with either `.json` or `.asciimtn`.
  Still the best shape to borrow for an editor format; still not adopted.
- **DECRQM / DEC synchronized update** (ansicode.eversources.app) — `CSI ? Pd $ p` queries
  the mode; reply is `CSI ? Pd ; Ps $ y` with Ps 0 unrecognised, 1 set, 2 reset, 3
  permanently set, 4 permanently reset. Confirmed against the spec pages.

### Implemented

| Borrow | Source | Where |
|---|---|---|
| Ordered (Bayer) palette dithering, no error diffusion | research §3 rule, previously unimplemented | `asciifx/tint/dither.go`, `asciifx/tint/quantize.go`, `asciifx/term/encode.go`, `--dither` |
| Position-stable quantisation, so still frames do not crawl | same | `To256At`/`To16At`, cache keyed by colour so dithering does not grow the cache |
| Dither picks between the two nearest entries by projected fraction | classic ordered dithering; tachyonfx's nearest-entry model | `pickDithered`, `between`, `twoNearest` |
| Pattern combinators `min`/`max`/`blend`/`invert` | tachyonfx `CombinedPattern`, `BlendPattern`, `InvertedPattern` | `asciifx/fx/pattern.go`, `ParsePattern` |
| Missing pattern shapes `checkerboard`, `wave` | tachyonfx `CheckerboardPattern`, `WavePattern` | `asciifx/fx/pattern.go` |
| Real capability negotiation with a bounded timeout and an override for every probe | research §3 rule; tachyonfx and TTE do not probe at all | `asciifx/term/probe.go`, `Caps.SyncKnown`, `play --probe`, `ASCIIFX_SYNC` |
| Transport-aware frame-rate cap (30 local, 15 SSH/tmux/screen) | research §3 rule | `asciifx/term/caps.go` `transportFPS`, `ASCIIFX_FPS` |
| Golden frames at fixed ticks with a forced profile, marked `-text` | research §3 rule | `cmd/asciifx/golden_test.go`, `cmd/asciifx/testdata/*.golden`, `.gitattributes` |
| Pattern grammar exposed to agents | research §5 "machine-readable registry" | `fx.PatternGrammar`, `catalog.json.pattern_syntax` |
| Composition reachable from the CLI: a chain of effects with per-step params | tachyonfx `sequence` + method chains; the existing `fx.Sequence` was dead code from the CLI's side | `fx.Compose`, `fx.Timed`, `--then`, `--for`, scoped `-p` |
| Per-step seed derivation, so appending a link does not invalidate earlier frames | tachyonfx's stateless effects make this automatic; here effects capture a seed at construction | `Compose`'s `New`, tested by `TestComposeAppendingAStepKeepsEarlierFrames` |

### Verified numbers

Ordered dithering was checked, not assumed. `TestDitherReducesPerceivedError` renders a
gradient, quantises it, and compares the mean colour of each 8x8 block against the mean of
the ideal colours there — the quantity an eye integrates, since per-cell error *rises* by
design under dithering. Mean block error in OKLab Euclidean distance:

| Profile | none | bayer4 | bayer8 |
|---|---|---|---|
| 16 | 0.07333 | 0.05469 | 0.05457 |
| 256 | 0.01960 | 0.01175 | 0.01185 |

That is a 26% reduction at 16 colours and 40% at 256. A lightness-only dither was tried
first and managed only 0.4–0.8%, because the error in a saturated palette is mostly chroma;
dithering along the segment between the two nearest entries is what recovers it.

### Deliberately not taken

- **Mode 2027 (grapheme clustering) probing** — this engine emits single-width glyphs only,
  so clustering cannot change what it draws. Probing it would be dead code.
- **OSC 11 background query** — the value has nowhere to go until effects can fade to the
  terminal background rather than to a palette stop. Probing it now would be dead code.
- **Error diffusion** — kept out on purpose; see the `Dither` doc comment.
- **An effect DSL and completions engine** — tachyonfx's `src/dsl/` is a large surface (a
  tokenizer, parser, an editor completion engine and a formatter). The typed-params-plus-CLI
  model already gives agents validation and "did you mean" hints for a fraction of the code.
  Worth revisiting only if agents start *writing* effects rather than invoking them.
- **Time-varying pattern blend** — an ordering has no time axis here, so `blend` takes a
  constant weight. TachyonFX can crossfade patterns over an effect's lifetime; that belongs
  in the effect's easing, not its pattern.
- **Fast-forwarding a composition past a simulation** — `Run.Resize` resumes a pure
  transition by jumping to its tick, because a transition is a function of the tick. A
  looping child is not: its state depends on how many times it has been stepped, so a
  resized run containing one restarts that step instead of resuming it. Making this exact
  would mean replaying from tick zero on every resize, which for a ten-minute ambient run
  is thousands of ticks of work in the middle of a redraw. Documented in `Compose`, and
  either behaviour is defensible; restarting is the cheaper one.
- **Per-step content** — every transition in a chain transforms the same buffer, so
  "reveal A, then reveal B" is not expressible. TachyonFX has the same model (effects
  post-process one finished buffer), so this is a property of the shader architecture
  rather than a shortcut here.
- **`--then` in `list`/`info`/`catalog`** — a composition is per-invocation and never
  registered, so the registry keeps describing effects and the shell composes them. A
  registered composition would need a name and a file format, which is the DSL this repo
  has already declined.

## 7. Addendum: the TUI integration round

Second pass, same day. The first pass deepened the engine; this one answers the question the engine
depth did not: *can a TUI builder actually use this?* Much of the answer was no, and that is recorded
here because it is the more useful finding.

### What was wrong

Three structural mismatches, each verified rather than assumed:

1. **Two Bubble Tea stacks.** `cmd/showcase` and `pkg/ui` were on `github.com/charmbracelet/bubbletea`
   v1.3.10 with Lip Gloss v1, while `asciifx/teafx` was on `charm.land/bubbletea/v2` v2.0.9. Both were
   direct dependencies. The only TUI integration point served v2, so the repository's own TUI could
   not use it.
2. **Ten of eighteen effects own the whole buffer** (`asciifx list --json`: 10 ambient, 7 transitions
   and content loops, 1 spinner). A TUI has a dashboard in that screen, so the ambient catalogue is
   for screensavers and CLI intros, not for a status line. The category a TUI actually needs —
   a spinner — was one effect, competing with `bubbles/spinner`, which is forty lines and already in
   every go.mod in the ecosystem.
3. **The rendering core duplicated Lip Gloss v2.** Verified by fetching the module: `charm.land/lipgloss/v2`
   v2.0.6 ships `Blend1D` and `Blend2D` (CIELAB), `NewCanvas` over `uv.Cell` with `SetCell`/`CellAt`/
   `Render`, `NewCompositor` and `Layer`. That is the same job as `asciifx/tint` plus
   `asciifx/cell.Buffer`, and it is what a Charm user already has installed. So the dithering work in
   §6 improves a gradient pipeline most TUI builders would never adopt, because they have one.

### What was done about it

| Change | Why |
|---|---|
| Migrated `cmd/showcase`, `pkg/ui` and `pkg/theme` to Bubble Tea v2 and Lip Gloss v2 | One stack, and the repository can now demonstrate its own integration path. The v1 modules are gone from `go.mod`. |
| `teafx` uv bridge: `ToUV`, `FromUV`, `UV`, `At`, `Blit`, `Snapshot`, `Content`, `Model.Draw` | A `cell.Buffer` and a `uv.Cell` are near-isomorphic, so eighteen effects become widgets in the stack the user already has, with one conversion instead of a second renderer. |
| `asciifx/spinner`, a faithful reimplementation of `bubbles/v2@v2.2.1/spinner` | The first attempt was only bubbles-*shaped*: `NewSpinner(style) (Spinner, error)`, no `Spinner{Frames,FPS}`, no `Tick() tea.Msg`, so migrating meant rewriting call sites. See §8. |
| `pkg/ui` tests, `examples/lipgloss`, README led by `check` and the agent loop | The winnable audience is CLI intros, agent-authored art and the linter — not dashboards. |

### Things learned the hard way, and now documented

- **`Canvas.Compose` hands every drawable the whole canvas**, so a widget meant for one region needs
  `teafx.At` to say where it belongs. Measured directly: composing a buffer draws at the canvas
  origin, and two unpinned composes overwrite each other.
- **A string `Layer` fills the entire area it is given.** `Layer.Draw` is
  `uv.NewStyledString(content).Draw(scr, area)` and `Compose` passes the canvas bounds, so composing
  a Layer wipes every drawable composed before it. Measured: inked cells went 148 → 4. The correct
  pattern for mixing an effect with styled text is a `Compositor` of layers, with the effect rendered
  to a string. `At` and Canvas are for a canvas whose widgets are all drawables.
- **`uv.Cell.Content` is a grapheme cluster.** Decoding only its first rune turned "e" plus a
  combining acute into a bare "e" — a different character — so a multi-rune cluster now becomes `?`,
  the same substitution `cell.Buffer.WriteString` makes.
- **Bubble Tea v2 already probes mode 2026/2027 with DECRQM**, and skips the probe for Apple Terminal
  and SSH (`shouldQuerySynchronizedOutput` in `tea.go`). Independent corroboration of the §6 decision
  to make asciifx's own probe opt-in with an override.
- **A pty with no terminal emulator cannot exercise a Bubble Tea v2 program**: nothing answers the
  capability probe, so the program reads the test's own keystrokes as the reply. The showcase is
  verified by driving the model directly in `pkg/ui/ui_test.go`, and by running the binary in a pty
  with `TERM_PROGRAM=Apple_Terminal` (which suppresses the probe) and a pty size set with `stty`.

### Still declined

- **Moving the uv bridge out of `teafx`.** A Lip Gloss-only user who wants it would pull Bubble Tea v2
  transitively. Accepted because the bridge's main consumers are `Model.Draw` and a canvas inside a
  Bubble Tea program, and the dependency-free path (`Model.View()` into a Layer) already exists and is
  the one the README leads with.
- **`CellFilter`** (tachyonfx's cell-precise targeting) — still the largest unbuilt borrow. It changes
  what an effect *may* touch rather than adding surface, so it needs a decision about which selectors
  earn their keep before it is written.

## 8. Addendum: review findings, and two claims that were not true

A review from a second agent (read-only, against `fa290cc`) found five items. All were reproduced
before being fixed; the reproductions are named here because two of them were claims this repository
was making about itself rather than simple bugs.

### Reproduced

- **`pkg/ui/ui.go:128`.** `case "enter", " ":` was dead. `KeyPressMsg.String()` returns `"space"` —
  ultraviolet refuses to stringify a bare space and falls back to the keystroke name — so the space
  bar had silently stopped opening a category. Verified by printing the string for
  `tea.KeyPressMsg{Code: tea.KeySpace}`.
- **Banner text field.** Typing `é` and pressing backspace left `"\xc3"`, invalid UTF-8, because the
  field took any rune a key produced while backspace removed one byte. Verified by driving the model
  and checking `utf8.ValidString`.
- **Duplicate tick chains.** `Tick()` twice from one state started two chains that both stayed live —
  two advances counted, two follow-up commands returned — because `gen` only ever changed in
  `Restart`. This is the same bug `bubbles/v2` fixes with a per-accepted-tick tag.
- **A doc example that did not compile.** `cmd := sp.Tick` where `Tick() tea.Cmd`:
  `cannot use sp.Tick (value of type func() tea.Cmd) as tea.Cmd`.
- **The bigger claim.** "Replaces bubbles/spinner without changing the structure" was false:
  `NewSpinner(style) (Spinner, error)` against upstream's `New(opts...) Model`, no
  `Spinner{Frames, FPS}`, no `ID()`, and `Tick` returning a `Cmd` where upstream returns a `Msg`. The
  package was bubbles-*shaped*, not a drop-in, and it shipped with that claim in its doc comment.

### Fixed

`asciifx/spinner` is now a faithful reimplementation of `bubbles/v2@v2.2.1/spinner`, read from the
module source rather than from a summary of it, with the twelve predefined spinners copied frame for
frame and the tag and ID filtering matching. The compatibility claim is checked by the compiler:
`compat_test.go` is an external test package, so it can only use the exported surface, and every
snippet in it is written the way upstream code is written.

The `styles` table moved to a leaf package (`asciifx/spinner/styles`) that imports neither Bubble Tea
nor Lip Gloss, so the headless CLI still links zero charm packages — `go list -deps ./cmd/asciifx`
reports none — while the effect and the drop-in spinner share one source of truth. Moving it also
caught a unit bug of mine: `Interval` became a `time.Duration` and the effect went on dividing
seconds by nanoseconds, a factor of a billion. The golden frames failed, which is what they are for.

The spinner effect's defaults became the cheap path: 15fps, which is the slowest rate that shows every
frame set without skipping one (the fastest is `bar` at 70ms), and the label highlight off. A
spinner is drawn inside a view that re-renders on every tick, so the rate is a cost the parent pays,
and the highlight is the only part that wants more ticks. Both facts are pinned by tests in
`asciifx/effects/spinner_test.go`, including the invariant that ties the rate to the fastest style.

### Not fixed, deliberately

Three minor items were left as they are, each because the alternative is worse:

- **`uv.go` hardcodes `Width: 1`.** A `cell.Buffer` can hold a rune of any width, and `Blit` declares
  every cell one column. Sanitising there would mean silently rewriting a caller's cell in the one
  place that is supposed to be a straight conversion; the invariant belongs to the buffer, and
  `FromUV` already substitutes `?` for anything unsafe on the way in.
- **`Blit` writes blank cells, so a buffer is opaque over a canvas.** That is the right default for
  an effect: it owns its rectangle, and a transparent blit would let stale content show through a
  frame that is supposed to have cleared. It is documented rather than changed.
- **Label width in the spinner.** The drop-in measures nothing, because upstream renders whatever
  frames it is given; `Cell`-level width safety is the job of the `styles` table, which validates
  every frame with `cell.Safe` at init.
