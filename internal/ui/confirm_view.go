package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/theme"
)

// confirmView is a modal yes/no prompt. On confirm it pops itself and
// runs onConfirm; on cancel it just pops. It Absorbs input so global
// hotkeys don't fire underneath.
type confirmView struct {
	title     string
	lines     []string
	onConfirm tea.Cmd
	w, h      int
}

func newConfirmView(title string, lines []string, onConfirm tea.Cmd) *confirmView {
	return &confirmView{title: title, lines: lines, onConfirm: onConfirm}
}

func (v *confirmView) Init() tea.Cmd   { return nil }
func (v *confirmView) Title() string   { return v.title }
func (v *confirmView) Absorbing() bool { return true }

func (v *confirmView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		return v, nil
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "y", "enter":
			return v, tea.Batch(func() tea.Msg { return popViewMsg{} }, v.onConfirm)
		case "n", "esc":
			return v, func() tea.Msg { return popViewMsg{} }
		}
	}
	return v, nil
}

func (v *confirmView) View(width, height int) string {
	var b strings.Builder
	b.WriteString("\n  " + theme.ErrorBox.Render(" "+v.title+" ") + "\n\n")
	for _, l := range v.lines {
		b.WriteString("  " + l + "\n")
	}
	b.WriteString("\n  " + theme.HelpKey.Render("y") + " " + theme.HelpDesc.Render("confirm") +
		"    " + theme.HelpKey.Render("n") + " " + theme.HelpDesc.Render("cancel"))
	return padView(b.String(), width)
}

func (v *confirmView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel")),
	}
}
func (v *confirmView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }
