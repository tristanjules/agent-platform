## Why

The `eino-tool-calling` change wired up tool invocation but left an open question: does `gemma3:1b` (the default Pi 5 model) actually emit valid Ollama tool calls reliably? Without integration tests and a measurable success rate, there is no way to validate that tool calling works before deploying to the field, or to compare models when tuning for Burning Man conditions.

## What Changes

- **Model configs for tool-calling validation** — new TOML presets (`configs/tool-test-gemma.toml`, `configs/tool-test-qwen.toml`) so different models can be A/B tested against the same tool schemas without code changes
- **Integration test layer** — a `//go:build integration` test suite that runs the full tool-calling path against a real local Ollama instance and asserts that tool calls were emitted and executed correctly
- **Eval runner** (`cmd/eval/`) — a command-line harness that loads golden test cases, runs the agent N times per case, measures per-tool success rates, and writes a JSON report for tracking quality over time
- **A/B model comparison** — the eval runner accepts a `--config` flag so the same golden cases can be run against different model configs and results compared

## Capabilities

### New Capabilities

- `integration-test-layer`: Build-tagged integration tests that validate tool calling end-to-end with a real Ollama model
- `eval-runner`: CLI harness for running golden test cases against the agent and measuring tool call success rates
- `model-configs-tool-test`: Named TOML config presets for tool-calling-oriented model selection (gemma3:1b, qwen2.5:1.5b, llama3.2:1b)

### Modified Capabilities

- `model-router`: Add a capability flag (`SupportsToolCalling() bool`) so callers can detect whether the selected model is known to support Ollama's tool calling protocol reliably

## Impact

- New directory: `cmd/eval/` (eval runner binary)
- New files: `configs/tool-test-gemma.toml`, `configs/tool-test-qwen.toml`, `configs/tool-test-llama.toml`
- New test files: `internal/agent/tool_integration_test.go` (build tag: `integration`)
- Modified: `internal/agent/router.go` — adds `SupportsToolCalling()` to `ModelRouter` interface
- New golden test case data: `testdata/eval/tool-calling-cases.yaml`
- No changes to core agent logic, existing unit tests, or mesh system
