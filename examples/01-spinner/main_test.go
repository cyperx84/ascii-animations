package main

import (
	"strings"
	"testing"
)

// The point of this example is that the upstream call shape still works, so
// the test is that all three spinners put their text on screen. The shimmer
// one colours every rune separately, which is why the check is against the
// stripped view rather than the styled one.
func TestRenderShowsThreeSpinners(t *testing.T) {
	out := stripANSI(newModel().render())
	for _, want := range []string{"working", "resolving hosts", "uploading layers"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q:\n%s", want, out)
		}
	}
}

// stripANSI drops SGR sequences, which is all these views carry.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
