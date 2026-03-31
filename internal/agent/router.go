package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/model"

	"github.com/tristanj/dusty/internal/agent/classifier"
	"github.com/tristanj/dusty/internal/config"
)

// RoutingPreference controls where inference happens.
type RoutingPreference int

const (
	// PreferLocal routes to Ollama. Fails if Ollama is unreachable.
	PreferLocal RoutingPreference = iota
	// PreferCloud routes to the configured cloud provider.
	PreferCloud
	// PreferAuto tries local first, falls back to cloud on error.
	PreferAuto
)

// ModelInfo describes an available model.
type ModelInfo struct {
	Name     string
	Provider string // "ollama", "anthropic"
	Local    bool
}

// toolCallingAllowlist contains model name prefixes known to reliably support
// Ollama's tool calling protocol. Conservative by design: models not listed
// return false from SupportsToolCalling() until empirically validated.
var toolCallingAllowlist = []string{
	"qwen2.5",
	"llama3.1",
	"llama3.2",
	"mistral",
	"phi4",
	"phi3.5",
}

// defaultConfidenceThreshold is used when ClassifierConfig.ConfidenceThreshold is zero.
const defaultConfidenceThreshold = 0.7

// ModelRouter selects the appropriate ChatModel based on routing preference
// and availability. It abstracts over local (Ollama) and cloud (Anthropic)
// providers behind Eino's BaseChatModel interface.
type ModelRouter interface {
	// Route returns a BaseChatModel ready for inference using the configured preference.
	Route(ctx context.Context) (model.BaseChatModel, error)
	// RouteWithClassification returns a BaseChatModel selected based on the classifier
	// result. When classifier routing is disabled or confidence is below threshold,
	// it falls back to Route().
	RouteWithClassification(ctx context.Context, result classifier.ClassifyResult) (model.BaseChatModel, error)
	// SetPreference changes the routing strategy at runtime.
	SetPreference(pref RoutingPreference)
	// ListAvailable returns info on which models are configured.
	ListAvailable() []ModelInfo
	// SupportsToolCalling reports whether the currently active model is known
	// to reliably support Ollama's tool calling protocol. When classifier routing
	// is enabled, evaluates the tool model. Cloud routing always returns true.
	SupportsToolCalling() bool
}

type router struct {
	cfg        *config.Config
	preference RoutingPreference
}

// NewModelRouter creates a router from the loaded configuration.
// The router is lazy — it creates ChatModel instances on each Route() call
// so that connectivity is checked at inference time, not startup.
func NewModelRouter(cfg *config.Config) ModelRouter {
	pref := PreferLocal
	switch cfg.Inference.Mode {
	case "cloud":
		pref = PreferCloud
	case "auto":
		pref = PreferAuto
	}
	return &router{cfg: cfg, preference: pref}
}

func (r *router) SetPreference(pref RoutingPreference) {
	r.preference = pref
}

func (r *router) confidenceThreshold() float64 {
	t := r.cfg.Inference.Classifier.ConfidenceThreshold
	if t <= 0 {
		return defaultConfidenceThreshold
	}
	return t
}

func (r *router) SupportsToolCalling() bool {
	if r.preference == PreferCloud {
		return true
	}
	// When classifier routing is enabled, evaluate the tool model.
	modelName := r.cfg.Inference.Local.Model
	if r.cfg.Inference.Classifier.Enabled && r.cfg.Inference.Classifier.ToolModel != "" {
		modelName = r.cfg.Inference.Classifier.ToolModel
	}
	for _, prefix := range toolCallingAllowlist {
		if strings.HasPrefix(modelName, prefix) {
			return true
		}
	}
	return false
}

func (r *router) ListAvailable() []ModelInfo {
	models := []ModelInfo{
		{
			Name:     r.cfg.Inference.Local.Model,
			Provider: r.cfg.Inference.Local.Provider,
			Local:    true,
		},
	}
	if config.CloudAPIKey() != "" {
		models = append(models, ModelInfo{
			Name:     r.cfg.Inference.Cloud.Model,
			Provider: r.cfg.Inference.Cloud.Provider,
			Local:    false,
		})
	}
	return models
}

func (r *router) Route(ctx context.Context) (model.BaseChatModel, error) {
	switch r.preference {
	case PreferLocal:
		return r.localModel(ctx, r.cfg.Inference.Local.Model)
	case PreferCloud:
		return r.cloudModel(ctx)
	case PreferAuto:
		m, err := r.localModel(ctx, r.cfg.Inference.Local.Model)
		if err == nil {
			return m, nil
		}
		return r.cloudModel(ctx)
	default:
		return r.localModel(ctx, r.cfg.Inference.Local.Model)
	}
}

// RouteWithClassification selects a model based on the classifier result.
// When classifier routing is disabled, or when cloud routing is active,
// it delegates to Route(). When confidence is below threshold, it falls back
// to the default local model with tools available (conservative).
func (r *router) RouteWithClassification(ctx context.Context, result classifier.ClassifyResult) (model.BaseChatModel, error) {
	cls := r.cfg.Inference.Classifier

	// Cloud routing or classifier disabled — use standard routing.
	if r.preference == PreferCloud || !cls.Enabled {
		return r.Route(ctx)
	}

	// Below confidence threshold — use default model (conservative fallback).
	if result.Confidence < r.confidenceThreshold() {
		return r.localModel(ctx, r.cfg.Inference.Local.Model)
	}

	// Route based on classification.
	if result.ShouldUseTool && cls.ToolModel != "" {
		return r.localModel(ctx, cls.ToolModel)
	}
	if !result.ShouldUseTool && cls.ChatModel != "" {
		return r.localModel(ctx, cls.ChatModel)
	}

	// Classifier enabled but no per-intent model configured — use default.
	return r.localModel(ctx, r.cfg.Inference.Local.Model)
}

// localModel creates an Ollama-backed ChatModel for the given model name.
func (r *router) localModel(_ context.Context, modelName string) (model.BaseChatModel, error) {
	local := r.cfg.Inference.Local
	endpoint := local.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	m, err := einoollama.NewChatModel(context.Background(), &einoollama.ChatModelConfig{
		BaseURL: endpoint,
		Model:   modelName,
		Timeout: 120 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Ollama chat model (%s @ %s): %w",
			modelName, endpoint, err)
	}
	return m, nil
}

// cloudModel creates an Anthropic Claude-backed ChatModel.
func (r *router) cloudModel(_ context.Context) (model.BaseChatModel, error) {
	apiKey := config.CloudAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("cloud inference requested but ANTHROPIC_API_KEY is not set")
	}

	cloud := r.cfg.Inference.Cloud
	maxTokens := 4096

	m, err := einoclaude.NewChatModel(context.Background(), &einoclaude.Config{
		APIKey:    apiKey,
		Model:     cloud.Model,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Claude chat model (%s): %w", cloud.Model, err)
	}
	return m, nil
}
