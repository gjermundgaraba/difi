package ui

import "github.com/charmbracelet/lipgloss"

var (
	// -- NORD PALETTE --
	nord0  = lipgloss.Color("#2E3440") // Dark background
	nord3  = lipgloss.Color("#4C566A") // Separators / Dimmed
	nord4  = lipgloss.Color("#D8DEE9") // Main Text
	nord11 = lipgloss.Color("#BF616A") // Red (Deleted)
	nord14 = lipgloss.Color("#A3BE8C") // Green (Added)
	nord9  = lipgloss.Color("#81A1C1") // Blue (Focus)

	// -- PANE STYLES --
	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(nord3). // Goal 1: Nord3 Separator
			Padding(0, 1)

	FocusedPaneStyle = PaneStyle.
				BorderForeground(nord9)

	// -- TOP BAR STYLES (Goal 2) --
	TopBarStyle = lipgloss.NewStyle().
			Background(nord0).
			Foreground(nord4).
			Height(1)

	TopInfoStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	TopStatsAddedStyle = lipgloss.NewStyle().
				Foreground(nord14).
				PaddingLeft(1)

	TopStatsDeletedStyle = lipgloss.NewStyle().
				Foreground(nord11).
				PaddingLeft(1).
				PaddingRight(1)

	// -- TREE STYLES --
	DirectoryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	FileStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// -- DIFF VIEW STYLES --
	DiffStyle          = lipgloss.NewStyle().Padding(0, 0)
	DiffSelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("255"))
	LineNumberStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(4).Align(lipgloss.Right).MarginRight(1)

	// -- EMPTY STATE STYLES --
	EmptyLogoStyle   = lipgloss.NewStyle().Foreground(nord9).Bold(true).MarginBottom(1)
	EmptyDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginBottom(1)
	EmptyStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).MarginBottom(2)
	EmptyHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true).MarginBottom(1)
	EmptyCodeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// -- HELPER STYLES --
	HelpDrawerStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(nord3).Padding(1, 2)
	HelpTextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginRight(2)

	// -- BOTTOM STATUS BAR STYLES --
	StatusBarStyle     = lipgloss.NewStyle().Background(nord0).Foreground(nord4).Height(1)
	StatusKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	StatusRepoStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")).Padding(0, 1)
	StatusBranchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Padding(0, 1)
	StatusAddedStyle   = lipgloss.NewStyle().Foreground(nord14).Padding(0, 1)
	StatusDeletedStyle = lipgloss.NewStyle().Foreground(nord11).Padding(0, 1)
	StatusDividerStyle = lipgloss.NewStyle().Foreground(nord3).Padding(0, 1)

	ColorText = lipgloss.Color("252")
)

func InitStyles(cfg interface{}) {}
