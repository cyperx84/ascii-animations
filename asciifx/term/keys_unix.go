//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package term

import (
	"context"
	"errors"

	"golang.org/x/sys/unix"
)

const canWatchKeys = true

// pollInterval bounds how long the watcher takes to notice cancellation.
const pollInterval = 25 // milliseconds

// watchKeys waits for input on fd and sends on keys when a quit key arrives.
// It uses select rather than poll because macOS poll does not support
// terminal devices. It never blocks in read: it reads only after select
// reports data and checks ctx between waits, so it exits promptly once ctx
// is cancelled and leaves unread input for whoever owns the terminal next.
// The returned channel is closed when the watcher has exited.
func watchKeys(ctx context.Context, fd int, keys chan<- struct{}) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		for ctx.Err() == nil {
			var set unix.FdSet
			set.Zero()
			set.Set(fd)
			tv := unix.NsecToTimeval(int64(pollInterval) * 1e6)
			n, err := unix.Select(fd+1, &set, nil, nil, &tv)
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if err != nil {
				return
			}
			if n == 0 || !set.IsSet(fd) || ctx.Err() != nil {
				continue
			}
			m, err := unix.Read(fd, buf)
			if errors.Is(err, unix.EINTR) || errors.Is(err, unix.EAGAIN) {
				continue
			}
			if err != nil || m == 0 {
				return
			}
			for _, b := range buf[:m] {
				if isQuitKey(b) {
					select {
					case keys <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()
	return done
}
