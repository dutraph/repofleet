package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Generic messages shared by views. Keep this list tight — only add
// here when a message is consumed by more than one view (or the root
// model). Per-view messages should be private to that view's file.

// errMsg surfaces an error to the root model so it can show it in a toast.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// toastMsg shows a transient confirmation at the bottom.
type toastMsg struct {
	text  string
	until time.Time
}

// clearToastMsg is delivered after a delay to clear an existing toast.
type clearToastMsg struct{}

// View-stack control.
type pushViewMsg struct{ v view }
type popViewMsg struct{}
type switchViewMsg struct{ v view }
type sectionSwitchMsg struct{ v view }

// tickMsg is the periodic refresh signal.
type tickMsg time.Time

// execCloneMsg asks the root model to suspend the TUI and run
// `git clone url dest` so the user sees git's live progress.
type execCloneMsg struct {
	url  string
	dest string
}

// cloneDoneMsg is delivered after the clone subprocess exits.
type cloneDoneMsg struct {
	dest string
	err  error
}

// execGitMsg asks the root model to suspend the TUI and run an arbitrary
// `git` command (entered via the `:` command bar) against one repo, so
// interactive commands like commit/rebase work and the user sees the
// full output.
type execGitMsg struct {
	path string
	args string // raw git arguments, e.g. "log --oneline -10"
}

// gitExecDoneMsg is delivered after the `:` git command exits.
type gitExecDoneMsg struct {
	path string
	args string
	err  error
}

// toast is a small helper to fire a confirmation toast from any view.
func toast(text string) tea.Cmd {
	return func() tea.Msg { return toastMsg{text: text} }
}

// fail wraps an error into an errMsg command.
func fail(err error) tea.Cmd {
	return func() tea.Msg { return errMsg{err} }
}
