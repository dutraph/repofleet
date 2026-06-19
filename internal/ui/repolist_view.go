package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/config"
	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
)

// repoListView is the home screen: every local git repo found on disk,
// with its provider icon, branch, status and duplicate marker. Repos
// can be multi-selected for bulk pull/fetch.
type repoListView struct {
	cfg      *config.Config
	repos    []scanner.Repo
	statuses map[string]gitops.Status
	selected map[string]bool // keyed by repo path

	tbl     table.Model
	w, h    int
	loading bool
	built   bool
}

func newRepoListView(cfg *config.Config) *repoListView {
	return &repoListView{
		cfg:      cfg,
		statuses: map[string]gitops.Status{},
		selected: map[string]bool{},
		loading:  true,
	}
}

// --- local messages ---

type scanDoneMsg struct {
	repos []scanner.Repo
	err   error
}
type statusMsg struct {
	path string
	st   gitops.Status
}
type gitResultMsg struct {
	res    gitops.Result
	action string
}

func scanCmd(roots []string) tea.Cmd {
	return func() tea.Msg {
		repos, err := scanner.Scan(roots, 7)
		return scanDoneMsg{repos: repos, err: err}
	}
}

func statusCmd(path string) tea.Cmd {
	return func() tea.Msg { return statusMsg{path: path, st: gitops.GetStatus(path)} }
}

func gitActionCmd(path, action string) tea.Cmd {
	return func() tea.Msg {
		var res gitops.Result
		switch action {
		case "pull":
			res = gitops.Pull(path)
		case "fetch":
			res = gitops.Fetch(path)
		}
		return gitResultMsg{res: res, action: action}
	}
}

func (v *repoListView) Init() tea.Cmd { return scanCmd(v.cfg.ScanRoots) }

func (v *repoListView) Title() string {
	if v.loading {
		return "scanning…"
	}
	dups := 0
	for _, r := range v.repos {
		if r.IsDuplicate() {
			dups++
		}
	}
	return fmt.Sprintf("repos · %d found · %d duplicates", len(v.repos), dups)
}

func (v *repoListView) Absorbing() bool { return false }

func (v *repoListView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil

	case scanDoneMsg:
		v.loading = false
		if msg.err != nil {
			return v, fail(msg.err)
		}
		v.repos = msg.repos
		v.rebuild()
		// kick off status loading for every repo
		cmds := make([]tea.Cmd, 0, len(v.repos))
		for _, r := range v.repos {
			cmds = append(cmds, statusCmd(r.Path))
		}
		return v, tea.Batch(cmds...)

	case statusMsg:
		v.statuses[msg.path] = msg.st
		v.rebuild()
		return v, nil

	case gitResultMsg:
		word := "ok"
		if msg.res.Err != nil {
			word = "failed"
		}
		// refresh status of the affected repo
		return v, tea.Batch(
			toast(fmt.Sprintf("%s %s: %s", msg.action, filepath.Base(msg.res.Path), word)),
			statusCmd(msg.res.Path),
		)

	case cloneDoneMsg:
		if msg.err != nil {
			return v, fail(fmt.Errorf("clone failed: %v", msg.err))
		}
		return v, tea.Batch(toast("cloned → "+msg.dest), scanCmd(v.cfg.ScanRoots))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keyToggle):
			if r := v.current(); r != nil {
				v.selected[r.Path] = !v.selected[r.Path]
				v.rebuild()
			}
			return v, nil
		case key.Matches(msg, keySelectAll):
			v.toggleAll()
			v.rebuild()
			return v, nil
		case key.Matches(msg, keyPull):
			return v, v.runOnTargets("pull")
		case key.Matches(msg, keyFetch):
			return v, v.runOnTargets("fetch")
		case key.Matches(msg, keyDetail):
			if r := v.current(); r != nil {
				return v, func() tea.Msg { return pushViewMsg{newDetailView(*r)} }
			}
			return v, nil
		case key.Matches(msg, keys.Remote):
			return v, func() tea.Msg { return pushViewMsg{newAccountView(v.cfg, v.repos)} }
		case key.Matches(msg, keys.Refresh):
			v.loading = true
			return v, scanCmd(v.cfg.ScanRoots)
		}
		// fast scroll J/K, then default table nav
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

