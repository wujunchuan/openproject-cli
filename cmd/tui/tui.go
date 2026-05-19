package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/printer"
	"github.com/opf/openproject-cli/components/requests"
	"github.com/spf13/cobra"
)

var TuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI for work packages",
	Run:   runTui,
}

func runTui(_ *cobra.Command, _ []string) {
	requests.SetSilent(true)
	defer requests.SetSilent(false)

	p := tea.NewProgram(
		NewApp(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		printer.ErrorText(fmt.Sprintf("TUI error: %v", err))
	}
}
