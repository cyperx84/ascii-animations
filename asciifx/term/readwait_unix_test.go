//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package term

import (
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestWaitReadableRejectsDescriptorsSelectCannotName pins the bounds check.
// FdSet is a fixed bitmap, so Set past FD_SETSIZE indexes past its array --
// a panic out of the exported Probe, for a process that simply has a lot of
// files open. It has to come back as an ordinary error instead.
func TestWaitReadableRejectsDescriptorsSelectCannotName(t *testing.T) {
	for _, fd := range []int{-1, unix.FD_SETSIZE, unix.FD_SETSIZE + 1} {
		ready, err := func() (ready bool, err error) {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("waitReadable(%d) panicked: %v", fd, p)
				}
			}()
			return waitReadable(fd, time.Millisecond)
		}()
		if err == nil {
			t.Errorf("waitReadable(%d) returned no error; FD_SETSIZE is %d", fd, unix.FD_SETSIZE)
		}
		if ready {
			t.Errorf("waitReadable(%d) reported ready", fd)
		}
	}
	// The descriptor one below the ceiling is in range, so the check bounds
	// the bitmap rather than just refusing anything large.
	if _, err := waitReadable(unix.FD_SETSIZE-1, time.Millisecond); err == errFDRange {
		t.Errorf("waitReadable(%d) was rejected as out of range", unix.FD_SETSIZE-1)
	}
}

// TestProbeRawFailsSoftOnAnUnnameableDescriptor is the same guard seen from
// the outside: the probe declines and reports unknown, and nothing panics.
func TestProbeRawFailsSoftOnAnUnnameableDescriptor(t *testing.T) {
	in := os.NewFile(uintptr(unix.FD_SETSIZE+1), "beyond-fd-setsize")
	out, err := os.CreateTemp(t.TempDir(), "probe")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	got := probeRaw(in, out, 50*time.Millisecond)
	if got.SyncKnown {
		t.Errorf("SyncKnown = true from a descriptor that was never read")
	}
	if !got.Animate {
		t.Error("probeRaw must not turn animation off")
	}
}
