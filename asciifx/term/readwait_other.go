//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package term

import (
	"errors"
	"time"
)

// Without a way to ask whether a descriptor is ready, a read could block for
// as long as the terminal stays silent, so the probe reads nothing at all and
// the caller keeps the defaults it already had.
const canWaitReadable = false

var errNoReadiness = errors.New("term: no readiness check on this platform")

func waitReadable(fd int, d time.Duration) (bool, error) { return false, errNoReadiness }

func readReady(fd int, buf []byte) (int, error) { return 0, errNoReadiness }
