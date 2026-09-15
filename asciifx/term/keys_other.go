//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package term

import "context"

// Without a pollable terminal the player does not read keys at all, so it can
// never strip input from a host; SIGINT still stops playback.
const canWatchKeys = false

func watchKeys(ctx context.Context, fd int, keys chan<- struct{}) <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}
