package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Search   key.Binding
	Filter   key.Binding
	Sort     key.Binding
	Edit     key.Binding
	NextPage key.Binding
	PrevPage key.Binding
	Refresh  key.Binding
	Open     key.Binding
	Help     key.Binding
	Quit     key.Binding
	Tree     key.Binding
	Expand   key.Binding
	Collapse key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev page"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Tree: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "tree toggle"),
	),
	Expand: key.NewBinding(
		key.WithKeys("right", ">"),
		key.WithHelp(">/→", "expand"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("left", "<"),
		key.WithHelp("</←", "collapse"),
	),
}
