package main

import (
	"fmt"
	"strings"
)

const runFlagsHelp = `Effect flags:
  -p key=value        effect param (repeatable); applies to the effect named last
  --seed N            random seed (default 1); same seed, same frames
  --w N --h N         size in cells (default: effect default, grown to fit content)
  --fps N             tick rate (default: effect's recommended fps)
  --profile P         colour profile: truecolor, 256, 16, none (default: detected)
  --dither D          ordered dither for 16/256 colour: bayer8 (default),
                      bayer4, none. Stable per cell, so still frames do not
                      shimmer. Ignored for truecolor and no-colour output.
Chaining:
  --then EFFECT       play another effect after the previous one (repeatable)
  --for D             give the effect named last a duration, e.g. 2s. Required
                      for an effect that loops, and it overrides a finite
                      effect's own length.
  --filter EXPR       restrict which cells the effect named last may change.
                      Cells the selector rejects are left exactly as the effect
                      found them, so the effect runs unchanged and its output is
                      clipped. Selectors:
                        ink                 cells with a visible glyph
                        not(A)              the opposite of A
                        all(A,B,..)         every one of them
                        any(A,B,..)         at least one of them
                        inner(H[,V])        inside a margin, V defaulting to H
                        outer(H[,V])        outside that margin
                        fg(#rrggbb|none)    an exact foreground colour
                      For example --filter 'not(ink)' lets an ambient effect
                      burn around a banner instead of through it, and
                      --filter 'all(ink, inner(1))' animates text inside a frame.
                      A filter that reads the cell (ink, fg and anything built
                      from them) needs content in the run, because there is
                      nothing to read otherwise; inner and outer do not.

-p, --for and --filter apply to the effect named last, so flags read in the
order they are written:

  asciifx render reveal --text HI -p pattern=center \
      --then shine --for 2s -p palette=matrix \
      --then fire --for 1s --filter 'not(ink)'

Content is shared: every transition in a chain transforms the same text, so a
chain cannot change its subject part-way through. A chain of one effect is the
plain single effect, flag for flag.
Content (transitions only; default is the block banner "ASCIIFX"):
  --text "A\nB"       literal text, \n starts a new line
  --banner TEXT       text rendered in a banner font (--font block|slim|mini)
  --file PATH         read content from a file, or - for stdin
`

