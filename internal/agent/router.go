package agent

import (
	"context"
	"fmt"
	"time"

	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/model"

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

// ModelRouter selects the appropriate ChatModel based on routing preference
// and availability. It abstracts over local (Ollama) and cloud (Anthropic)
// providers behind Eino's BaseChatModel interface.
type ModelRouter interface {
	// Route returns a BaseChatModel ready for inference.
	Route(ctx context.Context) (model.BaseChatModel, error)
	// SetPreference changes the routing strategy at runtime.
	SetPreference(pref RoutingPreference)
	// ListAvailable returns info on which models are configured.
	ListAvailable() []ModelInfo
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
		return r.localModel(ctx)
	case PreferCloud:
		return r.cloudModel(ctx)
	case PreferAuto:
		m, err := r.localModel(ctx)
		if err == nil {
			return m, nil
		}
		// Local unavailable — try cloud.
		return r.cloudModel(ctx)
	default:
		return r.localModel(ctx)
	}
}

// localModel creates an Ollama-backed ChatModel.
func (r *router) localModel(_ context.Context) (model.BaseChatModel, error) {
	local := r.cfg.Inference.Local
	if local.Endpoint == "" {
		local.Endpoint = "http://localhost:11434"
	}

	m, err := einoollama.NewChatModel(context.Background(), &einoollama.ChatModelConfig{
		BaseURL: local.Endpoint,
		Model:   local.Model,
		Timeout: 120 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Ollama chat model (%s @ %s): %w",
			local.Model, local.Endpoint, err)
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
