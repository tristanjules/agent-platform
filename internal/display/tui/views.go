package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/tristanj/dusty/internal/state"
)

const dustyVersion = "v0.1"

// RenderStatusBar renders the top status line showing agent name, version,
// inference mode, and current model.
func RenderStatusBar(width int, agentName, model, mode string, agentState state.AgentState, theme Theme) string {
	modeTag := "[" + strings.ToUpper(mode) + "]"
	stateTag := "(" + agentState.String() + ")"

	left := fmt.Sprintf("  %s %s  %s %s", agentName, dustyVersion, modeTag, model)
	right := "  " + stateTag + "  "

	// Pad between left and right.
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + strings.Repeat(" ", gap) + right
	return theme.StatusBar.Width(width).Render(bar)
}

// RenderHotkeyBar renders the bottom row of keyboard shortcut hints.
func RenderHotkeyBar(width int, voiceActive bool, theme Theme) string {
	voice := "[F1] Voice On"
	if voiceActive {
		voice = "[F1] Voice Off"
	}

	hints := []string{
		"[Tab] Visual",
		voice,
		"[F2] Settings",
		"[Ctrl+L] Clear",
		"[Ctrl+C] Quit",
	}

	bar := "  " + strings.Join(hints, "  ·  ") + "  "
	// Trim if too wide.
	if lipgloss.Width(bar) > width {
		bar = "  " + strings.Join(hints[:3], "  ·  ") + "  "
	}

	return theme.HotkeyBar.Width(width).Render(bar)
}

// RenderBorderedPane wraps content in a titled box-drawing border.
func RenderBorderedPane(title, content string, width, height int, theme Theme) string {
	inner := lipgloss.NewStyle().
		Width(width - 2).
		Height(height - 2).
		Render(content)

	return theme.Border.
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Border.GetForeground()).
		Width(width - 2).
		Height(height - 2).
		Render(inner)
}

// PaneTitle renders a pane title suitable for embedding in the border.
func PaneTitle(title string, theme Theme) string {
	return theme.Border.Render("─ " + title + " ")
}

// RenderPane wraps content with a simple titled border using box-drawing chars.
func RenderPane(title, content string, width, height int, theme Theme) string {
	// Build the top border with title.
	borderColor := theme.Border.GetForeground()
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	titleStyled := borderStyle.Render("─ ") + theme.Accent.Render(title) + borderStyle.Render(" ")
	titleWidth := lipgloss.Width(titleStyled)
	remaining := width - titleWidth - 2 // 2 for the corner chars
	if remaining < 0 {
		remaining = 0
	}

	topBorder := borderStyle.Render("┌") + titleStyled + borderStyle.Render(strings.Repeat("─", remaining)) + borderStyle.Render("┐")

	// Content area: each line padded to width-2, then bordered.
	contentLines := strings.Split(content, "\n")
	// Ensure we fill height-2 lines (subtract top/bottom borders).
	contentHeight := height - 2
	var body strings.Builder
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Truncate line if too wide.
		visibleWidth := lipgloss.Width(line)
		innerWidth := width - 2
		if visibleWidth > innerWidth {
			// Naive truncation — keep the string as-is; viewport handles wrapping.
			line = line
		} else {
			// Pad to fill.
			line = line + strings.Repeat(" ", innerWidth-visibleWidth)
		}
		body.WriteString(borderStyle.Render("│"))
		body.WriteString(theme.Primary.Render(line))
		body.WriteString(borderStyle.Render("│"))
		body.WriteString("\n")
	}

	bottomBorder := borderStyle.Render("└") + borderStyle.Render(strings.Repeat("─", width-2)) + borderStyle.Render("┘")

	return topBorder + "\n" + body.String() + bottomBorder
}
