package ui

import "github.com/charmbracelet/bubbles/key"

// globalKeys are handled by the root model regardless of the active
// view (unless the view is Absorbing key input for a text field).
type globalKeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Remote  key.Binding
	Home    key.Binding
}

var keys = globalKeyMap{
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Remote:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clone from server")),
	Home:    key.NewBinding(key.WithKeys("0"), key.WithHelp("0", "home")),
}
