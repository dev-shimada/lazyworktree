package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#5F5FD7", Dark: "#8787FF"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#767676", Dark: "#9E9E9E"}
	colorError  = lipgloss.AdaptiveColor{Light: "#D70000", Dark: "#FF5F5F"}
	colorOk     = lipgloss.AdaptiveColor{Light: "#008700", Dark: "#5FD75F"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#AF8700", Dark: "#FFD75F"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	statusErrStyle = lipgloss.NewStyle().Foreground(colorError).Padding(0, 1)
	statusOkStyle  = lipgloss.NewStyle().Foreground(colorOk).Padding(0, 1)

	lockedBadgeStyle   = lipgloss.NewStyle().Foreground(colorWarn)
	prunableBadgeStyle = lipgloss.NewStyle().Foreground(colorError)
	branchStyle        = lipgloss.NewStyle().Foreground(colorAccent)
	prBadgeStyle       = lipgloss.NewStyle().Foreground(colorOk)

	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Padding(0, 1).Underline(true)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)

	formLabelStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1, 0, 0)
	formBoxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(1, 2)
)
