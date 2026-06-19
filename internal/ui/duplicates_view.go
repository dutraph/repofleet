package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
)

// dupGroup is a set of working copies that share the same remote — i.e.
// the same repository cloned into more than one path.
type dupGroup struct {
	name     string
	provider provider.Kind
	remote   string
	repos    []scanner.Repo
}

// ───────────────────────── groups list ─────────────────────────

type duplicatesView struct {
	groups []dupGroup

	// provider filter: provSel 0 = all; otherwise provCycle[provSel-1].
	provCycle []provider.Kind
	provSel   int
	filtered  []int // indices into groups currently shown

	tbl   table.Model
	w, h  int
	built bool
}

func newDuplicatesView(repos []scanner.Repo) *duplicatesView {
	byKey := map[string]*dupGroup{}
	var order []string
	for _, r := range repos {
		if r.DupCount <= 1 || r.Key == "" {
			continue
		}
		g, ok := byKey[r.Key]
		if !ok {
			g = &dupGroup{name: r.Name, provider: r.Provider, remote: r.RemoteURL}
			byKey[r.Key] = g
			order = append(order, r.Key)
		}
		g.repos = append(g.repos, r)
	}
	groups := make([]dupGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, *byKey[k])
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })

	// providers present among the duplicate groups (for the `t` filter)
	present := map[provider.Kind]bool{}
	for _, g := range groups {
		present[g.provider] = true
	}
	var cycle []provider.Kind
	for _, k := range groupOrder {
		if present[k] {
			cycle = append(cycle, k)
		}
	}
	return &duplicatesView{groups: groups, provCycle: cycle}
}

func (v *duplicatesView) Init() tea.Cmd { return nil }

func (v *duplicatesView) Title() string {
	// Count reflects the active type filter (v.filtered is the shown subset).
	n := len(v.groups)
	if v.built {
		n = len(v.filtered)
	}
	base := fmt.Sprintf("duplicates · %d groups", n)
	if v.provSel > 0 && v.provSel-1 < len(v.provCycle) {
		base += " · " + provider.Meta(v.provCycle[v.provSel-1]).Name
	}
	return base
}

func (v *duplicatesView) Absorbing() bool { return false }

func (v *duplicatesView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "t":
			v.provSel = (v.provSel + 1) % (len(v.provCycle) + 1)
			v.rebuild()
			return v, nil
		case "enter":
			if g := v.currentGroup(); g != nil {
				gg := *g
				return v, func() tea.Msg { return pushViewMsg{newDupGroupView(gg)} }
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

func (v *duplicatesView) currentGroup() *dupGroup {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.filtered) {
		return nil
	}
	return &v.groups[v.filtered[i]]
}

func (v *duplicatesView) View(width, height int) string {
	if len(v.groups) == 0 {
		return padView("\n  no duplicate repositories — every remote is cloned once. 🎉", width)
	}
	return padView(v.tbl.View(), width)
}

func (v *duplicatesView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	const wCount, wType = 7, 12
	wRemote := v.w/2 - cellPadding
	wName := v.w - wCount - wType - wRemote - cellPadding*4 - 2
	if wName < 16 {
		wName = 16
	}
	cols := []table.Column{
		{Title: "NAME", Width: wName},
		{Title: "COPIES", Width: wCount},
		{Title: "TYPE", Width: wType},
		{Title: "REMOTE", Width: wRemote},
	}
	v.filtered = v.filtered[:0]
	for i, g := range v.groups {
		if v.provSel > 0 && v.provSel-1 < len(v.provCycle) && g.provider != v.provCycle[v.provSel-1] {
			continue
		}
		v.filtered = append(v.filtered, i)
	}

	rows := make([]table.Row, 0, len(v.filtered))
	for _, i := range v.filtered {
		g := v.groups[i]
		rows = append(rows, table.Row{
			g.name, fmt.Sprintf("%d", len(g.repos)),
			provider.Meta(g.provider).Name, g.remote,
		})
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
	if cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *duplicatesView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "see paths")),
		keyTypeFilter,
		keys.Back,
	}
}
func (v *duplicatesView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }

// ───────────────────────── one group's paths ─────────────────────────

type dupGroupView struct {
	group dupGroup
	tbl   table.Model
	w, h  int
	built bool
}

func newDupGroupView(g dupGroup) *dupGroupView { return &dupGroupView{group: g} }

func (v *dupGroupView) Init() tea.Cmd   { return nil }
func (v *dupGroupView) Title() string   { return "duplicate · " + v.group.name }
func (v *dupGroupView) Absorbing() bool { return false }

// SelectedRepo lets the root `:` command bar run git on the highlighted
// duplicate copy.
func (v *dupGroupView) SelectedRepo() (string, string, bool) {
	i := v.tbl.Cursor()
	if i >= 0 && i < len(v.group.repos) {
		r := v.group.repos[i]
		return r.Path, r.Name, true
	}
	return "", "", false
}

func (v *dupGroupView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w, v.h = msg.Width, msg.Height
		v.rebuild()
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "enter" && len(v.group.repos) > 0 {
			i := v.tbl.Cursor()
			if i >= 0 && i < len(v.group.repos) {
				r := v.group.repos[i]
				return v, func() tea.Msg { return pushViewMsg{newDetailView(r)} }
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

func (v *dupGroupView) View(width, height int) string {
	head := "  remote: " + v.group.remote + "\n\n"
	return padView("\n"+head+v.tbl.View(), width)
}

func (v *dupGroupView) rebuild() {
	if v.w == 0 {
		v.w = 100
	}
	cols := []table.Column{
		{Title: "#", Width: 4},
		{Title: "PATH", Width: v.w - 4 - cellPadding*2 - 2},
	}
	rows := make([]table.Row, 0, len(v.group.repos))
	for i, r := range v.group.repos {
		rows = append(rows, table.Row{fmt.Sprintf("%d", i+1), shortenPath(r.Path)})
	}
	h := v.h - 6
	if h < 3 {
		h = 3
	}
	cursor := 0
	if v.built {
		cursor = v.tbl.Cursor()
	}
	v.tbl = newStyledTable(cols, rows, h)
	v.built = true
	if cursor < len(rows) {
		v.tbl.SetCursor(cursor)
	}
}

func (v *dupGroupView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		keys.Back,
	}
}
func (v *dupGroupView) FullHelp() [][]key.Binding { return [][]key.Binding{v.ShortHelp()} }
