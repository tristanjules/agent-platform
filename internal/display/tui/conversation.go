package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// conversationEntry holds a single turn (user or agent).
type conversationEntry struct {
	role    string // "user" or "agent"
	content string
}

// ConversationPane displays chat history and streams the current response.
type ConversationPane struct {
	vp        viewport.Model
	entries   []conversationEntry
	streaming string // current streaming response fragment
	theme     Theme
	agentName string
}

// NewConversationPane creates a new conversation pane.
func NewConversationPane(agentName string, theme Theme) ConversationPane {
	vp := viewport.New()
	return ConversationPane{
		vp:        vp,
		agentName: agentName,
		theme:     theme,
	}
}

// SetSize updates the pane dimensions.
func (c *ConversationPane) SetSize(w, h int) {
	c.vp.SetWidth(w)
	c.vp.SetHeight(h)
}

// SetTheme updates the theme.
func (c *ConversationPane) SetTheme(t Theme) {
	c.theme = t
}

// AddUserMessage appends a completed user message to the history.
func (c *ConversationPane) AddUserMessage(text string) {
	c.entries = append(c.entries, conversationEntry{role: "user", content: text})
	c.refreshContent(true)
}

// AppendToken adds a streaming token to the current agent response.
func (c *ConversationPane) AppendToken(token string) {
	c.streaming += token
	c.refreshContent(true)
}

// FinalizeResponse completes the current streaming response and adds it to history.
// If fullResponse is non-empty it is used (reconciles any dropped tokens);
// otherwise the accumulated streaming buffer is used.
func (c *ConversationPane) FinalizeResponse(fullResponse string) {
	content := fullResponse
	if content == "" {
		content = c.streaming
	}
	if content != "" {
		c.entries = append(c.entries, conversationEntry{role: "agent", content: content})
	}
	c.streaming = ""
	c.refreshContent(true)
}

// AddSystemMessage adds a system/command output line.
func (c *ConversationPane) AddSystemMessage(text string) {
	c.entries = append(c.entries, conversationEntry{role: "system", content: text})
	c.refreshContent(true)
}

func (c *ConversationPane) refreshContent(scrollToBottom bool) {
	var sb strings.Builder

	for _, e := range c.entries {
		switch e.role {
		case "user":
			sb.WriteString(c.theme.UserLabel.Render("You: "))
			sb.WriteString(c.theme.Primary.Render(e.content))
		case "agent":
			sb.WriteString(c.theme.AgentLabel.Render(c.agentName + ": "))
			sb.WriteString(c.theme.Accent.Render(e.content))
		case "system":
			sb.WriteString(c.theme.Dimmed.Render("  " + e.content))
		}
		sb.WriteString("\n\n")
	}

	// Show in-progress streaming response with blinking cursor.
	if c.streaming != "" {
		sb.WriteString(c.theme.AgentLabel.Render(c.agentName + ": "))
		sb.WriteString(c.theme.Accent.Render(c.streaming))
		sb.WriteString(c.theme.Cursor.Render("▌"))
		sb.WriteString("\n")
	}

	c.vp.SetContent(sb.String())
	if scrollToBottom {
		c.vp.GotoBottom()
	}
}

// Update handles scroll key messages.
func (c *ConversationPane) Update(msg tea.Msg) {
	c.vp.Update(msg)
}

// View renders the conversation viewport.
func (c ConversationPane) View() string {
	return c.vp.View()
}
