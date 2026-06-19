// Package theme centralizes the lipgloss palette + reusable styles.
// Keep this self-contained — the rest of the codebase reaches for
// theme.ColorPrimary / theme.Faint / theme.StatusIcon / etc., never
// inlines hex codes.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette (256-color safe, adapts on truecolor terminals).
var (
	ColorBg     = lipgloss.Color("#0e1116")
	ColorBgAlt  = lipgloss.Color("#161b22")
	ColorBgSel  = lipgloss.Color("#1f2937")
	ColorBorder = lipgloss.Color("#30363d")
	ColorMuted  = lipgloss.Color("#7d8590")
	ColorText   = lipgloss.Color("#e6edf3")
	ColorBright = lipgloss.Color("#ffffff")

	ColorPrimary   = lipgloss.Color("#22d3ee") // cyan
	ColorSecondary = lipgloss.Color("#c084fc") // purple
	ColorAccent    = lipgloss.Color("#f472b6") // pink

	ColorSuccess  = lipgloss.Color("#3fb950")
	ColorWarning  = lipgloss.Color("#d29922")
	ColorError    = lipgloss.Color("#f85149")
	ColorRunning  = lipgloss.Color("#58a6ff")
	ColorPending  = lipgloss.Color("#bb8009")
	ColorCanceled = lipgloss.Color("#8b949e")
)

// Reusable styles.
var (
	Base = lipgloss.NewStyle().Foreground(ColorText)
	Faint = lipgloss.NewStyle().Foreground(ColorMuted)

	Title = lipgloss.NewStyle().
		Foreground(ColorBright).
		Background(ColorPrimary).
		Bold(true).
		Padding(0, 1)

	Logo = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	HeaderBar = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBgAlt).
		Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBgAlt).
		Padding(0, 1)

	CommandBar = lipgloss.NewStyle().
		Foreground(ColorBright).
		Background(ColorBgSel).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorPrimary)

	ViewBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	ViewBoxFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	ErrorBox = lipgloss.NewStyle().
		Foreground(ColorError).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorError).
		Padding(0, 1)

	Toast = lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorPrimary).
		Padding(0, 2).
		Bold(true)

	HelpKey  = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Banner is the small left-of-the-header brand mark.
func Banner() string {
	return Logo.Render(" ▲ fleet ")
}
