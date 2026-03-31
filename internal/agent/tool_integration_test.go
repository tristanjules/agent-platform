//go:build integration

package agent

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/state"
)

// meshSendTool returns a testTool stub with the same schema as the real mesh_send tool.
func meshSendTool(result string) *testTool {
	return &testTool{
		name:   "mesh_send",
		desc:   "Send a message to another DUSTY node on the mesh network.",
		result: result,
		params: map[string]*schema.ParameterInfo{
			"target":  {Type: schema.String, Desc: "Node ID or callsign of the recipient", Required: true},
			"message": {Type: schema.String, Desc: "Text content of the message", Required: true},
		},
	}
}

// meshInboxTool returns a testTool stub with the same schema as the real mesh_inbox tool.
func meshInboxTool(result string) *testTool {
	return &testTool{
		name:   "mesh_inbox",
		desc:   "Check the inbox for messages received over the mesh network.",
		result: result,
		params: map[string]*schema.ParameterInfo{
			"action": {Type: schema.String, Desc: "Action: list, read, or history", Required: false},
		},
	}
}

// ollamaAvailable dials the Ollama endpoint and skips the test if unreachable.
func ollamaAvailable(t *testing.T) string {
	t.Helper()
	endpoint := os.Getenv("OLLAMA_HOST")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		t.Skipf("Ollama not available at %s: %v", endpoint, err)
	}
	resp.Body.Close()
	return endpoint
}

// integrationCfg builds a config from environment variables.
func integrationCfg(t *testing.T) *config.Config {
	t.Helper()
	endpoint := ollamaAvailable(t)
	modelName := os.Getenv("DUSTY_TEST_MODEL")
	if modelName == "" {
		modelName = "gemma3:1b"
	}
	t.Logf("integration test: model=%s endpoint=%s", modelName, endpoint)

	cfg := &config.Config{}
	cfg.Inference.Mode = "local"
	cfg.Inference.Local.Provider = "ollama"
	cfg.Inference.Local.Model = modelName
	cfg.Inference.Local.Endpoint = endpoint
	cfg.Agent.MaxHistory = 10
	return cfg
}

// integrationAgent builds a real Agent with recording tools injected.
// Errors from the agent are written to t.Log via stderr handler.
func integrationAgent(t *testing.T, recTools ...*RecordingTool) *Agent {
	t.Helper()
	cfg := integrationCfg(t)

	bus := state.NewEventBus(16)
	sm := state.NewStateMachine(state.StateWarmup, bus)
	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatal(err)
	}
	mem, err := NewConversationMemory(10, "")
	if err != nil {
		t.Fatal(err)
	}

	registry := map[string]Tool{}
	for _, rt := range recTools {
		registry[rt.Name()] = rt
	}

	// Write agent logs to stderr so failures are visible in test output.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return &Agent{
		cfg:          cfg,
		router:       NewModelRouter(cfg),
		memory:       mem,
		persona:      GetPersona("default"),
		sm:           sm,
		bus:          bus,
		log:          log,
		toolRegistry: registry,
	}
}

// chatBlockingIntegration runs Chat() with a 90s timeout and returns the full response.
func chatBlockingIntegration(t *testing.T, a *Agent, msg string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ch, err := a.Chat(ctx, msg)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	var sb strings.Builder
	for tok := range ch {
		sb.WriteString(tok)
	}
	return sb.String()
}

// requiresToolCalling skips the test if the current model is not on the tool
// calling allowlist. Use this for tests that assert specific tool invocations.
func requiresToolCalling(t *testing.T, a *Agent) {
	t.Helper()
	if !a.router.SupportsToolCalling() {
		t.Skipf("model %q is not on the tool calling allowlist — "+
			"run with DUSTY_TEST_MODEL=qwen2.5:1.5b to test tool calling",
			a.cfg.Inference.Local.Model)
	}
}

// --- Basic connectivity ---

// TestIntegration_TextResponse validates that the model responds at all.
// Passes for any model — does not require tool calling support.
func TestIntegration_TextResponse(t *testing.T) {
	a := integrationAgent(t)
	response := chatBlockingIntegration(t, a, "Reply with exactly the word: hello")
	if strings.TrimSpace(response) == "" {
		t.Error("expected non-empty text response from model")
	}
	t.Logf("response: %s", response)
}

// TestIntegration_ToolCallingSupported logs whether the active model supports
// tool calling. Informational — never fails.
func TestIntegration_ToolCallingSupported(t *testing.T) {
	a := integrationAgent(t)
	supported := a.router.SupportsToolCalling()
	t.Logf("SupportsToolCalling() = %v for model %q", supported, a.cfg.Inference.Local.Model)
}

// --- Tool calling tests (skip for unsupported models) ---

func TestIntegration_MeshSendToolCall(t *testing.T) {
	send := NewRecordingTool(meshSendTool("Message queued."))
	inbox := NewRecordingTool(meshInboxTool("No unread messages."))
	a := integrationAgent(t, send, inbox)
	requiresToolCalling(t, a)

	chatBlockingIntegration(t, a, "send DUSTY-B the message hello")

	if !send.WasCalled() {
		t.Errorf("expected mesh_send to be invoked, but it was not called")
	}
}

func TestIntegration_MeshInboxToolCall(t *testing.T) {
	send := NewRecordingTool(meshSendTool("Message queued."))
	inbox := NewRecordingTool(meshInboxTool("No unread messages."))
	a := integrationAgent(t, send, inbox)
	requiresToolCalling(t, a)

	chatBlockingIntegration(t, a, "check my unread messages")

	if !inbox.WasCalled() {
		t.Errorf("expected mesh_inbox to be invoked, but it was not called")
	}
}

func TestIntegration_FinalResponseNonEmpty(t *testing.T) {
	send := NewRecordingTool(meshSendTool("Message queued for DUSTY-B."))
	a := integrationAgent(t, send)
	requiresToolCalling(t, a)

	response := chatBlockingIntegration(t, a, "send a message to DUSTY-B saying hi")
	if strings.TrimSpace(response) == "" {
		t.Error("expected non-empty final response after tool execution")
	}
	t.Logf("response: %s", response)
}
