package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// view is the interface every screen implements.
//
// Each screen is a self-contained Bubble Tea sub-model. The root model
// owns a stack of views and dispatches Update / View calls to the
// active (top-of-stack) one. The interface is small on purpose —
// extra knobs are passed via tea.Msg, not via method parameters.
type view interface {
	Init() tea.Cmd
	Update(tea.Msg) (view, tea.Cmd)
	View(width, height int) string
	Title() string
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
	// Absorbing reports whether the view currently has an active text
	// input or modal that should swallow key presses (so global hotkeys
	// like "q", "j", "/" do not fire while the user types).
	Absorbing() bool
}
