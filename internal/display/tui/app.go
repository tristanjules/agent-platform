package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/tristanj/dusty/internal/agent"
	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/state"
)

// App is the root Bubble Tea model for DUSTY's TUI.
type App struct {
	// Dependencies
	agent   *agent.Agent
	cfg     *config.Config
	bus     *state.EventBus
	voice   VoiceController
	cmdCtx  context.Context
	cancel  context.CancelFunc

	// Sub-components
	reasoning    ReasoningPane
	conversation ConversationPane
	input        InputComponent
	settings     SettingsOverlay
	commands     CommandHandler

	// Layout state
	theme        Theme
	width        int
	height       int
	agentState   state.AgentState
}

// NewApp creates the root TUI model. Call tea.NewProgram(NewApp(...)) to run.
func NewApp(
	ctx context.Context,
	a *agent.Agent,
	cfg *config.Config,
	bus *state.EventBus,
	voice VoiceController,
) App {
	theme := ThemeByName(cfg.Display.Theme)
	appCtx, cancel := context.WithCancel(ctx)

	return App{
		agent:        a,
		cfg:          cfg,
		bus:          bus,
		voice:        voice,
		cmdCtx:       appCtx,
		cancel:       cancel,
		reasoning:    NewReasoningPane(theme),
		conversation: NewConversationPane(cfg.Agent.Name, theme),
		input:        NewInputComponent(theme),
		settings:     NewSettingsOverlay(a, cfg, theme),
		commands:     NewCommandHandler(a, cfg, voice),
		theme:        theme,
		agentState:   a.State(),
	}
}

// Init is called once when the program starts.
func (m App) Init() tea.Cmd {
	return nil
}

// Update handles all incoming messages.
func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.relayout()
		return m, nil

	case EventMsg:
		m = m.handleEvent(msg.Event)
		return m, nil

	case SubmitMsg:
		return m, m.handleSubmit(msg.Text)

	case tea.KeyPressMsg:
		// Settings overlay absorbs all keys when visible.
		if m.settings.Visible() {
			result := m.settings.Update(msg)
			if result.Output != "" {
				m.conversation.AddSystemMessage(result.Output)
				m.reasoning.AddSystemLine(result.Output)
			}
			if result.Quit {
				m.cancel()
				return m, tea.Quit
			}
			return m, nil
		}

		// Global hotkeys.
		switch msg.String() {
		case "ctrl+c":
			_ = m.agent.Close()
			m.cancel()
			return m, tea.Quit

		case "ctrl+l":
			m.agent.ClearMemory()
			m.conversation.AddSystemMessage("[memory cleared]")
			m.reasoning.AddSystemLine("[memory cleared]")
			return m, nil

		case "f1":
			out := m.voice.ToggleVoice()
			if out == "" {
				if m.voice.Active() {
					out = "[voice mode started — speak to DUSTY]"
				} else {
					out = "[voice mode stopped]"
				}
			}
			m.conversation.AddSystemMessage(out)
			m.reasoning.AddSystemLine(out)
			return m, nil

		case "f2":
			m.settings.Toggle()
			return m, nil

		case "tab":
			// Reserved for visual mode (Phase 4).
			m.conversation.AddSystemMessage("[visual mode not yet available — coming in Phase 4]")
			return m, nil
		}
	}

	// Forward remaining key/mouse messages to input and viewports.
	cmd := m.input.Update(msg)
	m.reasoning.Update(msg)
	m.conversation.Update(msg)
	return m, cmd
}

