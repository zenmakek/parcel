package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("#00D4FF")
	colorSuccess = lipgloss.Color("#00FF88")
	colorError   = lipgloss.Color("#FF4444")
	colorMuted   = lipgloss.Color("#555555")
	colorText    = lipgloss.Color("#EEEEEE")
	colorBorder  = lipgloss.Color("#333333")
	colorOTP     = lipgloss.Color("#FFD700")

	styleTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleNormal = lipgloss.NewStyle().
			Foreground(colorText)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	styleOTP = lipgloss.NewStyle().
			Foreground(colorOTP).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(2)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 3).
			MarginTop(1).
			MarginBottom(1)

	styleInput = lipgloss.NewStyle().
			Foreground(colorText).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorPrimary).
			Width(40)

	stylePrompt = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleProgressBar = lipgloss.NewStyle().
				Foreground(colorPrimary)

	styleProgressDone = lipgloss.NewStyle().
				Foreground(colorSuccess)
)
