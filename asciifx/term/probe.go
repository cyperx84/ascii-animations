package term

import (
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	xterm "golang.org/x/term"
)

// capQueries asks the terminal two questions and then a third whose only job
// is to produce a reply:
//
//   - DECRQM ?2026$p — is synchronized output supported? Answered "?2026;1$y"
//     when set, "?2026;2$y" when recognised but reset, "?2026;0$y" when the
//     mode is not recognised at all.
//   - DA1 — device attributes. Every terminal answers this, so its reply is
//     the sentinel that ends the read even on terminals that ignore the first
//     query. Without it a terminal with no 2026 support would leave us
//     waiting for the full timeout.
//
// Mode 2027 (grapheme clustering) is deliberately not probed: asciifx only
// emits single-width glyphs, so clustering cannot change what it draws.
const capQueries = "\x1b[?2026$p\x1b[c"

// Probe asks the terminal what it supports. It is the only function here that
// reads from the terminal, so it is opt-in: it runs at most for timeout and
// may consume input typed during that window.
//
// The answer is merged over the zero value, so a terminal that stays silent
// leaves every field unset and the caller keeps its defaults. Prefer this over
// guessing: DECRQM gives a real yes/no for mode 2026, where TERM and
// TERM_PROGRAM only give hints, and tmux passes 2026 through only from 3.7.
func Probe(in, out *os.File, timeout time.Duration) Caps {
	c := Caps{Animate: true}
	if in == nil || out == nil {
		return c
	}
	fd := int(in.Fd())
	if !xterm.IsTerminal(fd) || !xterm.IsTerminal(int(out.Fd())) {
		return c
	}
	state, err := xterm.MakeRaw(fd)
	if err != nil {
		return c
	}
	defer func() { _ = xterm.Restore(fd, state) }()
	return probeRaw(in, out, timeout)
}

// probeRaw is Probe for a terminal that is already in raw mode. Play uses it
// so the probe shares Play's raw-mode window instead of switching raw mode
// twice, which would restore the terminal to raw on the way out.
func probeRaw(in, out *os.File, timeout time.Duration) Caps {
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}
	reply := query(in, out, timeout, in.SetReadDeadline)
	c := Caps{Animate: true}
	if sync := parseSync(reply); sync >= 0 {
		c.SyncKnown = true
		c.NoSync = sync == 0
	}
	return c
}

// query writes the capability probes and reads replies until DA1 arrives, the
// deadline passes, or 4 KiB have arrived. It returns whatever was read, which
// may be empty. A nil deadline function reads until DA1 or an error.
func query(in io.Reader, out io.Writer, timeout time.Duration, deadline func(time.Time) error) string {
	if _, err := io.WriteString(out, capQueries); err != nil {
		return ""
	}
	if deadline != nil {
		if err := deadline(time.Now().Add(timeout)); err != nil {
			// The descriptor is not pollable; skip rather than block forever.
			return ""
		}
		defer func() { _ = deadline(time.Time{}) }()
	}
	var sb strings.Builder
	buf := make([]byte, 256)
	for sb.Len() < 4096 {
		n, err := in.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
			if hasDA1(sb.String()) {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return sb.String()
}

// hasDA1 reports whether a complete DA1 reply is present.
func hasDA1(s string) bool { return da1End(s) >= 0 }

// da1End returns the index just past a "CSI ? ... c" reply, or -1.
func da1End(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] != 0x1b || s[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
			j++
		}
		if j < len(s) && s[j] == 'c' {
			return j + 1
		}
	}
	return -1
}

// parseSync extracts the DECRQM answer for mode 2026 from a terminal reply.
// It returns 1 for supported, 0 for explicitly unsupported and -1 when the
// terminal said nothing, so a caller can tell "no" from "no answer".
func parseSync(reply string) int {
	for i := 0; i+1 < len(reply); i++ {
		if reply[i] != 0x1b || reply[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(reply) && (reply[j] < 0x40 || reply[j] > 0x7e) {
			j++
		}
		if j >= len(reply) || reply[j] != 'y' {
			continue
		}
		mode, ps, ok := parseDECRPM(reply[i+2 : j])
		if !ok || mode != 2026 {
			continue
		}
		switch ps {
		case 1, 3:
			// 3 is "permanently set"; both mean the terminal will honour it.
			return 1
		case 2, 4:
			return 0
		}
		return -1 // 0: mode not recognised
	}
	return -1
}

// parseDECRPM splits a DECRPM parameter string such as "?2026;1$" into the
// mode number and its status. The trailing '$' is the DECRQM intermediate
// byte; it arrives as part of the parameters because only the final byte
// ends the CSI sequence.
func parseDECRPM(params string) (mode, ps int, ok bool) {
	params = strings.TrimSuffix(strings.TrimPrefix(params, "?"), "$")
	parts := strings.Split(params, ";")
	if len(parts) != 2 {
		return 0, 0, false
	}
	m, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	p, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return m, p, true
}