func init() {
	commands = []*command{
		{
			name:    "list",
			summary: "List effects as a table or JSON specs.",
			usage:   "asciifx list [--json] [--kind ambient|transition|spinner]",
			details: `Flags:
  --json              array of effect specs, including params
  --kind K            only effects of kind K

Examples:
  asciifx list
  asciifx list --kind transition --json
`,
			run: cmdList,
		},
		{
			name:    "info",
			summary: "Show one effect's full spec: params with types, defaults, ranges and options.",
			usage:   "asciifx info <effect> [--json]",
			details: `Flags:
  --json              the spec plus "frames" (count at default fps, 0 = loops)

Examples:
  asciifx info reveal
  asciifx info fire --json
`,
			run: cmdInfo,
		},
		{
			name:    "render",
			summary: "Render frames headlessly and deterministically as text, luma, ANSI or JSON.",
			usage:   "asciifx render <effect> [--frame N | --frames 0,10,20 | --every K | --at SECONDS] [--format plain|luma|ansi|json]",
			details: `Frame selection (pick one; default is the static frame: last frame of a
finite effect, 1s into a looping one):
  --frame N           tick N; negative counts from the end (-1 = last)
  --frames 0,10,-1    several ticks
  --every K           every Kth tick; finite effects always include the last
                      frame, looping effects stop at --seconds (default 3)
  --at SECONDS        tick at a time; negative counts from the end

Formats:
  plain               the characters, every row padded to --w
  luma                brightness map (" .:-=+*#%@") so colour-only effects
                      like half-block fire stay readable as text
  ansi                styled output for --profile
  json                {"effect","tick","time","w","h","fps","seed","params",
                      "steps","frames_total","lines","luma","styles"}; styles
                      holds, per row, runs {"x","len","fg","bg","attrs"} of
                      coloured cells (unstyled runs omitted). "steps" lists a
                      chain's effects in order, and its params are keyed
                      "<step>.<name>". A --filter expression is reported beside
                      them as "filter", or "<step>.filter" for a chain, so a
                      frame can be reproduced from the JSON alone. Several
                      frames => array.
Several frames in text formats are separated by "--- frame N (t=0.40s)".

` + runFlagsHelp + `
Examples:
  asciifx render reveal --text "HELLO" --frame -1
  asciifx render reveal --banner HI --frames 0,12,-1 --format json
  asciifx render fire --at 2 --format luma --w 40 --h 10
  asciifx render reveal --banner HI --then fire --for 1s --frame -1
  asciifx render reveal --banner HI --then fire --for 1s --filter 'not(ink)'
  asciifx render glitch --text ACCESS --filter 'all(ink,inner(2,1))'
  asciifx render reveal --every 10 | asciifx check -
`,
			run: cmdRender,
		},
		{
			name:    "play",
			summary: "Play an effect in the terminal; prints one static frame when not a TTY.",
			usage:   "asciifx play <effect> [--inline] [--loop] [--limit 5s] [--hold 1s]",
			details: `Flags:
  --inline            draw below the cursor instead of the alternate screen
  --loop              restart finite effects
  --limit D           stop after D (e.g. 5s); looping effects otherwise run
                      until q, Esc or Ctrl-C
  --hold D            keep a finished fullscreen effect on screen (default 1s)
  --probe             ask the terminal whether it supports synchronized output
                      (mode 2026) before the first frame. Costs at most 100ms
                      and may consume input typed during that window, so it is
                      off by default. ASCIIFX_SYNC always wins.

Fullscreen play fits the terminal unless --w/--h are given. When stdout is
not a terminal, CI is set, TERM=dumb or ASCIIFX_REDUCED_MOTION=1, a single
static frame is printed instead. ASCIIFX_FORCE_ANIMATION=1 overrides that;
ASCIIFX_COLOR and NO_COLOR control colour, and ASCIIFX_FPS caps the tick rate
(30 locally, 15 over SSH, tmux or screen).

` + runFlagsHelp + `
Examples:
  asciifx play fire --limit 5s
  asciifx play reveal --banner HELLO --inline -p palette=synthwave
  asciifx play reveal --text HELLO --inline --then shine --for 3s
  asciifx play reveal --banner HI --then fire --for 5s --filter 'not(ink)'
`,
			run: cmdPlay,
		},
		{
			name:    "check",
			summary: "Lint ASCII art frames for unsafe glyphs, tabs, ragged lines and frame size drift.",
			usage:   "asciifx check [--json] [--frames-sep '---'] <file|->",
			details: `Input is text frames separated by a separator line (a line equal to the
separator or starting with "separator "), or cli-spinners JSON:
{"interval":80,"frames":["..."]} or a map of name to that.

Reports runes that are not single-width-safe (emoji, wide, variable width,
control, zero-width, tab), trailing whitespace that differs between lines of
a frame, lines of different display width within a frame, and frames whose
height or width differ from the first frame.

Output: file:frame:line:col: message   (frame 0-based; line and col 1-based;
line is the file line for text input, the line within the frame for JSON)

Flags:
  --json              {"ok","file","frames","issues":[{"file","set","frame",
                      "line","col","kind","rune","code","message"}]}
  --frames-sep S      frame separator line (default ---)

Exit status 3 when issues are found.

Examples:
  asciifx check art.txt
  asciifx check --json spinners.json
  asciifx render reveal --every 8 | asciifx check -
`,
			run: cmdCheck,
		},
		{
			name:    "catalog",
			summary: "Write catalog.json and llms.txt describing every effect, palette, pattern and easing.",
			usage:   "asciifx catalog [--out DIR]",
			details: `Flags:
  --out DIR           write DIR/catalog.json and DIR/llms.txt; without it,
                      print catalog.json to stdout

Examples:
  asciifx catalog --out docs/
  asciifx catalog | jq '.effects[].name'
`,
			run: cmdCatalog,
		},
		{
			name:    "cast",
			summary: "Export an asciicast v3 recording, deterministically.",
			usage:   "asciifx cast <effect> [--seconds 3] > out.cast",
			details: `Flags:
  --seconds S         length (default: the whole effect if finite, 3s if it loops)

Colour defaults to truecolor; pass --profile to change it.

` + runFlagsHelp + `
Examples:
  asciifx cast fire --seconds 3 > fire.cast
  asciifx cast reveal --banner HELLO > hello.cast && asciinema play hello.cast
`,
			run: cmdCast,
		},
		{
			name:    "version",
			summary: "Print the version.",
			usage:   "asciifx version [--json]",
			details: `Examples:
  asciifx version
`,
			run: cmdVersion,
		},
		{
			name:    "help",
			summary: "Show help for a command.",
			usage:   "asciifx help [command]",
			details: `Examples:
  asciifx help render
`,
			run: cmdHelp,
		},
	}
}

func mainHelp() string {
	var b strings.Builder
	b.WriteString(`asciifx: deterministic terminal animations for humans and coding agents

Usage:
  asciifx <command> [arguments]

Commands:
`)
	for _, c := range commands {
		fmt.Fprintf(&b, "  %-9s %s\n", c.name, c.summary)
	}
	b.WriteString(`
Agents: start with "asciifx catalog" or "asciifx list --json", inspect an effect
with "asciifx info <effect> --json", and look at motion with
"asciifx render <effect> --frames 0,10,-1".

Exit codes: 0 ok, 1 runtime error, 2 usage error, 3 check found problems.
Commands with --json print errors as {"error": "...", "hint": "..."} on stdout.

Run "asciifx help <command>" for flags and examples.
`)
	return b.String()
}

func commandHelp(c *command) string {
	return fmt.Sprintf("%s\n\nUsage:\n  %s\n\n%s", c.summary, c.usage, c.details)
}

func cmdHelp(e *env, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(e.stdout, mainHelp())
		return nil
	}
	c := findCommand(args[0])
	if c == nil {
		names := make([]string, len(commands))
		for i, c := range commands {
			names[i] = c.name
		}
		hint := "commands: " + strings.Join(names, ", ")
		if s := suggest(args[0], names); s != "" {
			hint = fmt.Sprintf("did you mean %q? %s", s, hint)
		}
		return usageErr(hint, "no help for unknown command %q", args[0])
	}
	fmt.Fprint(e.stdout, commandHelp(c))
	return nil
}

func cmdVersion(e *env, args []string) error {
	c := findCommand("version")
	fs := newFlags(c.name)
	asJSON := fs.Bool("json", false, "emit JSON")
	if _, err := parse(e, c, fs, args); err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(e.stdout, map[string]string{"name": "asciifx", "version": versionString()})
	}
	fmt.Fprintf(e.stdout, "asciifx %s\n", versionString())
	return nil
}
