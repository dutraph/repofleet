package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/config"
	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
)

// repoListView is the home screen: every local git repo found on disk,
// with its provider icon, branch, status and duplicate marker. Repos
// can be multi-selected for bulk actions, searched with `/`, and
// narrowed to a single provider with `t`.
type repoListView struct {
	cfg      *config.Config
	repos    []scanner.Repo
	statuses map[string]gitops.Status
	selected map[string]bool // keyed by repo path

	filter    textinput.Model
	filtering bool

	// provider filter: provSel 0 = all; otherwise provCycle[provSel-1].
	provCycle []provider.Kind
	provSel   int

	filtered []int // indices into v.repos currently shown
	tbl      table.Model
	w, h     int
	loading  bool
	built    bool
}

// groupOrder is the fixed provider order used by the `t` type filter.
var groupOrder = []provider.Kind{
	provider.GitHub, provider.GitLab, provider.AzureDevOps,
	provider.Bitbucket, provider.Local, provider.Unknown,
}

func newRepoListView(cfg *config.Config) *repoListView {
	fi := textinput.New()
	fi.Placeholder = "filter by name, path or provider…"
	fi.Prompt = "/"

	return &repoListView{
		cfg:      cfg,
		statuses: map[string]gitops.Status{},
		selected: map[string]bool{},
		filter:   fi,
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
type repoDeletedMsg struct {
	path string
	err  error
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
		case "pullprune":
			res = gitops.PullPrune(path)
		case "fetch":
			res = gitops.Fetch(path)
		}
		return gitResultMsg{res: res, action: action}
	}
}

func deleteRepoCmd(path string) tea.Cmd {
	return func() tea.Msg {
		err := os.RemoveAll(path)
		return repoDeletedMsg{path: path, err: err}
	}
}

func (v *repoListView) Init() tea.Cmd { return scanCmd(v.cfg.ScanRoots) }

func (v *repoListView) Title() string {
	if v.loading {
		return "scanning…"
	}
	// Counts reflect the active filter (provider + search). v.filtered is
	// the currently-shown subset; it equals all repos when no filter is on.
	idx := v.filtered
	if !v.built {
		idx = nil // before the first rebuild, fall back to totals below
	}
	found, dups := len(v.repos), 0
	if v.built {
		found = len(idx)
		for _, i := range idx {
			if v.repos[i].IsDuplicate() {
				dups++
			}
		}
	} else {
		for _, r := range v.repos {
			if r.IsDuplicate() {
				dups++
			}
		}
	}
	base := fmt.Sprintf("repos · %d found · %d dup", found, dups)
	if v.provSel > 0 && v.provSel-1 < len(v.provCycle) {
		base += " · " + provider.Meta(v.provCycle[v.provSel-1]).Name
	}
	if q := strings.TrimSpace(v.filter.Value()); q != "" {
		base += " · /" + q
	}
	return base
}

func (v *repoListView) Absorbing() bool { return v.filtering }

// SelectedRepo lets the root `:` command bar target the cursor repo.
func (v *repoListView) SelectedRepo() (string, string, bool) {
	if r := v.current(); r != nil {
		return r.Path, r.Name, true
	}
	return "", "", false
}

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
		v.computeProvCycle()
		v.rebuild()
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
		return v, tea.Batch(
			toast(fmt.Sprintf("%s %s: %s", msg.action, filepath.Base(msg.res.Path), word)),
			statusCmd(msg.res.Path),
		)

	case repoDeletedMsg:
		if msg.err != nil {
			return v, fail(fmt.Errorf("remove failed: %v", msg.err))
		}
		delete(v.selected, msg.path)
		return v, tea.Batch(toast("removed "+filepath.Base(msg.path)), scanCmd(v.cfg.ScanRoots))

	case cloneDoneMsg:
		if msg.err != nil {
			return v, fail(fmt.Errorf("clone failed: %v", msg.err))
		}
		return v, tea.Batch(toast("cloned → "+msg.dest), scanCmd(v.cfg.ScanRoots))

	case gitExecDoneMsg:
		// Root shows the toast; we just refresh this repo's status.
		return v, statusCmd(msg.path)

	case tea.KeyMsg:
		if v.filtering {
			switch msg.String() {
			case "esc":
				v.filtering = false
				v.filter.Blur()
				v.filter.SetValue("")
				v.rebuild()
				return v, nil
			case "enter":
				v.filtering = false
				v.filter.Blur()
				return v, nil
			}
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			v.rebuild()
			return v, cmd
		}

		switch {
		case key.Matches(msg, keyFilter):
			v.filtering = true
			v.filter.Focus()
			return v, textinput.Blink
		case key.Matches(msg, keyTypeFilter):
			v.provSel = (v.provSel + 1) % (len(v.provCycle) + 1)
			v.rebuild()
			return v, nil
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
		case key.Matches(msg, keyPullPrune):
			return v, v.runOnTargets("pullprune")
		case key.Matches(msg, keyFetch):
			return v, v.runOnTargets("fetch")
		case key.Matches(msg, keyFetchAll):
			return v, v.runOnAll("fetch")
		case key.Matches(msg, keyBranch):
			if r := v.current(); r != nil {
				rr := *r
				return v, func() tea.Msg { return pushViewMsg{newBranchView(rr)} }
			}
			return v, nil
		case key.Matches(msg, keyDelete):
			if r := v.current(); r != nil {
				path := r.Path
				cv := newConfirmView("remove repository?", []string{
					"Permanently deletes this working copy from disk:",
					"  " + shortenPath(path),
					"Unpushed commits or stashes will be lost.",
				}, deleteRepoCmd(path))
				return v, func() tea.Msg { return pushViewMsg{cv} }
			}
			return v, nil
		case key.Matches(msg, keyDetail):
			if r := v.current(); r != nil {
				return v, func() tea.Msg { return pushViewMsg{newDetailView(*r)} }
			}
			return v, nil
		case key.Matches(msg, keyDuplicates):
			return v, func() tea.Msg { return pushViewMsg{newDuplicatesView(v.repos)} }
		case key.Matches(msg, keys.Remote):
			return v, func() tea.Msg { return pushViewMsg{newAccountView(v.cfg, v.repos)} }
		case key.Matches(msg, keys.Refresh):
			v.loading = true
			return v, scanCmd(v.cfg.ScanRoots)
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

func (v *repoListView) View(width, height int) string {
	if v.loading {
		return padView("\n  ⟳ scanning "+strings.Join(v.cfg.ScanRoots, ", ")+" …", width)
	}
	if len(v.repos) == 0 {
		return padView("\n  no git repositories found under:\n  "+
			strings.Join(v.cfg.ScanRoots, "\n  ")+
			"\n\n  edit scan_roots in the config, then press r to rescan.", width)
	}
	body := v.tbl.View()
	if v.filtering || v.filter.Value() != "" {
		body = v.filter.View() + "\n" + body
	}
	return padView(body, width)
}

// --- helpers ---

func (v *repoListView) filterShown() bool {
	return v.filtering || v.filter.Value() != ""
}

// computeProvCycle records which providers are actually present so the
// `t` filter only cycles through meaningful options.
func (v *repoListView) computeProvCycle() {
	present := map[provider.Kind]bool{}
	for _, r := range v.repos {
		present[r.Provider] = true
	}
	v.provCycle = v.provCycle[:0]
	for _, k := range groupOrder {
		if present[k] {
			v.provCycle = append(v.provCycle, k)
		}
	}
	if v.provSel > len(v.provCycle) {
		v.provSel = 0
	}
}

func (v *repoListView) current() *scanner.Repo {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.filtered) {
		return nil
	}
	return &v.repos[v.filtered[i]]
}

// toggleAll selects (or clears) every currently visible repo.
func (v *repoListView) toggleAll() {
	all := len(v.filtered) > 0
	for _, i := range v.filtered {
		if !v.selected[v.repos[i].Path] {
			all = false
			break
		}
	}
	for _, i := range v.filtered {
		v.selected[v.repos[i].Path] = !all
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
	return v.runOn(v.targets(), action)
}

func (v *repoListView) runOnAll(action string) tea.Cmd {
	return v.runOn(v.repos, action)
}

func (v *repoListView) runOn(targets []scanner.Repo, action string) tea.Cmd {
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

// matches reports whether a repo passes the text + provider filters.
func (v *repoListView) matches(r scanner.Repo, q string) bool {
	if v.provSel > 0 && v.provSel-1 < len(v.provCycle) {
		if r.Provider != v.provCycle[v.provSel-1] {
			return false
		}
	}
	if q == "" {
		return true
	}
	hay := strings.ToLower(r.Name + " " + r.Path + " " + provider.Meta(r.Provider).Name)
	return strings.Contains(hay, q)
}

func (v *repoListView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	q := strings.ToLower(strings.TrimSpace(v.filter.Value()))
	v.filtered = v.filtered[:0]
	for i, r := range v.repos {
		if v.matches(r, q) {
			v.filtered = append(v.filtered, i)
		}
	}

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

	rows := make([]table.Row, 0, len(v.filtered))
	for _, i := range v.filtered {
		r := v.repos[i]
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

	height := v.h - 4
	if v.filterShown() {
		height--
	}
	if height < 3 {
		height = 3
	}
	cursor := 0
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
	keyToggle     = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))
	keySelectAll  = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all"))
	keyPull       = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pull (ff-only)"))
	keyPullPrune  = key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pull --prune"))
	keyFetch      = key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fetch"))
	keyFetchAll   = key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "fetch all (sync)"))
	keyBranch     = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branch"))
	keyDelete     = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove"))
	keyDetail     = key.NewBinding(key.WithKeys("enter", "s"), key.WithHelp("enter", "details"))
	keyFilter     = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search"))
	keyTypeFilter = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "filter by type"))
	keyCmd        = key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "git command"))
	keyDuplicates = key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "duplicates"))
)

func (v *repoListView) ShortHelp() []key.Binding {
	return []key.Binding{keyFilter, keyTypeFilter, keyCmd, keyToggle, keyPull, keys.Remote}
}

func (v *repoListView) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keyFilter, keyTypeFilter, keyCmd, keyToggle, keySelectAll, keyDetail},
		{keyPull, keyPullPrune, keyFetch, keyFetchAll},
		{keyBranch, keyDelete, keyDuplicates, keys.Remote, keys.Refresh},
	}
}
