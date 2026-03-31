// Package agent implements the DUSTY conversational agent.
// It wires together the model router, conversation memory, persona,
// state machine, and event bus into a single Chat interface.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/tristanj/dusty/internal/agent/classifier"
	"github.com/tristanj/dusty/internal/agent/training"
	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/state"
)

const maxToolCallRounds = 5

// Agent orchestrates LLM inference, conversation memory, persona, and state.
type Agent struct {
	cfg          *config.Config
	router       ModelRouter
	memory       *ConversationMemory
	persona      Persona
	sm           *state.StateMachine
	bus          *state.EventBus
	log          *slog.Logger
	toolRegistry map[string]Tool
	classifier   classifier.Classifier
	collector    *training.Collector
}

// NewAgentWithRegistry creates an Agent with a pre-built tool registry.
// Used by the eval runner and integration tests to inject recording tools.
// Pass a non-nil cls to enable classification; nil disables it.
func NewAgentWithRegistry(cfg *config.Config, bus *state.EventBus, log *slog.Logger, registry map[string]Tool, cls classifier.Classifier) *Agent {
	sm := state.NewStateMachine(state.StateWarmup, bus)
	mem, _ := NewConversationMemory(cfg.Agent.MaxHistory, "")
	sm.Transition(state.StateIdle) //nolint:errcheck

	// Build training data collector when path is configured.
	var collector *training.Collector
	if cfg != nil && cfg.Inference.Classifier.TrainingDataPath != "" {
		var err error
		collector, err = training.New(cfg.Inference.Classifier.TrainingDataPath, log)
		if err != nil {
			log.Warn("training collector disabled", "err", err)
		}
	}

	return &Agent{
		cfg:          cfg,
		router:       NewModelRouter(cfg),
		memory:       mem,
		persona:      GetPersona(cfg.Agent.Personality),
		sm:           sm,
		bus:          bus,
		log:          log,
		toolRegistry: registry,
		classifier:   cls,
		collector:    collector,
	}
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

	registry := make(map[string]Tool)
	for _, t := range DefaultTools() {
		registry[t.Name()] = t
	}

	// Build classifier when enabled.
	var cls classifier.Classifier
	if cfg.Inference.Classifier.Enabled {
		cls = classifier.NewRuleClassifier(nil)
	}

	// Build training data collector when path is configured.
	collector, err := training.New(cfg.Inference.Classifier.TrainingDataPath, log)
	if err != nil {
		log.Warn("training collector disabled: could not open file",
			"path", cfg.Inference.Classifier.TrainingDataPath, "err", err)
		collector = nil
	}

	a := &Agent{
		cfg:          cfg,
		router:       router,
		memory:       mem,
		persona:      persona,
		sm:           sm,
		bus:          bus,
		log:          log,
		toolRegistry: registry,
		classifier:   cls,
		collector:    collector,
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
//
// When a Classifier is configured, Chat classifies the message before routing
// and gates tool schema injection on the result. When no Classifier is present,
// the existing SupportsToolCalling() gate applies unchanged.
//
// Training data is recorded asynchronously after each call when a Collector
// is configured.
func (a *Agent) Chat(ctx context.Context, userMessage string) (<-chan string, error) {
	if err := a.sm.Transition(state.StateThinking); err != nil {
		return nil, fmt.Errorf("cannot start chat in current state (%s): %w",
			a.sm.Current(), err)
	}

	// Publish the user message event.
	a.bus.Publish(state.NewEvent(state.EventUserMessage, userMessage))

	// Record user turn in memory.
	a.memory.Add("user", userMessage)

	// Classify before routing when a classifier is present.
	var classifyResult classifier.ClassifyResult
	hasClassifier := a.classifier != nil
	if hasClassifier {
		classifyResult = a.classifier.Classify(ctx, userMessage)
	}

	// Build the full message history for the LLM.
	messages, err := a.buildMessages()
	if err != nil {
		_ = a.sm.Transition(state.StateError)
		return nil, fmt.Errorf("building messages: %w", err)
	}

	// Resolve the chat model via the router.
	var chatModel model.BaseChatModel
	if hasClassifier {
		chatModel, err = a.router.RouteWithClassification(ctx, classifyResult)
	} else {
		chatModel, err = a.router.Route(ctx)
	}
	if err != nil {
		_ = a.sm.Transition(state.StateError)
		return nil, fmt.Errorf("routing to model: %w\n\nMake sure Ollama is running: ollama serve", err)
	}

	// Determine the active model name for training data.
	var activeModel string
	if a.cfg != nil {
		activeModel = a.cfg.Inference.Local.Model
		if hasClassifier && a.cfg.Inference.Classifier.Enabled {
			if classifyResult.ShouldUseTool && a.cfg.Inference.Classifier.ToolModel != "" {
				activeModel = a.cfg.Inference.Classifier.ToolModel
			} else if !classifyResult.ShouldUseTool && a.cfg.Inference.Classifier.ChatModel != "" {
				activeModel = a.cfg.Inference.Classifier.ChatModel
			}
		}
	}

	tokens := make(chan string, 32)

	go func() {
		defer close(tokens)

		var fullResponse string
		var toolsCalled []string
		toolSuccess := true

		toolInfos := a.toolInfos()
		shouldInjectTools := a.shouldInjectTools(classifyResult, hasClassifier, toolInfos)

		if shouldInjectTools {
			fullResponse, toolsCalled, toolSuccess = a.runWithTools(ctx, chatModel, messages, toolInfos, tokens)
		} else {
			if len(toolInfos) > 0 && !hasClassifier {
				a.log.Warn("model does not support tool calling; falling back to text-only",
					"model", a.cfg.Inference.Local.Model)
			}
			fullResponse = a.runStream(ctx, chatModel, messages, nil, tokens)
		}

		// Record only the final assistant response in memory.
		if fullResponse != "" {
			a.memory.Add("assistant", fullResponse)
			a.bus.Publish(state.NewEvent(state.EventAgentResponse, fullResponse))
		}

		// Async training data collection.
		a.collector.Record(userMessage, classifyResult, toolsCalled, toolSuccess, activeModel)

		// Transition back to idle.
		if err := a.sm.Transition(state.StateIdle); err != nil {
			a.log.Warn("could not transition to idle after chat", "err", err)
			a.sm.ForceState(state.StateIdle)
		}
	}()

	return tokens, nil
}

// shouldInjectTools determines whether to pass tool schemas to the model.
//
// With classifier:
//   - ShouldUseTool=true  → inject tools (regardless of confidence; we want the model to act)
//   - ShouldUseTool=false, high confidence → skip tools (clear conversation)
//   - Low confidence → inject tools (conservative fallback)
//
// Without classifier: use SupportsToolCalling() as before.
func (a *Agent) shouldInjectTools(result classifier.ClassifyResult, hasClassifier bool, toolInfos []*schema.ToolInfo) bool {
	if len(toolInfos) == 0 {
		return false
	}
	if !hasClassifier {
		return a.router.SupportsToolCalling()
	}

	threshold := 0.7
	if a.cfg != nil && a.cfg.Inference.Classifier.ConfidenceThreshold > 0 {
		threshold = a.cfg.Inference.Classifier.ConfidenceThreshold
	}

	if result.ShouldUseTool {
		return true
	}
	// ShouldUseTool=false: only skip tools when we're confident it's conversation.
	return result.Confidence < threshold
}

// runWithTools executes the tool-calling loop using Generate() for intermediate rounds,
// then streams the final response. Returns the full final response text, tools called, and success.
func (a *Agent) runWithTools(
	ctx context.Context,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
	toolInfos []*schema.ToolInfo,
	tokens chan<- string,
) (string, []string, bool) {
	opts := []model.Option{
		model.WithTools(toolInfos),
		model.WithToolChoice(schema.ToolChoiceAllowed),
	}

	loopMsgs := messages
	toolsWereCalled := false
	var allToolsCalled []string
	toolSuccess := true

	for round := 0; round < maxToolCallRounds; round++ {
		resp, err := chatModel.Generate(ctx, loopMsgs, opts...)
		if err != nil {
			a.log.Error("tool-calling Generate error", "round", round, "err", err)
			a.bus.Publish(state.NewEvent(state.EventAgentError, err))
			return "", allToolsCalled, false
		}

		if len(resp.ToolCalls) == 0 {
			// No tool calls — final response from Generate().
			if toolsWereCalled {
				// Tools were invoked; stream the final response properly.
				resp := a.runStream(ctx, chatModel, loopMsgs, opts, tokens)
				return resp, allToolsCalled, toolSuccess
			}
			// No tools called at all — emit the Generate() text as tokens.
			if resp.Content != "" {
				tokens <- resp.Content
				a.bus.Publish(state.NewEvent(state.EventAgentTokens, resp.Content))
			}
			return resp.Content, allToolsCalled, toolSuccess
		}

		toolsWereCalled = true

		// Append the assistant's tool-call message to the context.
		loopMsgs = append(loopMsgs, resp)

		// Execute each tool call and append results.
		for _, tc := range resp.ToolCalls {
			allToolsCalled = append(allToolsCalled, tc.Function.Name)
			result, execErr := a.executeTool(tc)
			if execErr != nil {
				toolSuccess = false
			}
			loopMsgs = append(loopMsgs, schema.ToolMessage(result, tc.ID))
		}
	}

	// Max rounds hit — do a final stream with whatever context we have.
	a.log.Warn("tool-calling max rounds reached", "max", maxToolCallRounds)
	resp := a.runStream(ctx, chatModel, loopMsgs, opts, tokens)
	return resp, allToolsCalled, toolSuccess
}

// executeTool looks up and runs a tool by name. Returns the result string and any error.
// Unknown tools and execution errors are returned as plain-text error strings
// so the model can self-correct on the next round.
func (a *Agent) executeTool(tc schema.ToolCall) (string, error) {
	tool, ok := a.toolRegistry[tc.Function.Name]
	if !ok {
		err := fmt.Errorf("unknown tool: %s", tc.Function.Name)
		return err.Error(), err
	}

	var args map[string]any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			errMsg := fmt.Sprintf("Invalid tool arguments for %s: %s", tc.Function.Name, err)
			return errMsg, err
		}
	}

	result, err := tool.Execute(args)
	if err != nil {
		return fmt.Sprintf("Tool %s error: %s", tc.Function.Name, err), err
	}
	return result, nil
}

