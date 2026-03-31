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
