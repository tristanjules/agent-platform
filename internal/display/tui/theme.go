// Package tui implements the Bubble Tea v2 terminal UI for DUSTY.
package tui

import (
	"charm.land/lipgloss/v2"
)

// Theme holds all Lip Gloss styles for a display theme.
type Theme struct {
	Name string

	// Primary is the main text style.
	Primary lipgloss.Style
	// Dimmed is for secondary content (reasoning pane, timestamps).
	Dimmed lipgloss.Style
	// Accent is for the agent name and highlighted responses.
	Accent lipgloss.Style
	// UserLabel is for the "You:" label in conversation.
	UserLabel lipgloss.Style
	// AgentLabel is for the "DUSTY:" label in conversation.
	AgentLabel lipgloss.Style
	// Border is the style for pane borders and titles.
	Border lipgloss.Style
	// StatusBar is for the top status bar.
	StatusBar lipgloss.Style
	// HotkeyBar is for the bottom hotkey hint bar.
	HotkeyBar lipgloss.Style
	// HotkeyKey is for the key part of hotkey hints (e.g. "[F1]").
	HotkeyKey lipgloss.Style
	// Input is for the text input field.
	Input lipgloss.Style
	// Error is for error messages.
	Error lipgloss.Style
	// Cursor is for the streaming cursor character.
	Cursor lipgloss.Style
}

// ThemeByName returns the named theme, defaulting to phosphor-green.
func ThemeByName(name string) Theme {
	switch name {
	case "amber":
		return amberTheme()
	case "blue":
		return blueTheme()
	default:
		return phosphorGreenTheme()
	}
}

func phosphorGreenTheme() Theme {
	fg := lipgloss.Color("#33ff33")
	fgDim := lipgloss.Color("#1a8c1a")
	fgAccent := lipgloss.Color("#88ff88")
	bg := lipgloss.Color("#0a0a0a")
	statusBg := lipgloss.Color("#1a3a1a")

	return Theme{
		Name:    "phosphor-green",
		Primary: lipgloss.NewStyle().Foreground(fg).Background(bg),
		Dimmed:  lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		Accent:  lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		UserLabel: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		AgentLabel: lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		StatusBar: lipgloss.NewStyle().Foreground(fg).Background(statusBg).Bold(true),
		HotkeyBar: lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		HotkeyKey: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		Input:     lipgloss.NewStyle().Foreground(fg).Background(bg),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Background(bg),
		Cursor:    lipgloss.NewStyle().Foreground(fgAccent).Background(bg),
	}
}

func amberTheme() Theme {
	fg := lipgloss.Color("#ffb000")
	fgDim := lipgloss.Color("#8c6000")
	fgAccent := lipgloss.Color("#ffd080")
	bg := lipgloss.Color("#0a0800")
	statusBg := lipgloss.Color("#3a2800")

	return Theme{
		Name:    "amber",
		Primary: lipgloss.NewStyle().Foreground(fg).Background(bg),
		Dimmed:  lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		Accent:  lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		UserLabel: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		AgentLabel: lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		StatusBar: lipgloss.NewStyle().Foreground(fg).Background(statusBg).Bold(true),
		HotkeyBar: lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		HotkeyKey: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		Input:     lipgloss.NewStyle().Foreground(fg).Background(bg),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Background(bg),
		Cursor:    lipgloss.NewStyle().Foreground(fgAccent).Background(bg),
	}
}

func blueTheme() Theme {
	fg := lipgloss.Color("#00aaff")
	fgDim := lipgloss.Color("#005580")
	fgAccent := lipgloss.Color("#66ccff")
	bg := lipgloss.Color("#00080a")
	statusBg := lipgloss.Color("#00203a")

	return Theme{
		Name:    "blue",
		Primary: lipgloss.NewStyle().Foreground(fg).Background(bg),
		Dimmed:  lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		Accent:  lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		UserLabel: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		AgentLabel: lipgloss.NewStyle().Foreground(fgAccent).Background(bg).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		StatusBar: lipgloss.NewStyle().Foreground(fg).Background(statusBg).Bold(true),
		HotkeyBar: lipgloss.NewStyle().Foreground(fgDim).Background(bg),
		HotkeyKey: lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true),
		Input:     lipgloss.NewStyle().Foreground(fg).Background(bg),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Background(bg),
		Cursor:    lipgloss.NewStyle().Foreground(fgAccent).Background(bg),
	}
}
