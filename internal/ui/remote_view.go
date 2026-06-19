package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

type clonePathView struct {
	cfg      *config.Config
	repo     api.RemoteRepo
	dupPaths []string
	input    textinput.Model
	protocol string // "https" or "ssh"
	hasSSH   bool
	w, h     int
}

func newClonePathView(cfg *config.Config, repo api.RemoteRepo, local []scanner.Repo) *clonePathView {
	dups := scanner.FindByKey(local, provider.Normalize(repo.CloneURL))

	root := ""
	if len(cfg.ScanRoots) > 0 {
		root = cfg.ScanRoots[0]
	} else if home, err := os.UserHomeDir(); err == nil {
		root = home
	}
	def := filepath.Join(root, repo.Name)

	in := textinput.New()
	in.Prompt = "› "
	in.SetValue(def)
	in.CursorEnd()
	in.Focus()
	in.Width = 60

	// Default to HTTPS, but fall back to whichever protocol the API
	// actually returned a URL for.
	protocol := "https"
	if repo.CloneURL == "" && repo.SSHURL != "" {
		protocol = "ssh"
	}
	return &clonePathView{
		cfg: cfg, repo: repo, dupPaths: dups, input: in,
		protocol: protocol, hasSSH: repo.SSHURL != "",
	}
}

// toggleProtocol flips between https and ssh when both are available.
func (v *clonePathView) toggleProtocol() {
	if !v.hasSSH || v.repo.CloneURL == "" {
		return // only one protocol available — nothing to toggle
	}
	if v.protocol == "https" {
		v.protocol = "ssh"
	} else {
		v.protocol = "https"
	}
}

func (v *clonePathView) Init() tea.Cmd   { return textinput.Blink }
func (v *clonePathView) Title() string   { return "clone · " + v.repo.Name }
func (v *clonePathView) Absorbing() bool { return true }

func (v *clonePathView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return v, func() tea.Msg { return popViewMsg{} }
		case "tab", "shift+tab", "ctrl+t":
			v.toggleProtocol()
			return v, nil
		case "enter":
			dest := expandHome(strings.TrimSpace(v.input.Value()))
			if dest == "" {
				return v, fail(fmt.Errorf("destination path is empty"))
			}
			if gitops.DestExists(dest) {
				return v, fail(fmt.Errorf("%s already exists and is not empty", dest))
			}
			url := v.repo.CloneURLFor(v.protocol)
			if url == "" {
				return v, fail(fmt.Errorf("no %s url available for this repo", v.protocol))
			}
			// Switch back to a fresh home view so the post-clone
			// cloneDoneMsg lands on the repo list (which rescans).
			return v, tea.Batch(
				func() tea.Msg { return execCloneMsg{url: url, dest: dest} },
				func() tea.Msg { return switchViewMsg{newRepoListView(v.cfg)} },
			)
		}
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *clonePathView) View(width, height int) string {
	var b strings.Builder
	b.WriteString("\n  cloning ")
	b.WriteString(v.repo.FullName)
	b.WriteString("\n\n")

	// Protocol selector — highlighted chips, active one filled.
	hint := ""
	if v.hasSSH && v.repo.CloneURL != "" {
		hint = theme.Faint.Render("   tab ⇄ switch")
	}
	b.WriteString("  choose clone protocol:" + hint + "\n\n")
	chips := lipgloss.JoinHorizontal(
		lipgloss.Top,
		protoChip("HTTPS", v.protocol == "https"),
		"   ",
		protoChip("SSH", v.protocol == "ssh"),
	)
	b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(chips))
	url := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render(v.repo.CloneURLFor(v.protocol))
	b.WriteString("\n\n  " + theme.Faint.Render("url:") + " " + url + "\n\n")

	if len(v.dupPaths) > 0 {
		b.WriteString("  ⚠ this repository is already cloned locally at:\n")
		for _, p := range v.dupPaths {
			b.WriteString("      " + shortenPath(p) + "\n")
		}
		b.WriteString("  cloning again will create a duplicate.\n\n")
	}

	b.WriteString("  clone into:\n  ")
	b.WriteString(v.input.View())
	b.WriteString("\n\n  ")
	b.WriteString("enter = clone   tab = protocol   esc = cancel")
	return padView(b.String(), width)
}

func (v *clonePathView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "clone")),
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
