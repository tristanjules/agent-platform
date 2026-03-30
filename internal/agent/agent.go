// Package agent implements the DUSTY conversational agent.
// It wires together the model router, conversation memory, persona,
// state machine, and event bus into a single Chat interface.
package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/cloudwego/eino/schema"

	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/state"
)

// Agent orchestrates LLM inference, conversation memory, persona, and state.
type Agent struct {
	cfg     *config.Config
	router  ModelRouter
	memory  *ConversationMemory
	persona Persona
	sm      *state.StateMachine
	bus     *state.EventBus
	log     *slog.Logger
}

// NewAgent creates and initializes the agent from the given config.
// It loads conversation history from disk if persistence is configured.
func NewAgent(cfg *config.Config, bus *state.EventBus, log *slog.Logger) (*Agent, error) {
	sm := state.NewStateMachine(state.StateWarmup, bus)

	// Load conversation memory.
	var persistPath string
	if cfg.Memory.Persist {
		persistPath = cfg.Memory.Path
	}
	mem, err := NewConversationMemory(cfg.Agent.MaxHistory, persistPath)
	if err != nil {
		return nil, fmt.Errorf("initializing memory: %w", err)
	}

	persona := GetPersona(cfg.Agent.Personality)
	router := NewModelRouter(cfg)

	a := &Agent{
		cfg:     cfg,
		router:  router,
		memory:  mem,
		persona: persona,
		sm:      sm,
		bus:     bus,
		log:     log,
	}

	// Transition out of warmup.
	if err := sm.Transition(state.StateIdle); err != nil {
		return nil, fmt.Errorf("state machine warmup: %w", err)
	}

	if mem.Len() > 0 {
		log.Info("loaded conversation history", "messages", mem.Len())
	}

	return a, nil
}

// Chat sends a user message to the LLM and returns a channel of streaming
// token strings. The channel is closed when the response is complete.
// The caller must drain the channel to avoid goroutine leaks.
func (a *Agent) Chat(ctx context.Context, userMessage string) (<-chan string, error) {
	if err := a.sm.Transition(state.StateThinking); err != nil {
		return nil, fmt.Errorf("cannot start chat in current state (%s): %w",
			a.sm.Current(), err)
	}

	// Publish the user message event.
	a.bus.Publish(state.NewEvent(state.EventUserMessage, userMessage))

	// Record user turn in memory.
	a.memory.Add("user", userMessage)

	// Build the full message history for the LLM.
	messages, err := a.buildMessages()
	if err != nil {
		_ = a.sm.Transition(state.StateError)
		return nil, fmt.Errorf("building messages: %w", err)
	}

	// Resolve the chat model via the router.
	chatModel, err := a.router.Route(ctx)
	if err != nil {
		_ = a.sm.Transition(state.StateError)
		return nil, fmt.Errorf("routing to model: %w\n\nMake sure Ollama is running: ollama serve", err)
	}

	// Start streaming inference.
	streamReader, err := chatModel.Stream(ctx, messages)
	if err != nil {
		_ = a.sm.Transition(state.StateError)
		return nil, fmt.Errorf("starting LLM stream: %w", err)
	}

	tokens := make(chan string, 32)

	go func() {
		defer close(tokens)
		defer streamReader.Close()

		var fullResponse string

		for {
			chunk, err := streamReader.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				a.log.Error("stream error", "err", err)
				a.bus.Publish(state.NewEvent(state.EventAgentError, err))
				break
			}

			if chunk.Content != "" {
				tokens <- chunk.Content
				fullResponse += chunk.Content
				a.bus.Publish(state.NewEvent(state.EventAgentTokens, chunk.Content))
			}
		}

		// Record the complete assistant response in memory.
		if fullResponse != "" {
			a.memory.Add("assistant", fullResponse)
			a.bus.Publish(state.NewEvent(state.EventAgentResponse, fullResponse))
		}

		// Transition back to idle.
		if err := a.sm.Transition(state.StateIdle); err != nil {
			a.log.Warn("could not transition to idle after chat", "err", err)
			a.sm.ForceState(state.StateIdle)
		}
	}()

	return tokens, nil
}

// buildMessages constructs the full message slice for the LLM:
// system prompt + conversation history.
func (a *Agent) buildMessages() ([]*schema.Message, error) {
	history := a.memory.Messages()

	// Pre-allocate: system message + all history.
	messages := make([]*schema.Message, 0, 1+len(history))

	// System prompt from persona.
	messages = append(messages, schema.SystemMessage(a.persona.SystemPrompt))

	// Conversation history.
	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, schema.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(msg.Content, nil))
		// Skip system messages from history — we prepend the current system prompt.
		}
	}

	return messages, nil
}

// SetPersona swaps the active persona at runtime.
func (a *Agent) SetPersona(name string) {
	a.persona = GetPersona(name)
	a.log.Info("persona changed", "name", name)
}

// SetRoutingPreference changes the model routing strategy at runtime.
func (a *Agent) SetRoutingPreference(pref RoutingPreference) {
	a.router.SetPreference(pref)
}

// ClearMemory wipes conversation history.
func (a *Agent) ClearMemory() {
	a.memory.Clear()
	a.log.Info("conversation memory cleared")
}

// State returns the current agent state.
func (a *Agent) State() state.AgentState {
	return a.sm.Current()
}

// MemoryLen returns the number of messages in conversation history.
func (a *Agent) MemoryLen() int {
	return a.memory.Len()
}

// ListModels returns info on all configured models.
func (a *Agent) ListModels() []ModelInfo {
	return a.router.ListAvailable()
}

// Close saves memory to disk and cleans up resources.
func (a *Agent) Close() error {
	if err := a.memory.Save(); err != nil {
		return fmt.Errorf("saving memory: %w", err)
	}
	a.log.Info("memory saved", "path", a.cfg.Memory.Path)
	return nil
}