// handleEvent processes EventBus events received via the bridge.
func (m App) handleEvent(ev state.Event) App {
	switch ev.Type {
	case state.EventStateChanged:
		if t, ok := ev.Payload.(state.StateTransition); ok {
			m.agentState = t.To
		}
		m.reasoning.HandleEvent(ev)

	case state.EventAgentTokens:
		if token, ok := ev.Payload.(string); ok {
			m.conversation.AppendToken(token)
		}
		m.reasoning.HandleEvent(ev)

	case state.EventAgentResponse:
		if full, ok := ev.Payload.(string); ok {
			m.conversation.FinalizeResponse(full)
		} else {
			m.conversation.FinalizeResponse("")
		}

	case state.EventAgentError:
		if err, ok := ev.Payload.(error); ok {
			m.conversation.AddSystemMessage(m.theme.Error.Render("Error: " + err.Error()))
		}

	case state.EventSTTResult, state.EventTTSStarted, state.EventTTSDone:
		m.reasoning.HandleEvent(ev)
	}
	return m
}

// handleSubmit processes user input (message or slash command).
func (m *App) handleSubmit(text string) tea.Cmd {
	if strings.HasPrefix(text, "/") {
		result := m.commands.Handle(text)
		if result.Output != "" {
			m.conversation.AddSystemMessage(result.Output)
			m.reasoning.AddSystemLine(result.Output)
		}
		if result.Quit {
			m.cancel()
			return tea.Quit
		}
		return nil
	}

	// Normal message — show in conversation immediately, then send to agent.
	m.conversation.AddUserMessage(text)

	ctx := m.cmdCtx
	a := m.agent
	return func() tea.Msg {
		tokens, err := a.Chat(ctx, text)
		if err != nil {
			return EventMsg{Event: state.NewEvent(state.EventAgentError, err)}
		}
		// Drain the token channel; the EventBus bridge already forwards tokens
		// to the TUI via EventAgentTokens events.
		go func() {
			for range tokens {
			}
		}()
		return nil
	}
}

// relayout recalculates pane sizes based on terminal dimensions.
func (m *App) relayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	const (
		statusBarHeight = 1
		hotkeyBarHeight = 1
		inputHeight     = 3
		borderOverhead  = 2 // top + bottom border lines per pane
	)

	availableHeight := m.height - statusBarHeight - hotkeyBarHeight - inputHeight

	// Reasoning gets 25%, conversation gets 75% of available height.
	reasoningHeight := availableHeight / 4
	if reasoningHeight < 3 {
		reasoningHeight = 3
	}
	convHeight := availableHeight - reasoningHeight

	innerWidth := m.width - borderOverhead

	m.reasoning.SetSize(innerWidth, reasoningHeight-borderOverhead)
	m.conversation.SetSize(innerWidth, convHeight-borderOverhead)
	m.input.SetWidth(m.width)
	m.settings.SetSize(m.width, m.height)
}

// View renders the full TUI.
func (m App) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	var sb strings.Builder

	// 1. Status bar.
	sb.WriteString(RenderStatusBar(
		m.width,
		m.cfg.Agent.Name,
		m.cfg.Inference.Local.Model,
		m.cfg.Inference.Mode,
		m.agentState,
		m.theme,
	))
	sb.WriteString("\n")

	// 2. Reasoning pane.
	sb.WriteString(RenderPane("Reasoning", m.reasoning.View(), m.width, m.height/4, m.theme))
	sb.WriteString("\n")

	// 3. Conversation pane — remaining height.
	convHeight := m.height - 1 - 1 - 3 - m.height/4 // total - status - hotkey - input - reasoning
	if convHeight < 5 {
		convHeight = 5
	}
	sb.WriteString(RenderPane("Conversation", m.conversation.View(), m.width, convHeight, m.theme))
	sb.WriteString("\n")

	// 4. Input pane.
	sb.WriteString(RenderPane("Input", m.input.View(), m.width, 3, m.theme))
	sb.WriteString("\n")

	// 5. Hotkey bar.
	sb.WriteString(RenderHotkeyBar(m.width, m.voice.Active(), m.theme))

	content := sb.String()

	// Overlay settings panel if visible.
	if m.settings.Visible() {
		// Append settings view below the main UI (simple approach for now).
		content += "\n" + m.settings.View()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
