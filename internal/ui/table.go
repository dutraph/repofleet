package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dutraph/repofleet/internal/theme"
)

// cellPadding is the horizontal padding bubbles/table applies to every
// cell (Padding(0, 1) on Cell). We subtract 2*N from the available
// width so the table doesn't overflow.
const cellPadding = 2

// fastScrollStep is how many rows J/K jump per press. Five rows is
// noticeably faster than holding j/k while still small enough that
// the cursor doesn't fly past where the eye expects it.
const fastScrollStep = 5

// newStyledTable returns a bubbles/table configured with our palette.
//
// Selection rendering — this is the most important detail to keep:
// bubbles/table v0.18 renders each cell with Cell, joins them, and
// then wraps the whole joined row with Selected. lipgloss has to
// "compose" Selected (Background+Foreground) on top of a string that
// already contains ANSI codes from Cell (Foreground + reset at the
// end of each cell). On some terminals the compose loses the outer
// Background after every inner reset — selection shows only on the
// first cell, or nowhere at all.
//
// Workaround: make Cell ANSI-free — keep only Padding(0,1) and let
// the terminal default handle the foreground. Selected then wraps a
// plain-text row with no inner escapes, so its Background actually
// paints the whole row.
func newStyledTable(cols []table.Column, rows []table.Row, height int) table.Model {
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Foreground(theme.ColorPrimary).
		Bold(true)
	s.Cell = lipgloss.NewStyle().Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Foreground(theme.ColorBg).
		Background(theme.ColorPrimary).
		Bold(true)
	t.SetStyles(s)
	return t
}

// tableLayout returns the table width + per-column-budget for a
// list-style view that occupies the full content area minus a
// 1-char left/right margin. colSum already subtracts the per-cell
// padding so columns can be sized without overflow.
func tableLayout(totalW, nCols int) (tableW, colSum int) {
	tableW = totalW - 2
	if tableW < 20 {
		tableW = 20
	}
	colSum = tableW - cellPadding*nCols
	if colSum < 10 {
		colSum = 10
	}
	return
}

// padView wraps body with 1-char horizontal padding sized to w. This
// is what list-style views return from View() — no box border, just a
// little breathing room.
func padView(body string, w int) string {
	return lipgloss.NewStyle().Padding(0, 1).Width(w).Render(body)
}

// applyFastTableScroll intercepts J/K (capital) and shift+up/down to
// jump multiple rows at a time. Returns the (possibly-mutated) table
// model and true when the message was consumed; callers should skip
// passing the same msg to tbl.Update when handled is true.
//
// Implementation: feed the table N copies of lower-case j/k (the
// default LineUp/LineDown binding) so all of bubbles/table's internal
// scrolling logic — clamping, viewport offset, etc. — still runs
// correctly. No need to touch unexported internals.
func applyFastTableScroll(tbl table.Model, msg tea.KeyMsg) (table.Model, bool) {
	var direction string
	switch msg.String() {
	case "J", "shift+down":
		direction = "j"
	case "K", "shift+up":
		direction = "k"
	default:
		return tbl, false
	}
	step := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(direction)}
	for i := 0; i < fastScrollStep; i++ {
		tbl, _ = tbl.Update(step)
	}
	return tbl, true
}
