package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/tristanj/dusty/internal/agent"
	"github.com/tristanj/dusty/internal/config"
)

// settingsSection represents a named group of selectable options.
type settingsSection struct {
	label   string
	options []string
	current int
}

// SettingsOverlay is an F2-triggered overlay for changing persona, model, and
// routing mode at runtime.
type SettingsOverlay struct {
	visible  bool
	sections []settingsSection
	secIdx   int // active section index
	theme    Theme
	width    int
	height   int
	agent    *agent.Agent
	cfg      *config.Config
}

// NewSettingsOverlay creates the settings overlay (hidden by default).
func NewSettingsOverlay(a *agent.Agent, cfg *config.Config, theme Theme) SettingsOverlay {
	personas := agent.ListPersonas()
	personaIdx := 0
	for i, p := range personas {
		if p == cfg.Agent.Personality {
			personaIdx = i
			break
		}
	}

	modes := []string{"local", "cloud", "auto"}
	modeIdx := 0
	for i, m := range modes {
		if m == cfg.Inference.Mode {
			modeIdx = i
			break
		}
	}

	return SettingsOverlay{
		visible: false,
		sections: []settingsSection{
			{label: "Persona", options: personas, current: personaIdx},
			{label: "Mode", options: modes, current: modeIdx},
		},
		secIdx: 0,
		theme:  theme,
		agent:  a,
		cfg:    cfg,
	}
}

// SetSize updates dimensions used for centering.
func (s *SettingsOverlay) SetSize(w, h int) {
	s.width = w
	s.height = h
}

// SetTheme updates the theme.
func (s *SettingsOverlay) SetTheme(t Theme) {
	s.theme = t
}

// Toggle shows/hides the overlay.
func (s *SettingsOverlay) Toggle() {
	s.visible = !s.visible
}

// Visible reports whether the overlay is shown.
func (s SettingsOverlay) Visible() bool {
	return s.visible
}

// Update handles key input when the overlay is visible.
// Returns a CommandResult describing any changes made.
func (s *SettingsOverlay) Update(msg tea.Msg) (changed CommandResult) {
	if !s.visible {
		return
	}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return
	}

	sec := &s.sections[s.secIdx]

	switch kp.String() {
	case "up", "k":
		if sec.current > 0 {
			sec.current--
		}
	case "down", "j":
		if sec.current < len(sec.options)-1 {
			sec.current++
		}
	case "left", "h":
		if s.secIdx > 0 {
			s.secIdx--
		}
	case "right", "l":
		if s.secIdx < len(s.sections)-1 {
			s.secIdx++
		}
	case "tab":
		s.secIdx = (s.secIdx + 1) % len(s.sections)
	case "enter":
		changed = s.applySelection()
	case "esc", "f2":
		s.visible = false
	}
	return
}

// applySelection applies the currently highlighted option.
func (s *SettingsOverlay) applySelection() CommandResult {
	sec := s.sections[s.secIdx]
	selected := sec.options[sec.current]

	switch sec.label {
	case "Persona":
		s.agent.SetPersona(selected)
		s.cfg.Agent.Personality = selected
		return CommandResult{
			Output:          fmt.Sprintf("[persona set to: %s]", selected),
			SettingsChanged: true,
		}
	case "Mode":
		switch selected {
		case "local":
			s.agent.SetRoutingPreference(agent.PreferLocal)
		case "cloud":
			s.agent.SetRoutingPreference(agent.PreferCloud)
		case "auto":
			s.agent.SetRoutingPreference(agent.PreferAuto)
		}
		s.cfg.Inference.Mode = selected
		return CommandResult{
			Output:          fmt.Sprintf("[routing mode set to: %s]", selected),
			SettingsChanged: true,
		}
	}
	return CommandResult{}
}

// View renders the settings overlay as a string to be overlaid on the main UI.
func (s SettingsOverlay) View() string {
	if !s.visible {
		return ""
	}

	panelWidth := 36
	var sb strings.Builder

	// Title.
	title := s.theme.Accent.Render(" Settings ")
	titleLine := "┌─" + title + strings.Repeat("─", panelWidth-4-len(" Settings ")+2) + "─┐"
	sb.WriteString(s.theme.Border.Render(titleLine) + "\n")

	for si, sec := range s.sections {
		active := si == s.secIdx
		sectionLabel := sec.label
		if active {
			sectionLabel = s.theme.Accent.Render("▶ " + sec.label)
		} else {
			sectionLabel = s.theme.Dimmed.Render("  " + sec.label)
		}
		sb.WriteString(s.theme.Border.Render("│ "))
		sb.WriteString(sectionLabel)
		sb.WriteString("\n")

		for oi, opt := range sec.options {
			selected := oi == sec.current
			cursor := "  "
			var line string
			if selected && active {
				cursor = "● "
				line = s.theme.Accent.Render(cursor + opt)
			} else if selected {
				cursor = "○ "
				line = s.theme.Primary.Render(cursor + opt)
			} else {
				line = s.theme.Dimmed.Render(cursor + opt)
			}
			sb.WriteString(s.theme.Border.Render("│    "))
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(s.theme.Border.Render("│") + "\n")
	}

	footerLine := "└" + strings.Repeat("─", panelWidth-2) + "┘"
	sb.WriteString(s.theme.Border.Render(footerLine) + "\n")
	sb.WriteString(s.theme.Dimmed.Render("  ↑↓ select  Tab next section  Enter apply  Esc close"))

	return sb.String()
}
