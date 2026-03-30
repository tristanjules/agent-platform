## ADDED Requirements

### Requirement: Config loads from a TOML file with defaults applied first
`config.Load(path)` SHALL parse the TOML file at `path` into a `Config` struct, with `config.Defaults()` applied before parsing so that missing fields retain sensible defaults.

#### Scenario: Missing fields use defaults
- **WHEN** a TOML file specifying only `[agent] name = "TEST"` is loaded
- **THEN** the returned config has `Agent.Name == "TEST"` and all other fields set to their defaults (e.g., `Agent.Personality == "philosopher"`)

#### Scenario: Non-existent file returns error
- **WHEN** `config.Load("/nonexistent.toml")` is called
- **THEN** an error is returned and no config is produced

### Requirement: OLLAMA_HOST env var overrides the Ollama endpoint
If the `OLLAMA_HOST` environment variable is set, it SHALL override `cfg.Inference.Local.Endpoint` after TOML parsing.

#### Scenario: OLLAMA_HOST takes precedence over config file
- **WHEN** `OLLAMA_HOST=http://pi.local:11434` is set and a config file specifying `endpoint = "http://localhost:11434"` is loaded
- **THEN** the resulting config has `Inference.Local.Endpoint == "http://pi.local:11434"`

### Requirement: ANTHROPIC_API_KEY is loaded from env, not config file
The Anthropic API key SHALL be read from the `ANTHROPIC_API_KEY` environment variable during `Load()` and accessible via `config.CloudAPIKey()`. It SHALL NOT appear as a field in the `Config` struct.

#### Scenario: API key accessible after load
- **WHEN** `ANTHROPIC_API_KEY=sk-test` is set and `config.Load()` is called
- **THEN** `config.CloudAPIKey()` returns `"sk-test"`

#### Scenario: API key not in struct
- **WHEN** the Config struct is marshaled to JSON for debugging
- **THEN** no field named `api_key` or similar appears in the output

### Requirement: Full schema is defined for all 5 phases
The `Config` struct SHALL include fields for `Agent`, `Inference`, `STT`, `TTS`, `Display`, and `Memory` sections. TOML files produced in Phase 1 SHALL remain valid (no unmarshaling errors) in all subsequent phases.

#### Scenario: Config with future-phase sections loads without error
- **WHEN** a TOML file containing `[tts]`, `[stt]`, and `[display]` sections is loaded
- **THEN** no error is returned and the fields are populated correctly
