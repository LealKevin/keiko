package main

import (
	"fmt"
	"os"

	"github.com/LealKevin/keiko/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tuiModel := tui.New()
	if _, err := tea.NewProgram(tuiModel, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
