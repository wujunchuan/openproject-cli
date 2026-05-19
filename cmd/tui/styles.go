package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	subtleColor    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	specialColor   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	errorColor     = lipgloss.Color("#F25D94")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(highlightColor).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(highlightColor).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDD"))

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottomForeground(subtleColor)

	// Layout
	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlightColor).
				Padding(1, 2)
)

// statusColorStyle returns a lipgloss style with the given hex color foreground.
// Falls back to gold (#FFD700) if the color string is empty.
func statusColorStyle(hex string) lipgloss.Style {
	if hex == "" {
		hex = "#FFD700"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}
