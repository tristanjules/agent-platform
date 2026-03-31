## MODIFIED Requirements

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

## ADDED Requirements

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
