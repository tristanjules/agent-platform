package agent_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/agent"
	"github.com/tristanj/dusty/internal/config"
)

func TestSupportsToolCalling_Gemma1b_ReturnsFalse(t *testing.T) {
	r := agent.NewModelRouter(cfgWithModel("gemma3:1b", "local"))
	if r.SupportsToolCalling() {
		t.Error("gemma3:1b should NOT be on the tool calling allowlist")
	}
}

func TestSupportsToolCalling_Qwen25_ReturnsTrue(t *testing.T) {
	r := agent.NewModelRouter(cfgWithModel("qwen2.5:1.5b", "local"))
	if !r.SupportsToolCalling() {
		t.Error("qwen2.5:1.5b should be on the tool calling allowlist")
	}
}

func TestSupportsToolCalling_LlamaLong_ReturnsTrue(t *testing.T) {
	// Verify prefix matching works for fully-qualified model names.
	r := agent.NewModelRouter(cfgWithModel("llama3.2:3b-instruct-q4_K_M", "local"))
	if !r.SupportsToolCalling() {
		t.Error("llama3.2 prefix should match despite long model name suffix")
	}
}

func TestSupportsToolCalling_UnknownModel_ReturnsFalse(t *testing.T) {
	r := agent.NewModelRouter(cfgWithModel("some-unknown-model:7b", "local"))
	if r.SupportsToolCalling() {
		t.Error("unknown model should return false (conservative default)")
	}
}

func TestSupportsToolCalling_CloudRouting_ReturnsTrue(t *testing.T) {
	r := agent.NewModelRouter(cfgWithModel("gemma3:1b", "cloud"))
	if !r.SupportsToolCalling() {
		t.Error("cloud routing should always return true (Claude supports tool calling)")
	}
}

func TestSupportsToolCalling_Mistral_ReturnsTrue(t *testing.T) {
	r := agent.NewModelRouter(cfgWithModel("mistral:7b-instruct", "local"))
	if !r.SupportsToolCalling() {
		t.Error("mistral should be on the tool calling allowlist")
	}
}

func TestSupportsToolCalling_ClassifierEnabled_UsesToolModel(t *testing.T) {
	// When classifier routing is on, SupportsToolCalling should evaluate
	// the tool_model, not the default local model.
	cfg := cfgWithModel("gemma3:1b", "local") // gemma = not on allowlist
	cfg.Inference.Classifier.Enabled = true
	cfg.Inference.Classifier.ToolModel = "qwen2.5:1.5b" // qwen = on allowlist
	cfg.Inference.Classifier.ConfidenceThreshold = 0.7

	r := agent.NewModelRouter(cfg)
	if !r.SupportsToolCalling() {
		t.Error("SupportsToolCalling should evaluate tool_model (qwen2.5), not default (gemma3)")
	}
}

func TestRouteWithClassification_Disabled_UsesDefaultModel(t *testing.T) {
	cfg := cfgWithModel("llama3.2:1b", "local")
	// Classifier disabled — RouteWithClassification should behave like Route().
	cfg.Inference.Classifier.Enabled = false

	r := agent.NewModelRouter(cfg)
	// We can't call Route() in unit tests without a live Ollama, but we can verify
	// SupportsToolCalling reflects the default model, not a non-existent tool model.
	if r.SupportsToolCalling() == false {
		// llama3.2 IS on the allowlist
		t.Error("llama3.2:1b should be on the allowlist")
	}
}

func TestSupportsToolCalling_ClassifierEnabled_NoToolModel_UsesDefault(t *testing.T) {
	cfg := cfgWithModel("llama3.2:1b", "local")
	cfg.Inference.Classifier.Enabled = true
	cfg.Inference.Classifier.ToolModel = "" // not set — falls back to default
	cfg.Inference.Classifier.ConfidenceThreshold = 0.7

	r := agent.NewModelRouter(cfg)
	if !r.SupportsToolCalling() {
		t.Error("llama3.2:1b default should still be on allowlist when tool_model not set")
	}
}

// cfgWithModel builds a minimal config with the given local model and inference mode.
func cfgWithModel(model, mode string) *config.Config {
	cfg := &config.Config{}
	cfg.Inference.Mode = mode
	cfg.Inference.Local.Model = model
	cfg.Inference.Local.Provider = "ollama"
	cfg.Inference.Local.Endpoint = "http://localhost:11434"
	cfg.Inference.Cloud.Model = "claude-sonnet-4-20250514"
	cfg.Inference.Cloud.Provider = "anthropic"
	return cfg
}
