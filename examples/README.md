# Examples

Five programs, each one a step harder than the last. Run any of them with `go run`.

Start at the one that matches what you want:

| | Run it | What it shows |
|---|---|---|
| [**01-spinner**](01-spinner) | `go run ./examples/01-spinner` | The smallest way in: change the import line and your `bubbles/spinner` code keeps working. Then `WithLabel`, `WithPalette` and `WithShimmer`, which are the only new calls. |
| [**02-intro**](02-intro) | `go run ./examples/02-intro` | A banner reveal as a startup intro, handing the screen to the app when `Done()`. `SetCaps` gives the component the terminal's verdict — try `ASCIIFX_REDUCED_MOTION=1` and watch it show one frame instead. |
| [**03-chain**](03-chain) | `go run ./examples/03-chain` | Three effects composed into one: a reveal, then fire filtered to `not(ink)` so it burns *around* the letters, then a shine. `teafx.FromRun` is what puts a chain in a Bubble Tea model. |
| [**04-panels**](04-panels) | `go run ./examples/04-panels` | Effects as widgets in a lipgloss layout. Press `p` to switch between the string path and the cell-precise Canvas path; a test asserts they paint identical glyphs. |
| [**05-dashboard**](05-dashboard) | `go run ./examples/05-dashboard` | A live dashboard. Press space and the node table is snapshotted and animated back in by `decrypt`, so the effect transforms the layout instead of replacing it. |

Three things are easy to miss, and each demo above that needs one says so in its comments:

- `m.SetCaps(term.Detect(os.Stdout))` — the frame-rate cap on slow transports and the static frame
  under CI, a pipe, `TERM=dumb` or `ASCIIFX_REDUCED_MOTION`. A component gets none of it otherwise.
- `teafx.FromRun(run)` — the only way to play an `fx.Compose` chain, whose Spec is never registered.
- `teafx.At(drawable, rect)` — needed by *everything* composed onto a lipgloss `Canvas`, chrome
  included. `Canvas.Compose` hands each drawable the whole canvas, and a `Layer` ignores its own
  X and Y there, so unpinned layers all paint at the origin.

Every demo has a headless test beside it (`go test ./examples/...`), and three of them lint their
own frames with `asciifx/lint`.
