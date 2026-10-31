package ui

import "charm.land/lipgloss/v2"

var (
	accent       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9D2E"))
	muted        = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C8799"))
	selected     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#243B55")).Bold(true)
	panel        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3D4D63")).Padding(1, 2)
	focusedPanel = panel.Copy().BorderForeground(lipgloss.Color("#FF9D2E"))
	header       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	good         = lipgloss.NewStyle().Foreground(lipgloss.Color("#78D64B"))
)
