// Command showcase launches the ASCII Animations TUI.
package main

import (
	"fmt"
	"os"

	"github.com/cyperx84/ascii-animations/pkg/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("ascii-animations %s\n", version)
		return
	}

	m := ui.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
