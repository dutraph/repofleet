package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
)

// detailView shows one repo's metadata plus a live `git status -sb`.
type detailView struct {
	repo scanner.Repo
	vp   viewport.Model
	body string
	w, h int
}

func newDetailView(r scanner.Repo) *detailView {
	return &detailView{repo: r}
}

type detailStatusMsg struct{ res gitops.Result }

func (v *detailView) Init() tea.Cmd {
	r := v.repo
	return func() tea.Msg { return detailStatusMsg{gitops.StatusText(r.Path)} }
}

func (v *detailView) Title() string { return "details · " + v.repo.Name }

func (v *detailView) Absorbing() bool { return false }

func (v *detailView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.vp = viewport.New(msg.Width-4, msg.Height-4)
		v.vp.SetContent(v.render())
		return v, nil
	case detailStatusMsg:
		out := msg.res.Output
		if msg.res.Err != nil && out == "" {
			out = "git status error"
		}
		v.body = out
		if v.w > 0 {
			v.vp.SetContent(v.render())
		}
		return v, nil
	}
	var cmd tea.Cmd
	v.vp, cmd = v.vp.Update(msg)
	return v, cmd
}

func (v *detailView) render() string {
	meta := provider.Meta(v.repo.Provider)
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n", meta.Icon, v.repo.Name)
	fmt.Fprintf(&b, "provider : %s\n", meta.Name)
	fmt.Fprintf(&b, "path     : %s\n", v.repo.Path)
	remote := v.repo.RemoteURL
	if remote == "" {
		remote = "(local only — no remote)"
	}
	fmt.Fprintf(&b, "remote   : %s\n", remote)
	if v.repo.IsDuplicate() {
		fmt.Fprintf(&b, "duplicate: copy %d of %d (same remote cloned elsewhere)\n", v.repo.DupIndex, v.repo.DupCount)
	}
	b.WriteString("\n── git status ──\n")
	if v.body == "" {
		b.WriteString("loading…")
	} else {
		b.WriteString(v.body)
	}
	return b.String()
}

func (v *detailView) View(width, height int) string {
	if v.w == 0 {
		return padView("\n  loading…", width)
	}
	return padView(v.vp.View(), width)
}

func (v *detailView) ShortHelp() []key.Binding {
	return []key.Binding{keys.Back}
}
func (v *detailView) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keys.Back}}
}
