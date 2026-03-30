package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// SubmitMsg is sent when the user presses Enter in the input field.
type SubmitMsg struct {
	Text string
}

// InputComponent wraps a bubbles textinput.
type InputComponent struct {
	ti    textinput.Model
	theme Theme
}

// NewInputComponent creates a focused text input component.
func NewInputComponent(theme Theme) InputComponent {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Prompt = "> "
	ti.Focus()

	return InputComponent{ti: ti, theme: theme}
}

// SetTheme updates the theme.
func (c *InputComponent) SetTheme(t Theme) {
	c.theme = t
}

// SetWidth updates the input width.
func (c *InputComponent) SetWidth(w int) {
	c.ti.SetWidth(w - 4) // account for prompt + padding
}

// Update handles messages for the input component. Returns a SubmitMsg command
// when Enter is pressed, or nil otherwise.
func (c *InputComponent) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(c.ti.Value())
			if val == "" {
				return nil
			}
			c.ti.SetValue("")
			return func() tea.Msg { return SubmitMsg{Text: val} }
		}
	}

	var cmd tea.Cmd
	c.ti, cmd = c.ti.Update(msg)
	return cmd
}

// View renders the input field.
func (c InputComponent) View() string {
	return c.theme.Input.Render(c.ti.View())
}
