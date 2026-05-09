package tui

import "github.com/charmbracelet/lipgloss"

var (
	// ── Palette ──────────────────────────────────────────────────
	bg        = lipgloss.Color("#0D0D0D")
	fg        = lipgloss.Color("#D4D4D4")
	subtle    = lipgloss.Color("#555555")
	accent    = lipgloss.Color("#7C6AF7") // violet
	accentDim = lipgloss.Color("#3D3562")
	green     = lipgloss.Color("#4EC994")
	red       = lipgloss.Color("#E06C75")
	orange    = lipgloss.Color("#E5A24A")
	blue      = lipgloss.Color("#61AFEF")
	white     = lipgloss.Color("#EEEEEE")

	// ── Base ─────────────────────────────────────────────────────
	Base = lipgloss.NewStyle().
		Background(bg).
		Foreground(fg)

	// ── Header bar ───────────────────────────────────────────────
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Padding(0, 1)

	VersionStyle = lipgloss.NewStyle().
			Foreground(subtle)

	// ── Tabs ─────────────────────────────────────────────────────
	TabBar = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle).
		BorderBottom(true)

	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 2)

	InactiveTab = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 2)

	// ── List panel ───────────────────────────────────────────────
	ListPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1)

	ListPanelActive = ListPanel.
		BorderForeground(accent)


	// ── List items ───────────────────────────────────────────────
	ItemName = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	ItemMeta = lipgloss.NewStyle().
			Foreground(subtle)

	SelectedItemName = lipgloss.NewStyle().
				Foreground(accent).
				Bold(true)

	SelectedItemMeta = lipgloss.NewStyle().
				Foreground(accentDim)

	// ── Detail panel ─────────────────────────────────────────────
	DetailPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)

	DetailTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			MarginBottom(1)

	DetailLabel = lipgloss.NewStyle().
			Foreground(subtle).
			Width(18)

	DetailValue = lipgloss.NewStyle().
			Foreground(fg)

	// ── Status chips ─────────────────────────────────────────────
	ChipSynced  = lipgloss.NewStyle().Bold(true).Foreground(green)
	ChipError   = lipgloss.NewStyle().Bold(true).Foreground(red)
	ChipPending = lipgloss.NewStyle().Bold(true).Foreground(orange)
	ChipDefault = lipgloss.NewStyle().Bold(true).Foreground(blue)

	// ── Help bar ─────────────────────────────────────────────────
	HelpKey  = lipgloss.NewStyle().Foreground(accent)
	HelpSep  = lipgloss.NewStyle().Foreground(subtle)
	HelpDesc = lipgloss.NewStyle().Foreground(subtle)

	// ── Confirm prompt ───────────────────────────────────────────
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(orange).
			Bold(true)

	// ── Error ─────────────────────────────────────────────────────
	ErrStyle = lipgloss.NewStyle().Foreground(red)
)

// StatusChip returns a styled status indicator.
func StatusChip(status string) string {
	switch status {
	case "Synced", "Active":
		return ChipSynced.Render("● " + status)
	case "Error", "Unreachable":
		return ChipError.Render("● " + status)
	case "Pending", "Syncing", "OutOfSync":
		return ChipPending.Render("● " + status)
	default:
		if status == "" {
			return ChipDefault.Render("● —")
		}
		return ChipDefault.Render("● " + status)
	}
}
