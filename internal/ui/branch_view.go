package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/scanner"
)

// branchView lets the user pick a branch to check out for one repo.
type branchView struct {
	repo     scanner.Repo
	branches []gitops.Branch
	tbl      table.Model
	loading  bool
	w, h     int
	built    bool
}

func newBranchView(r scanner.Repo) *branchView {
	return &branchView{repo: r, loading: true}
}

type branchesMsg struct {
	branches []gitops.Branch
	err      error
}
type checkoutDoneMsg struct {
	path   string
	branch string
	err    error
}

func loadBranchesCmd(path string) tea.Cmd {
	return func() tea.Msg {
		b, err := gitops.Branches(path)
		return branchesMsg{branches: b, err: err}
	}
}

func checkoutCmd(path, branch string) tea.Cmd {
	return func() tea.Msg {
		res := gitops.Checkout(path, branch)
		return checkoutDoneMsg{path: path, branch: branch, err: res.Err}
	}
}

func (v *branchView) Init() tea.Cmd   { return loadBranchesCmd(v.repo.Path) }
func (v *branchView) Title() string   { return "branch · " + v.repo.Name }
func (v *branchView) Absorbing() bool { return false }

func (v *branchView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil

	case branchesMsg:
		v.loading = false
		if msg.err != nil {
			return v, fail(msg.err)
		}
		v.branches = msg.branches
		v.rebuild()
		return v, nil

	case checkoutDoneMsg:
		if msg.err != nil {
			return v, fail(fmt.Errorf("checkout %s: %v", msg.branch, msg.err))
		}
		return v, tea.Batch(
			func() tea.Msg { return popViewMsg{} },
			toast("switched to "+msg.branch),
			statusCmd(msg.path),
		)

	case tea.KeyMsg:
		if msg.String() == "enter" && !v.loading {
			if b := v.currentBranch(); b != nil {
				return v, checkoutCmd(v.repo.Path, b.CheckoutName())
			}
			return v, nil
		}
		if t, handled := applyFastTableScroll(v.tbl, msg); handled {
			v.tbl = t
			return v, nil
		}
		var cmd tea.Cmd
		v.tbl, cmd = v.tbl.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *branchView) currentBranch() *gitops.Branch {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.branches) {
		return nil
	}
	return &v.branches[i]
}

func (v *branchView) View(width, height int) string {
	if v.loading {
		return padView("\n  ⟳ reading branches…", width)
	}
	if len(v.branches) == 0 {
		return padView("\n  no branches found", width)
	}
	return padView(v.tbl.View(), width)
}

func (v *branchView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	nameW := v.w - 10 - 10 - cellPadding*3 - 2
	if nameW < 20 {
		nameW = 20
	}
	cols := []table.Column{
		{Title: "BRANCH", Width: nameW},
		{Title: "WHERE", Width: 10},
		{Title: "CURRENT", Width: 10},
	}
	rows := make([]table.Row, 0, len(v.branches))
	for _, b := range v.branches {
		where := "local"
		if b.Remote {
			where = "remote"
		}
		cur := ""
		if b.Current {
			cur = "★"
		}
		rows = append(rows, table.Row{b.Name, where, cur})
	}
	h := v.h - 4
	if h < 3 {
		h = 3
	}
	cursor := 0
	if v.built {
		cursor = v.tbl.Cursor()
	}
	v.tbl = newStyledTable(cols, rows, h)
	v.built = true
	if cursor >= 0 && cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *branchView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "checkout")),
		keys.Back,
	}
}
func (v *branchView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }
