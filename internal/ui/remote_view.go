package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dutraph/repofleet/internal/api"
	"github.com/dutraph/repofleet/internal/config"
	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
	"github.com/dutraph/repofleet/internal/theme"
)

// protoChip renders one protocol option as a bordered box. The active
// one gets a solid accent fill; the inactive one is dimmed. Both share
// the same border so they line up when joined horizontally.
func protoChip(label string, active bool) string {
	st := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	if active {
		return st.
			Foreground(theme.ColorBg).
			Background(theme.ColorPrimary).
			BorderForeground(theme.ColorPrimary).
			Bold(true).
			Render("● " + label)
	}
	return st.
		Foreground(theme.ColorMuted).
		BorderForeground(theme.ColorBorder).
		Render("○ " + label)
}

// ───────────────────────── account picker ─────────────────────────

// accountView lists the configured git-server accounts. Selecting one
// opens the remote repo browser for it.
type accountView struct {
	cfg   *config.Config
	local []scanner.Repo
	tbl   table.Model
	w, h  int
	built bool
}

func newAccountView(cfg *config.Config, local []scanner.Repo) *accountView {
	return &accountView{cfg: cfg, local: local}
}

func (v *accountView) Init() tea.Cmd   { return nil }
func (v *accountView) Title() string   { return "clone from server · pick account" }
func (v *accountView) Absorbing() bool { return false }

func (v *accountView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "enter" && len(v.cfg.Accounts) > 0 {
			i := v.tbl.Cursor()
			if i >= 0 && i < len(v.cfg.Accounts) {
				acct := v.cfg.Accounts[i]
				return v, func() tea.Msg { return pushViewMsg{newRemoteListView(v.cfg, acct, v.local)} }
			}
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

func (v *accountView) View(width, height int) string {
	if len(v.cfg.Accounts) == 0 {
		return padView("\n  no git-server accounts configured.\n\n"+
			"  run `fleet login` in your shell to connect GitHub, GitLab,\n"+
			"  Azure DevOps or Bitbucket with a PAT, then come back here.", width)
	}
	return padView(v.tbl.View(), width)
}

func (v *accountView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	detailW := v.w - 24 - 16 - cellPadding*3 - 2
	if detailW < 10 {
		detailW = 10
	}
	cols := []table.Column{
		{Title: "ACCOUNT", Width: 24},
		{Title: "PROVIDER", Width: 16},
		{Title: "DETAIL", Width: detailW},
	}
	rows := make([]table.Row, 0, len(v.cfg.Accounts))
	for _, a := range v.cfg.Accounts {
		detail := a.BaseURL
		if a.Org != "" {
			detail = "org: " + a.Org
		}
		if a.Username != "" {
			detail = "user: " + a.Username
		}
		mark := a.Name
		if a.Name == v.cfg.Active {
			mark += " ★"
		}
		rows = append(rows, table.Row{mark, a.Provider, detail})
	}
	cursor := 0
	if v.built {
		cursor = v.tbl.Cursor()
	}
	h := v.h - 4
	if h < 3 {
		h = 3
	}
	v.tbl = newStyledTable(cols, rows, h)
	v.built = true
	if cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *accountView) ShortHelp() []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "browse repos")), keys.Back}
}
func (v *accountView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }

// ──────────────────────── remote repo browser ────────────────────────

type remoteListView struct {
	cfg     *config.Config
	acct    config.Account
	local   []scanner.Repo
	remotes []api.RemoteRepo
	view    []int // indices into remotes after filtering

	filter    textinput.Model
	filtering bool
	tbl       table.Model
	loading   bool
	w, h      int
	built     bool
}

func newRemoteListView(cfg *config.Config, acct config.Account, local []scanner.Repo) *remoteListView {
	fi := textinput.New()
	fi.Placeholder = "filter…"
	fi.Prompt = "/"
	return &remoteListView{cfg: cfg, acct: acct, local: local, filter: fi, loading: true}
}

type remotesMsg struct {
	repos []api.RemoteRepo
	err   error
}

func listRemotesCmd(acct config.Account) tea.Cmd {
	return func() tea.Msg {
		client, err := api.New(acct)
		if err != nil {
			return remotesMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		repos, err := client.List(ctx)
		return remotesMsg{repos: repos, err: err}
	}
}

func (v *remoteListView) Init() tea.Cmd   { return listRemotesCmd(v.acct) }
func (v *remoteListView) Title() string {
	if v.loading {
		return "loading " + v.acct.Name + "…"
	}
	return fmt.Sprintf("%s · %d repos", v.acct.Name, len(v.remotes))
}
func (v *remoteListView) Absorbing() bool { return v.filtering }

func (v *remoteListView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil

	case remotesMsg:
		v.loading = false
		if msg.err != nil {
			return v, fail(msg.err)
		}
		v.remotes = msg.repos
		v.applyFilter()
		v.rebuild()
		return v, nil

	case tea.KeyMsg:
		if v.filtering {
			switch msg.String() {
			case "esc":
				v.filtering = false
				v.filter.Blur()
				v.filter.SetValue("")
				v.applyFilter()
				v.rebuild()
				return v, nil
			case "enter":
				v.filtering = false
				v.filter.Blur()
				return v, nil
			}
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			v.applyFilter()
			v.rebuild()
			return v, cmd
		}
		switch msg.String() {
		case "/":
			v.filtering = true
			v.filter.Focus()
			return v, textinput.Blink
		case "enter":
			if r := v.current(); r != nil {
				return v, func() tea.Msg { return pushViewMsg{newClonePathView(v.cfg, *r, v.local)} }
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

func (v *remoteListView) current() *api.RemoteRepo {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.view) {
		return nil
	}
	r := v.remotes[v.view[i]]
	return &r
}

func (v *remoteListView) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(v.filter.Value()))
	v.view = v.view[:0]
	for i, r := range v.remotes {
		if q == "" || strings.Contains(strings.ToLower(r.FullName), q) {
			v.view = append(v.view, i)
		}
	}
}

func (v *remoteListView) View(width, height int) string {
	if v.loading {
		return padView("\n  ⟳ fetching repositories from "+v.acct.Name+" …", width)
	}
	body := v.tbl.View()
	if v.filtering || v.filter.Value() != "" {
		body = v.filter.View() + "\n" + body
	}
	return padView(body, width)
}

func (v *remoteListView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	const wVis = 9
	nameW := (v.w - wVis - cellPadding*3 - 2) * 5 / 9
	if nameW < 20 {
		nameW = 20
	}
	descW := v.w - wVis - nameW - cellPadding*3 - 2
	if descW < 10 {
		descW = 10
	}
	cols := []table.Column{
		{Title: "REPOSITORY", Width: nameW},
		{Title: "VIS", Width: wVis},
		{Title: "DESCRIPTION", Width: descW},
	}
	localKeys := map[string]bool{}
	for _, lr := range v.local {
		if lr.Key != "" {
			localKeys[lr.Key] = true
		}
	}
	rows := make([]table.Row, 0, len(v.view))
	for _, idx := range v.view {
		r := v.remotes[idx]
		name := r.FullName
		if localKeys[provider.Normalize(r.CloneURL)] {
			name += " ✓cloned" // already present locally
		}
		vis := "public"
		if r.Private {
			vis = "private"
		}
		rows = append(rows, table.Row{name, vis, oneLine(r.Description)})
	}
	cursor := 0
	if v.built {
		cursor = v.tbl.Cursor()
	}
	h := v.h - 5
	if h < 3 {
		h = 3
	}
	v.tbl = newStyledTable(cols, rows, h)
	v.built = true
	if cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *remoteListView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "clone")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		keys.Back,
	}
}
func (v *remoteListView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }

// ──────────────────────── clone destination ────────────────────────

// browse item kinds.
const (
	itemClone     = iota // confirm: clone into the current directory
	itemNewFolder        // create a new folder here
	itemUp               // ".." — go to parent
	itemDir              // a subdirectory to descend into
)

type browseItem struct {
	kind int
	name string // subdir name (for itemDir)
}

// clonePathView is a filesystem browser: the user navigates directories
// with a live filter (type to narrow, ↑/↓ to move, Enter to descend),
// picks where to drop the clone, and toggles HTTPS/SSH with Tab.
type clonePathView struct {
	cfg      *config.Config
	repo     api.RemoteRepo
	dupPaths []string
	protocol string // "https" or "ssh"
	hasSSH   bool

	dir     string   // directory currently being browsed
	entries []string // subdir names of dir (unfiltered)
	items   []browseItem
	filter  textinput.Model
	newDir  textinput.Model // "new folder" name input
	creating bool
	tbl     table.Model
	w, h    int
	built   bool
}

func newClonePathView(cfg *config.Config, repo api.RemoteRepo, local []scanner.Repo) *clonePathView {
	dups := scanner.FindByKey(local, provider.Normalize(repo.CloneURL))

	start := ""
	if len(cfg.ScanRoots) > 0 {
		start = expandHome(cfg.ScanRoots[0])
	}
	if start == "" {
		if home, err := os.UserHomeDir(); err == nil {
			start = home
		} else {
			start = "/"
		}
	}

	fi := textinput.New()
	fi.Prompt = "find: "
	fi.Placeholder = "type to filter folders…"
	fi.Focus()

	nd := textinput.New()
	nd.Prompt = "new folder: "
	nd.Placeholder = "name"

	protocol := "https"
	if repo.CloneURL == "" && repo.SSHURL != "" {
		protocol = "ssh"
	}
	v := &clonePathView{
		cfg: cfg, repo: repo, dupPaths: dups,
		protocol: protocol, hasSSH: repo.SSHURL != "",
		dir: filepath.Clean(start), filter: fi, newDir: nd,
	}
	v.loadDir()
	return v
}

// makeFolder creates name under the current directory and navigates into
// it. name may be a nested path (e.g. "work/clients").
func (v *clonePathView) makeFolder(name string) (view, tea.Cmd) {
	name = strings.TrimSpace(name)
	v.creating = false
	v.newDir.Blur()
	v.newDir.SetValue("")
	if name == "" {
		return v, nil
	}
	target := filepath.Join(v.dir, name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return v, fail(fmt.Errorf("create folder: %v", err))
	}
	v.setDir(target)
	return v, toast("created " + name)
}

func (v *clonePathView) toggleProtocol() {
	if !v.hasSSH || v.repo.CloneURL == "" {
		return
	}
	if v.protocol == "https" {
		v.protocol = "ssh"
	} else {
		v.protocol = "https"
	}
}

// loadDir reads the subdirectories of v.dir.
func (v *clonePathView) loadDir() {
	v.entries = v.entries[:0]
	ents, err := os.ReadDir(v.dir)
	if err == nil {
		for _, e := range ents {
			if e.IsDir() {
				v.entries = append(v.entries, e.Name())
			}
		}
		sort.Slice(v.entries, func(i, j int) bool {
			return strings.ToLower(v.entries[i]) < strings.ToLower(v.entries[j])
		})
	}
}

// setDir navigates to d, clearing the filter and reloading entries.
func (v *clonePathView) setDir(d string) {
	v.dir = filepath.Clean(d)
	v.filter.SetValue("")
	v.loadDir()
	v.rebuild()
	v.tbl.SetCursor(0)
}

// dest is where the repo would be cloned: <current dir>/<repo name>.
func (v *clonePathView) dest() string { return filepath.Join(v.dir, v.repo.Name) }

func (v *clonePathView) Init() tea.Cmd   { return textinput.Blink }
func (v *clonePathView) Title() string   { return "clone · " + v.repo.Name }
func (v *clonePathView) Absorbing() bool { return true }

func (v *clonePathView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil
	case tea.KeyMsg:
		// "new folder" name entry takes over the keyboard while active.
		if v.creating {
			switch msg.String() {
			case "esc":
				v.creating = false
				v.newDir.Blur()
				v.newDir.SetValue("")
				return v, nil
			case "enter":
				return v.makeFolder(v.newDir.Value())
			}
			var cmd tea.Cmd
			v.newDir, cmd = v.newDir.Update(msg)
			return v, cmd
		}
		switch msg.String() {
		case "esc":
			return v, func() tea.Msg { return popViewMsg{} }
		case "ctrl+n":
			v.creating = true
			v.newDir.Focus()
			return v, textinput.Blink
		case "tab", "shift+tab", "ctrl+t":
			v.toggleProtocol()
			return v, nil
		case "enter":
			return v.activate()
		case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d":
			var cmd tea.Cmd
			v.tbl, cmd = v.tbl.Update(msg)
			return v, cmd
		case "left":
			if parent := filepath.Dir(v.dir); parent != v.dir {
				v.setDir(parent)
			}
			return v, nil
		case "right":
			if it := v.currentItem(); it != nil && it.kind == itemDir {
				v.setDir(filepath.Join(v.dir, it.name))
			}
			return v, nil
		}
		// anything else edits the filter
		var cmd tea.Cmd
		v.filter, cmd = v.filter.Update(msg)
		v.rebuild()
		return v, cmd
	}
	return v, nil
}

func (v *clonePathView) currentItem() *browseItem {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.items) {
		return nil
	}
	return &v.items[i]
}

// activate handles Enter on the selected row.
func (v *clonePathView) activate() (view, tea.Cmd) {
	it := v.currentItem()
	if it == nil {
		return v, nil
	}
	switch it.kind {
	case itemNewFolder:
		v.creating = true
		v.newDir.Focus()
		return v, textinput.Blink
	case itemUp:
		if parent := filepath.Dir(v.dir); parent != v.dir {
			v.setDir(parent)
		}
		return v, nil
	case itemDir:
		v.setDir(filepath.Join(v.dir, it.name))
		return v, nil
	default: // itemClone
		dest := v.dest()
		if gitops.DestExists(dest) {
			return v, fail(fmt.Errorf("%s already exists and is not empty", dest))
		}
		url := v.repo.CloneURLFor(v.protocol)
		if url == "" {
			return v, fail(fmt.Errorf("no %s url available for this repo", v.protocol))
		}
		return v, tea.Batch(
			func() tea.Msg { return execCloneMsg{url: url, dest: dest} },
			func() tea.Msg { return switchViewMsg{newRepoListView(v.cfg)} },
		)
	}
}

func (v *clonePathView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	q := strings.ToLower(strings.TrimSpace(v.filter.Value()))

	v.items = v.items[:0]
	v.items = append(v.items, browseItem{kind: itemClone})
	v.items = append(v.items, browseItem{kind: itemNewFolder})
	if parent := filepath.Dir(v.dir); parent != v.dir {
		v.items = append(v.items, browseItem{kind: itemUp})
	}
	for _, name := range v.entries {
		if q == "" || strings.Contains(strings.ToLower(name), q) {
			v.items = append(v.items, browseItem{kind: itemDir, name: name})
		}
	}

	col := []table.Column{{Title: "FOLDER", Width: v.w - cellPadding - 2}}
	if col[0].Width < 20 {
		col[0].Width = 20
	}
	rows := make([]table.Row, 0, len(v.items))
	for _, it := range v.items {
		switch it.kind {
		case itemClone:
			rows = append(rows, table.Row{" clone here → " + filepath.Base(v.dest()) + "/"})
		case itemNewFolder:
			rows = append(rows, table.Row{" ＋ new folder here…"})
		case itemUp:
			rows = append(rows, table.Row{" .."})
		default:
			rows = append(rows, table.Row{"  " + it.name + "/"})
		}
	}

	height := v.h - 12
	if len(v.dupPaths) > 0 {
		height -= len(v.dupPaths) + 2
	}
	if height < 3 {
		height = 3
	}
	cursor := 0
	if v.built {
		cursor = v.tbl.Cursor()
	}
	v.tbl = newStyledTable(col, rows, height)
	v.built = true
	if cursor >= 0 && cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *clonePathView) View(width, height int) string {
	var b strings.Builder
	b.WriteString("\n  cloning " + v.repo.FullName + "\n\n")

	// Protocol chips.
	hint := ""
	if v.hasSSH && v.repo.CloneURL != "" {
		hint = theme.Faint.Render("   tab ⇄ switch")
	}
	b.WriteString("  protocol:" + hint + "\n  ")
	chips := lipgloss.JoinHorizontal(lipgloss.Top,
		protoChip("HTTPS", v.protocol == "https"), "  ", protoChip("SSH", v.protocol == "ssh"))
	b.WriteString(chips + "\n")

	if len(v.dupPaths) > 0 {
		b.WriteString("\n  ⚠ already cloned at:\n")
		for _, p := range v.dupPaths {
			b.WriteString("      " + shortenPath(p) + "\n")
		}
	}

	// Destination preview + folder browser.
	b.WriteString("\n  target: " +
		lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render(shortenPath(v.dest())) + "\n")
	b.WriteString("  " + theme.Faint.Render(shortenPath(v.dir)) + "\n")
	if v.creating {
		b.WriteString("  " + v.newDir.View() + "\n")
		b.WriteString(v.tbl.View())
		b.WriteString("\n  " + theme.Faint.Render("type a name · enter = create & open · esc = cancel"))
		return padView(b.String(), width)
	}
	b.WriteString("  " + v.filter.View() + "\n")
	b.WriteString(v.tbl.View())
	b.WriteString("\n  " + theme.Faint.Render("↑↓ move · → enter folder · ← up · ctrl+n new folder · enter = act · tab = protocol · esc = cancel"))
	return padView(b.String(), width)
}

func (v *clonePathView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open/clone")),
		key.NewBinding(key.WithKeys("→"), key.WithHelp("→", "enter folder")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new folder")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "https/ssh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}
func (v *clonePathView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }

// ── helpers ──

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
