# model-router Specification

## Purpose
TBD - created by archiving change phase-1-foundations. Update Purpose after archive.
## Requirements
### Requirement: Router selects provider based on routing preference
The `ModelRouter` SHALL return an Eino `BaseChatModel` appropriate for the current routing preference: `PreferLocal` always routes to Ollama, `PreferCloud` always routes to Anthropic Claude, and `PreferAuto` tries Ollama first and falls back to Claude on error.

#### Scenario: Local routing with Ollama available
- **WHEN** `router.Route(ctx)` is called with `PreferLocal` and Ollama is reachable
- **THEN** an Ollama-backed `BaseChatModel` is returned with no error

#### Scenario: Auto routing falls back to cloud
- **WHEN** `router.Route(ctx)` is called with `PreferAuto` and Ollama is unreachable
- **THEN** a Claude-backed `BaseChatModel` is returned with no error

#### Scenario: Cloud without API key
- **WHEN** `router.Route(ctx)` is called with `PreferCloud` and `ANTHROPIC_API_KEY` is not set
- **THEN** an error is returned describing that the API key is missing

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

