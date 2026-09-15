// Command showcase launches the ASCII Animations TUI.
package main

import (
	"fmt"
	"os"

	"github.com/cyperx84/ascii-animations/pkg/ui"

	tea "charm.land/bubbletea/v2"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("ascii-animations %s\n", version)
		return
	}

	// The alternate screen and mouse mode live on the View in v2, so the
	// program takes no options.
	p := tea.NewProgram(ui.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
