package term

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	xterm "golang.org/x/term"
)

// PlayOptions configure Play.
type PlayOptions struct {
	Out  *os.File
	Caps Caps
	// Inline draws below the cursor instead of taking over the alternate
	// screen. Good for spinners and intro banners; the last frame stays in
	// the scrollback.
	Inline bool
	// Fit resizes the run to the terminal each frame (fullscreen only).
	Fit bool
	// Limit stops after this long; 0 plays a finite effect once, or an
	// ambient effect until a key is pressed.
	Limit time.Duration
	// Loop restarts finite effects instead of stopping.
	Loop bool
	// Hold keeps a finished fullscreen effect on screen before exiting.
	Hold time.Duration
	// Probe asks the terminal whether it supports synchronized output
	// (mode 2026) before the first frame, inside the raw-mode window Play
	// already owns. It costs at most probeTimeout and can consume input typed
	// during that window, so it is off by default. ASCIIFX_SYNC always wins.
	Probe bool
}

// probeTimeout bounds the capability probe. A terminal answers DECRQM and DA1
// within a few milliseconds; the rest is slack for a slow multiplexer.
const probeTimeout = 100 * time.Millisecond

// ErrInterrupted is returned when the user pressed q, Esc or Ctrl-C.
var ErrInterrupted = errors.New("interrupted")

// StaticTick picks the frame shown instead of an animation: the final frame
// of a finite effect, or one second into an ambient one.
func StaticTick(r *fx.Run) int {
	if n := r.Frames(); n > 0 {
		return n - 1
	}
	return r.FPS()
}

// Play runs r in the terminal. Terminal state (cursor, colours, alternate
// screen, raw mode) is restored on every exit path: normal end, error, key
// press, SIGINT/SIGTERM and panic.
func Play(ctx context.Context, r *fx.Run, o PlayOptions) (err error) {
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if !o.Caps.Animate {
		b, err := r.Seek(StaticTick(r))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(o.Out, ANSIWith(b, o.Caps.Profile, o.Caps.dither()))
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restore := setupTerminal(o, r)
	defer func() {
		if p := recover(); p != nil {
			restore()
			panic(p)
		}
		restore()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	keys := make(chan struct{}, 1)
	if raw, ok := makeRaw(); ok {
		defer raw()
		if o.Probe && !o.Caps.syncSet {
			if got := probeRaw(os.Stdin, o.Out, probeTimeout); got.SyncKnown {
				o.Caps.NoSync, o.Caps.SyncKnown = got.NoSync, true
			}
		}
		// Stop the watcher and wait for it before raw mode is restored and
		// Play returns, so no goroutine is left reading the host's stdin.
		keysDone := watchKeys(ctx, int(os.Stdin.Fd()), keys)
		defer func() {
			cancel()
			<-keysDone
		}()
	}

	ren := &Renderer{Profile: o.Caps.Profile, Sync: !o.Caps.NoSync, Dither: o.Caps.dither()}
	dt := time.Second / time.Duration(playFPS(r.FPS(), o.Caps.FPS))
	ticker := time.NewTicker(dt)
	defer ticker.Stop()
	began := time.Now() // whole playback, for Limit
	cycle := began      // current loop iteration, for frame timing
	var doneAt time.Time

	draw := func() error {
		if out := ren.Frame(r.Current()); out != nil {
			_, err := o.Out.Write(out)
			return err
		}
		return nil
	}

	for {
		if o.Fit && !o.Inline {
			if w, h, err := xterm.GetSize(int(o.Out.Fd())); err == nil {
				if cw, ch := r.Size(); cw != w || ch != h {
					if err := r.Resize(w, h); err == nil {
						ren.Reset()
						o.Out.WriteString("\x1b[H\x1b[2J")
						if r.Tick() >= 0 {
							if err := draw(); err != nil {
								return err
							}
						}
					}
				}
			}
		}
		// Catch up to wall-clock time, then draw once: slow terminals drop
		// frames instead of falling ever further behind.
		target := int(time.Since(cycle) / dt)
		if n := r.Frames(); n > 0 && target >= n {
			target = n - 1
		}
		if target > r.Tick() {
			if _, err := r.Seek(target); err != nil {
				return err
			}
			if err := draw(); err != nil {
				return err
			}
		}

		if o.Limit > 0 && time.Since(began) >= o.Limit {
			return nil
		}
		if r.Done() {
			if o.Loop {
				cycle = time.Now()
				if _, err := r.Seek(0); err != nil {
					return err
				}
				// Show frame zero now rather than leaving the last frame up
				// for a tick.
				if err := draw(); err != nil {
					return err
				}
			} else {
				if doneAt.IsZero() {
					doneAt = time.Now()
				}
				hold := o.Hold
				if o.Inline {
					hold = 0
				}
				if time.Since(doneAt) >= hold {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigs:
			return ErrInterrupted
		case <-keys:
			return ErrInterrupted
		case <-ticker.C:
		}
	}
}

// playFPS is the wall-clock tick rate for a run: its own rate, held down by
// Caps.FPS. Tick semantics stay the run's; only the wall-clock rate changes,
// so a capped playback still shows exactly the same frames. A caller that
// wants a rate rather than a ceiling sets Caps.FPS to that rate.
func playFPS(runFPS, cap int) int {
	if cap > 0 {
		return min(runFPS, cap)
	}
	return runFPS
}

func setupTerminal(o PlayOptions, r *fx.Run) (restore func()) {
	var sb strings.Builder
	sb.WriteString("\x1b[?25l") // hide cursor
	_, h := r.Size()
	if o.Inline {
		// Reserve rows below the prompt, then return to the region's top.
		sb.WriteString("\r" + strings.Repeat("\n", h-1))
		if h > 1 {
			fmt.Fprintf(&sb, "\x1b[%dA", h-1)
		}
	} else {
		sb.WriteString("\x1b[?1049h\x1b[H\x1b[2J")
	}
	o.Out.WriteString(sb.String())
	done := false
	return func() {
		if done {
			return
		}
		done = true
		var sb strings.Builder
		sb.WriteString(syncEnd + reset)
		if o.Inline {
			_, h := r.Size()
			fmt.Fprintf(&sb, "\x1b[%dB\r\n", max(h-1, 0))
		} else {
			sb.WriteString("\x1b[?1049l")
		}
		sb.WriteString("\x1b[?25h")
		o.Out.WriteString(sb.String())
	}
}

func makeRaw() (restore func(), ok bool) {
	fd := int(os.Stdin.Fd())
	if !xterm.IsTerminal(fd) || !canWatchKeys {
		return nil, false
	}
	state, err := xterm.MakeRaw(fd)
	if err != nil {
		return nil, false
	}
	// Restore without flushing so typeahead survives for the host. On BSD
	// and macOS the kernel then leaves PENDIN set until the next read, the
	// same as `stty raw; stty <saved>`; that is expected and harmless.
	return func() { _ = xterm.Restore(fd, state) }, true
}

// isQuitKey reports whether b is q, Q, Esc, Ctrl-C or Ctrl-D.
func isQuitKey(b byte) bool {
	switch b {
	case 'q', 'Q', 0x1b, 0x03, 0x04:
		return true
	}
	return false
}
