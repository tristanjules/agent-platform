# model-router Specification

## Purpose
Routes LLM inference requests to the appropriate provider (local Ollama or cloud Anthropic) based on configuration and availability. Exposes model capability metadata including tool calling support.
## Requirements
### Requirement: Router selects provider based on routing preference
The `ModelRouter` SHALL return an Eino `BaseChatModel` appropriate for the current routing preference: `PreferLocal` always routes to Ollama, `PreferCloud` always routes to Anthropic Claude, and `PreferAuto` tries Ollama first and falls back to Claude on error. When classifier-driven routing is enabled, `RouteWithClassification()` SHALL select the model configured for the classification result.

#### Scenario: Local routing with Ollama available
- **WHEN** `router.Route(ctx)` is called with `PreferLocal` and Ollama is reachable
- **THEN** an Ollama-backed `BaseChatModel` is returned with no error

#### Scenario: Auto routing falls back to cloud
- **WHEN** `router.Route(ctx)` is called with `PreferAuto` and Ollama is unreachable
- **THEN** a Claude-backed `BaseChatModel` is returned with no error

#### Scenario: Cloud without API key
- **WHEN** `router.Route(ctx)` is called with `PreferCloud` and `ANTHROPIC_API_KEY` is not set
- **THEN** an error is returned describing that the API key is missing

#### Scenario: Classification-based routing selects tool model
- **WHEN** classifier routing is enabled and `RouteWithClassification(ctx, result)` is called with `ShouldUseTool: true` and `Confidence >= threshold`
- **THEN** the model configured in `inference.classifier.tool_model` is returned

#### Scenario: Classification-based routing selects chat model
- **WHEN** classifier routing is enabled and `RouteWithClassification(ctx, result)` is called with `ShouldUseTool: false` and `Confidence >= threshold`
- **THEN** the model configured in `inference.classifier.chat_model` is returned

#### Scenario: Low-confidence classification uses default model
- **WHEN** classifier routing is enabled and `RouteWithClassification(ctx, result)` is called with `Confidence < threshold`
- **THEN** the default `inference.local.model` is returned (conservative fallback with tools enabled)

#### Scenario: Classifier routing disabled falls back to single model
- **WHEN** `inference.classifier.enabled` is false or absent
- **THEN** `Route()` behaves identically to the current single-model implementation regardless of classification

### Requirement: Router derives preference from config mode string
The `NewModelRouter(cfg)` constructor SHALL initialize routing preference from `cfg.Inference.Mode`: `"local"` → `PreferLocal`, `"cloud"` → `PreferCloud`, `"auto"` → `PreferAuto`. Unrecognized values SHALL default to `PreferLocal`.

#### Scenario: Config mode maps to preference
- **WHEN** `NewModelRouter` is called with `cfg.Inference.Mode = "auto"`
- **THEN** `router.Route(ctx)` attempts local first

### Requirement: Router exposes available model list
The `ModelRouter.ListAvailable()` method SHALL return info on every configured model (name, provider, local flag). It SHALL always include the local Ollama model. It SHALL include the cloud model only if `config.CloudAPIKey()` is non-empty.

#### Scenario: List with only Ollama configured
- **WHEN** `router.ListAvailable()` is called with no API key set
- **THEN** one `ModelInfo` is returned with `Local: true`

#### Scenario: List with both providers configured
- **WHEN** `router.ListAvailable()` is called with an API key set
- **THEN** two `ModelInfo` entries are returned — one local, one cloud

### Requirement: Routing preference is mutable at runtime
`ModelRouter.SetPreference(pref)` SHALL change the routing preference immediately, affecting all subsequent `Route()` calls.

#### Scenario: Preference change takes effect
- **WHEN** `router.SetPreference(PreferCloud)` is called
- **THEN** the next `router.Route(ctx)` uses the cloud provider

### Requirement: ModelRouter exposes a SupportsToolCalling method
`ModelRouter` SHALL expose a `SupportsToolCalling() bool` method that returns `true` if the currently configured model is known to reliably support Ollama's tool calling protocol. The determination SHALL be based on a static allowlist of model name prefixes. Cloud routing (Anthropic) SHALL always return `true`.

#### Scenario: Known-good model returns true
- **WHEN** `SupportsToolCalling()` is called with a router configured for `qwen2.5:1.5b`
- **THEN** it returns `true`

#### Scenario: Gemma 1B returns false
- **WHEN** `SupportsToolCalling()` is called with a router configured for `gemma3:1b`
- **THEN** it returns `false`

#### Scenario: Cloud routing returns true
- **WHEN** `SupportsToolCalling()` is called with routing preference set to `PreferCloud`
- **THEN** it returns `true` regardless of the local model name

#### Scenario: Unknown model returns false
- **WHEN** `SupportsToolCalling()` is called with an unrecognized model name
- **THEN** it returns `false` (conservative default)

### Requirement: SupportsToolCalling allowlist covers commonly used tool-capable models
The static allowlist SHALL include model name prefixes for models with documented Ollama tool calling support: `qwen2.5`, `llama3.1`, `llama3.2`, `mistral`, `phi4`, `phi3.5`. The list SHALL be a package-level variable so it can be extended without interface changes.

#### Scenario: Allowlist prefix matching
- **WHEN** the router model is `llama3.2:3b-instruct-q4_K_M`
- **THEN** `SupportsToolCalling()` returns `true` (prefix `llama3.2` is in the allowlist)

### Requirement: Router supports classifier-driven model configuration
The `ModelRouter` SHALL read an optional `[inference.classifier]` config section containing `enabled` (bool), `tool_model` (string), `chat_model` (string), and `confidence_threshold` (float64, default 0.7). When enabled, these values drive model selection based on `ClassifyResult`.

#### Scenario: Classifier config parsed
- **WHEN** `NewModelRouter(cfg)` is called with `[inference.classifier]` section present and `enabled = true`
- **THEN** the router uses `tool_model` for high-confidence tool classifications and `chat_model` for high-confidence conversation classifications

#### Scenario: Classifier config absent
- **WHEN** `NewModelRouter(cfg)` is called without `[inference.classifier]` section
- **THEN** the router uses the single `inference.local.model` for all requests

#### Scenario: Confidence threshold controls routing
- **WHEN** `RouteWithClassification(ctx, result)` is called with `Confidence = 0.5` and `confidence_threshold = 0.7`
- **THEN** the default model is returned (below threshold, conservative fallback)

### Requirement: SupportsToolCalling respects classifier routing
When classifier routing is enabled, `SupportsToolCalling()` SHALL evaluate the tool model name against the allowlist, not the default local model.

#### Scenario: Tool model is on allowlist
- **WHEN** classifier routing is enabled with `tool_model = "qwen2.5:1.5b"`
- **THEN** `SupportsToolCalling()` returns `true`

#### Scenario: Default model not on allowlist but tool model is
- **WHEN** classifier routing is enabled with `tool_model = "qwen2.5:1.5b"` and default model is `gemma3:1b`
- **THEN** `SupportsToolCalling()` returns `true` (evaluates tool model, not default)
