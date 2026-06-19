// Package ui hosts the Bubble Tea application. The root Model
// coordinates a stack of views, a transient toast, the help overlay,
// and the header/footer chrome. Each screen lives in its own *_view.go
// file and implements the view interface.
package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dutraph/repofleet/internal/config"
	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/theme"
	"github.com/dutraph/repofleet/internal/version"
)

// Model is the root Bubble Tea model.
type Model struct {
	cfg   *config.Config
	stack []view

	width, height int

	toast      string
	toastUntil time.Time

	showHelp bool
	quitting bool
}

// New builds the root model with the repo-list view on the stack.
func New(cfg *config.Config) Model {
	return Model{
		cfg:   cfg,
		stack: []view{newRepoListView(cfg)},
	}
}

func (m Model) top() view { return m.stack[len(m.stack)-1] }

func (m Model) Init() tea.Cmd { return m.top().Init() }

// Update is the root reducer. It intercepts global keys + stack/toast
// control messages, and forwards everything else to the active view.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, m.forward(msg)

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if !m.top().Absorbing() {
			switch {
			case key.Matches(msg, keys.Quit):
				m.quitting = true
				return m, tea.Quit
			case key.Matches(msg, keys.Help):
				m.showHelp = true
				return m, nil
			case key.Matches(msg, keys.Home):
				if len(m.stack) > 1 {
					m.stack = m.stack[:1]
					return m, tea.Batch(tea.ClearScreen, m.syncSize())
				}
			case key.Matches(msg, keys.Back):
				if len(m.stack) > 1 {
					m.stack = m.stack[:len(m.stack)-1]
					return m, tea.Batch(tea.ClearScreen, m.syncSize())
				}
			}
		}
		return m, m.forward(msg)

	case pushViewMsg:
		m.stack = append(m.stack, msg.v)
		return m, tea.Batch(tea.ClearScreen, m.top().Init(), m.syncSize())

	case popViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, tea.Batch(tea.ClearScreen, m.syncSize())

	case switchViewMsg:
		m.stack = []view{msg.v}
		return m, tea.Batch(tea.ClearScreen, m.top().Init(), m.syncSize())

	case toastMsg:
		m.toast = msg.text
		m.toastUntil = time.Now().Add(3 * time.Second)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearToastMsg{} })

	case clearToastMsg:
		if time.Now().After(m.toastUntil) {
			m.toast = ""
		}
		return m, nil

	case errMsg:
		m.toast = "⚠ " + msg.Error()
		m.toastUntil = time.Now().Add(5 * time.Second)
		return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return clearToastMsg{} })

	case execCloneMsg:
		cmd := gitops.CloneCommand(msg.url, msg.dest)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return cloneDoneMsg{dest: msg.dest, err: err}
		})

	case execGitMsg:
		cmd := gitops.CommandLine(msg.path, msg.args)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return gitExecDoneMsg{path: msg.path, args: msg.args, err: err}
		})
	}

	// Everything else (scan results, status updates, git results, clone
	// completion, …) goes to the active view.
	return m, m.forward(msg)
}

// forward dispatches a message to the active view and stores the
// returned view back on the stack.
func (m *Model) forward(msg tea.Msg) tea.Cmd {
	v, cmd := m.top().Update(msg)
	m.stack[len(m.stack)-1] = v
	return cmd
}

// syncSize re-sends the current window size so a newly-active view can
// lay out its table.
func (m Model) syncSize() tea.Cmd {
	w, h := m.width, m.height
	return func() tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "loading…"
	}
	if m.showHelp {
		return m.helpView()
	}

	header := m.headerBar()
	footer := m.footerBar()
	bodyH := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyH < 1 {
		bodyH = 1
	}
	body := m.top().View(m.width, bodyH)
	body = lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) headerBar() string {
	left := theme.Logo.Render(" ▲ fleet ") + theme.Faint.Render(version.String())
	title := theme.Title.Render(" " + m.top().Title() + " ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(title)
	if gap < 1 {
		gap = 1
	}
	row := left + strings.Repeat(" ", gap) + title
	return theme.HeaderBar.Width(m.width).Render(row)
}

func (m Model) footerBar() string {
	if m.toast != "" {
		return theme.Toast.Width(m.width).Render(m.toast)
	}
	var parts []string
	for _, b := range m.top().ShortHelp() {
		if b.Help().Key == "" {
			continue
		}
		parts = append(parts, theme.HelpKey.Render(b.Help().Key)+" "+theme.HelpDesc.Render(b.Help().Desc))
	}
	parts = append(parts, theme.HelpKey.Render("?")+" "+theme.HelpDesc.Render("help"))
	return theme.StatusBar.Width(m.width).Render(strings.Join(parts, theme.Faint.Render("  •  ")))
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString(theme.Title.Render(" fleet — help ") + "\n\n")

	// Collect view-specific bindings (FullHelp already supersets
	// ShortHelp), then the global ones, de-duplicating by key so the
	// overlay never lists the same shortcut twice.
	rows := append([][]key.Binding{}, m.top().FullHelp()...)
	rows = append(rows, []key.Binding{keys.Home, keys.Back, keys.Help, keys.Quit})

	seen := map[string]bool{}
	for _, row := range rows {
		for _, bind := range row {
			k := bind.Help().Key
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			b.WriteString("  " + theme.HelpKey.Render(pad(k, 10)) + theme.HelpDesc.Render(bind.Help().Desc) + "\n")
		}
	}
	b.WriteString("\n" + theme.Faint.Render("  press any key to close"))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func pad(s string, n int) string {
	for lipgloss.Width(s) < n {
		s += " "
	}
	return s
}