func (v *repoListView) View(width, height int) string {
	if v.loading {
		return padView("\n  ⟳ scanning "+strings.Join(v.cfg.ScanRoots, ", ")+" …", width)
	}
	if len(v.repos) == 0 {
		return padView("\n  no git repositories found under:\n  "+
			strings.Join(v.cfg.ScanRoots, "\n  ")+
			"\n\n  edit scan_roots in the config, then press r to rescan.", width)
	}
	return padView(v.tbl.View(), width)
}

// --- helpers ---

func (v *repoListView) current() *scanner.Repo {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.repos) {
		return nil
	}
	return &v.repos[i]
}

func (v *repoListView) toggleAll() {
	all := true
	for _, r := range v.repos {
		if !v.selected[r.Path] {
			all = false
			break
		}
	}
	for _, r := range v.repos {
		v.selected[r.Path] = !all
	}
}

// targets returns the selected repos, or the cursor row when none are
// selected.
func (v *repoListView) targets() []scanner.Repo {
	var out []scanner.Repo
	for _, r := range v.repos {
		if v.selected[r.Path] {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		if r := v.current(); r != nil {
			out = append(out, *r)
		}
	}
	return out
}

func (v *repoListView) runOnTargets(action string) tea.Cmd {
	targets := v.targets()
	if len(targets) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(targets)+1)
	for _, r := range targets {
		cmds = append(cmds, gitActionCmd(r.Path, action))
	}
	cmds = append(cmds, toast(fmt.Sprintf("%s on %d repo(s)…", action, len(targets))))
	return tea.Batch(cmds...)
}

func (v *repoListView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	// Column widths. Keep TYPE/STATUS/BRANCH fixed; NAME and PATH flex.
	const (
		wSel    = 2
		wIcon   = 3
		wType   = 10
		wBranch = 16
		wStatus = 9
	)
	flex := v.w - (wSel + wIcon + wType + wBranch + wStatus) - cellPadding*7 - 2
	if flex < 20 {
		flex = 20
	}
	wName := flex/2 + 6
	wPath := flex - wName
	if wPath < 12 {
		wPath = 12
	}

	cols := []table.Column{
		{Title: "", Width: wSel},
		{Title: "", Width: wIcon},
		{Title: "TYPE", Width: wType},
		{Title: "NAME", Width: wName},
		{Title: "BRANCH", Width: wBranch},
		{Title: "STATUS", Width: wStatus},
		{Title: "PATH", Width: wPath},
	}

	rows := make([]table.Row, 0, len(v.repos))
	for _, r := range v.repos {
		meta := provider.Meta(r.Provider)
		sel := " "
		if v.selected[r.Path] {
			sel = "✓"
		}
		name := r.Name
		if r.IsDuplicate() {
			name += fmt.Sprintf(" ⧉%d/%d", r.DupIndex, r.DupCount)
		}
		branch, status := "…", "…"
		if st, ok := v.statuses[r.Path]; ok {
			branch, status = st.Branch, st.Symbol()
		}
		rows = append(rows, table.Row{
			sel, meta.Icon, meta.Name, name, branch, status, shortenPath(r.Path),
		})
	}

	cursor := 0
	height := v.h - 4
	if height < 3 {
		height = 3
	}
	if v.built {
		cursor = v.tbl.Cursor()
	}
	v.tbl = newStyledTable(cols, rows, height)
	v.built = true
	if cursor >= 0 && cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

// shortenPath replaces the home prefix with ~ for compactness.
func shortenPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// --- view-local key bindings ---

var (
	keyToggle    = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))
	keySelectAll = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all"))
	keyPull      = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pull"))
	keyFetch     = key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fetch"))
	keyDetail    = key.NewBinding(key.WithKeys("enter", "s"), key.WithHelp("enter", "details"))
)

func (v *repoListView) ShortHelp() []key.Binding {
	return []key.Binding{keyToggle, keyPull, keyFetch, keys.Remote, keyDetail}
}

func (v *repoListView) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keyToggle, keySelectAll, keyDetail},
		{keyPull, keyFetch, keys.Remote, keys.Refresh},
	}
}
