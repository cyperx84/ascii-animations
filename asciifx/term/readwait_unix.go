//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package term

import (
	"errors"
	"io"
	"time"

	"golang.org/x/sys/unix"
)

// canWaitReadable reports whether this platform can ask whether a descriptor
// has data without committing to a read that might never return.
const canWaitReadable = true

// errFDRange is a descriptor select cannot name. FdSet is a fixed bitmap of
// FD_SETSIZE bits, so Set on anything outside it indexes past the array; the
// caller gets an ordinary error and declines to probe, rather than a panic out
// of an exported function.
var errFDRange = errors.New("term: descriptor outside the select range")

// waitReadable reports whether fd has data to read within d.
//
// It uses select for the same reason watchKeys does -- macOS poll does not
// support terminal devices -- and in preference to an os.File read deadline,
// which is not an option at all here: the runtime poller only accepts a
// descriptor it opened in non-blocking mode, so SetReadDeadline on a terminal
// stdin fails with "file type does not support deadline" and a probe built on
// it gives up before reading a byte.
//
// EINTR restarts the wait against the same deadline, so a signal can neither
// cut the wait short nor extend it.
func waitReadable(fd int, d time.Duration) (bool, error) {
	if fd < 0 || fd >= unix.FD_SETSIZE {
		return false, errFDRange
	}
	deadline := time.Now().Add(d)
	for {
		left := time.Until(deadline)
		if left <= 0 {
			return false, nil
		}
		var set unix.FdSet
		set.Zero()
		set.Set(fd)
		tv := unix.NsecToTimeval(int64(left))
		n, err := unix.Select(fd+1, &set, nil, nil, &tv)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return false, err
		}
		return n > 0 && set.IsSet(fd), nil
	}
}

// readReady reads once from a descriptor select has just reported ready.
// EINTR and EAGAIN mean "nothing this time, nothing wrong", so they come back
// as no bytes and no error and the caller waits again; a ready descriptor
// that yields nothing has reached end of input, which is io.EOF and final.
func readReady(fd int, buf []byte) (int, error) {
	n, err := unix.Read(fd, buf)
	switch {
	case errors.Is(err, unix.EINTR), errors.Is(err, unix.EAGAIN):
		return 0, nil
	case err != nil:
		return 0, err
	case n <= 0:
		return 0, io.EOF
	}
	return n, nil
}
