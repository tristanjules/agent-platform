## ADDED Requirements

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
