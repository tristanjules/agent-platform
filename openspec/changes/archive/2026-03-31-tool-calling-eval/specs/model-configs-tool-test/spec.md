## ADDED Requirements

### Requirement: Tool-test TOML configs provide named model presets
Three TOML config files SHALL exist at `configs/tool-test-gemma.toml`, `configs/tool-test-qwen.toml`, and `configs/tool-test-llama.toml`. Each SHALL configure a complete agent inference setup for the named model, derived from `configs/default.toml`, with only `inference.local.model` changed.

#### Scenario: Gemma preset loads without error
- **WHEN** `configs/tool-test-gemma.toml` is loaded via `config.Load("configs/tool-test-gemma.toml")`
- **THEN** `cfg.Inference.Local.Model == "gemma3:1b"` and all other fields are valid

#### Scenario: Qwen preset uses qwen2.5:1.5b
- **WHEN** `configs/tool-test-qwen.toml` is loaded
- **THEN** `cfg.Inference.Local.Model == "qwen2.5:1.5b"`

#### Scenario: Llama preset uses llama3.2:1b
- **WHEN** `configs/tool-test-llama.toml` is loaded
- **THEN** `cfg.Inference.Local.Model == "llama3.2:1b"`

### Requirement: Tool-test configs document tool calling support status
Each tool-test TOML config SHALL include an inline comment above the `model` key indicating the expected tool calling reliability (e.g., `# Tool calling: UNRELIABLE — included for baseline comparison`). This is documentation only; it does not affect runtime behavior.

#### Scenario: Configs contain reliability comments
- **WHEN** any tool-test TOML file is opened
- **THEN** a comment line beginning with `# Tool calling:` is present above the model field
