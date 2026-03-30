package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/tristanj/dusty/internal/state"
)

// ReasoningPane displays state transitions, dimmed token echo, and voice
// pipeline events. It serves as a window into the agent's internal activity.
type ReasoningPane struct {
	vp      viewport.Model
	lines   []string
	theme   Theme
	width   int
	height  int
}

// NewReasoningPane creates a new reasoning pane.
func NewReasoningPane(theme Theme) ReasoningPane {
	vp := viewport.New()
	return ReasoningPane{
		vp:    vp,
		theme: theme,
	}
}

// SetSize updates the pane dimensions.
func (r *ReasoningPane) SetSize(w, h int) {
	r.width = w
	r.height = h
	r.vp.SetWidth(w)
	r.vp.SetHeight(h)
}

// SetTheme updates the theme.
func (r *ReasoningPane) SetTheme(t Theme) {
	r.theme = t
}

// HandleEvent processes an EventMsg and appends relevant content.
func (r *ReasoningPane) HandleEvent(ev state.Event) {
	switch ev.Type {
	case state.EventStateChanged:
		if t, ok := ev.Payload.(state.StateTransition); ok {
			line := r.theme.Dimmed.Render(fmt.Sprintf("[%s → %s]", t.From, t.To))
			r.appendLine(line)
		}

	case state.EventAgentTokens:
		if token, ok := ev.Payload.(string); ok && token != "" {
			// Append tokens to the last line if it's a token line, otherwise start new.
			if len(r.lines) > 0 && strings.HasPrefix(r.lines[len(r.lines)-1], "\x1b") {
				// Check if the last line is a styled reasoning token line (starts with ANSI).
				r.lines[len(r.lines)-1] += token
			} else {
				r.appendLine(r.theme.Dimmed.Render(token))
			}
			r.refreshContent()
		}

	case state.EventSTTResult:
		if text, ok := ev.Payload.(string); ok && text != "" {
			line := r.theme.Dimmed.Render("[STT: " + text + "]")
			r.appendLine(line)
		}

	case state.EventTTSStarted:
		r.appendLine(r.theme.Dimmed.Render("[TTS: synthesizing...]"))

	case state.EventTTSDone:
		r.appendLine(r.theme.Dimmed.Render("[TTS: done]"))
	}
}

// AddSystemLine adds an informational line (e.g. command output).
func (r *ReasoningPane) AddSystemLine(text string) {
	r.appendLine(r.theme.Dimmed.Render(text))
}

func (r *ReasoningPane) appendLine(line string) {
	r.lines = append(r.lines, line)
	// Keep at most 200 lines.
	if len(r.lines) > 200 {
		r.lines = r.lines[len(r.lines)-200:]
	}
	r.refreshContent()
}

func (r *ReasoningPane) refreshContent() {
	r.vp.SetContent(strings.Join(r.lines, "\n"))
	r.vp.GotoBottom()
}

// Update handles scroll key messages.
func (r *ReasoningPane) Update(msg tea.Msg) {
	r.vp.Update(msg)
}

// View renders the scrollable reasoning pane (no border — app.go wraps it).
func (r ReasoningPane) View() string {
	return r.vp.View()
}