// runStream calls Stream() and pipes tokens to the channel. Returns the full response.
func (a *Agent) runStream(
	ctx context.Context,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
	opts []model.Option,
	tokens chan<- string,
) string {
	streamReader, err := chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		a.log.Error("stream start error", "err", err)
		a.bus.Publish(state.NewEvent(state.EventAgentError, err))
		return ""
	}
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
	return fullResponse
}

// buildMessages constructs the full message slice for the LLM:
// system prompt + conversation history.
func (a *Agent) buildMessages() ([]*schema.Message, error) {
	history := a.memory.Messages()

	// Pre-allocate: system message + all history.
	messages := make([]*schema.Message, 0, 1+len(history))

	// System prompt from persona, with optional mesh addendum.
	systemPrompt := a.persona.SystemPrompt
	if len(MeshTools) > 0 {
		systemPrompt += MeshSystemPromptAddendum
	}
	messages = append(messages, schema.SystemMessage(systemPrompt))

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

// toolInfos converts all registered tools into Eino ToolInfo slices for passing to the model.
// Returns nil if no tools are registered.
func (a *Agent) toolInfos() []*schema.ToolInfo {
	if len(a.toolRegistry) == 0 {
		return nil
	}
	infos := make([]*schema.ToolInfo, 0, len(a.toolRegistry))
	for _, t := range a.toolRegistry {
		infos = append(infos, ToolInfoAdapter(t))
	}
	return infos
}

// ListModels returns info on all configured models.
func (a *Agent) ListModels() []ModelInfo {
	return a.router.ListAvailable()
}

// Close saves memory to disk, flushes training data, and cleans up resources.
func (a *Agent) Close() error {
	a.collector.Close()
	if err := a.memory.Save(); err != nil {
		return fmt.Errorf("saving memory: %w", err)
	}
	a.log.Info("memory saved", "path", a.cfg.Memory.Path)
	return nil
}
