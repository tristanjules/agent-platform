## Context

The `eino-tool-calling` change implemented the tool-calling loop but created an untested assumption: `gemma3:1b` reliably emits Ollama-protocol tool calls. On Pi 5 hardware, this matters because (a) 1B models have borderline structured-output reliability, (b) swapping models is a TOML change not a code change, and (c) Burning Man is an offline, no-retry environment where silent tool-call failures degrade UX with no recovery path.

This change introduces three orthogonal additions: model config presets for A/B testing, an integration test layer for correctness checks, and an eval runner for measuring success rates across model variants.

## Goals / Non-Goals

**Goals:**
- Measure tool call success rate (% of prompts that yield a valid `ToolCall` response) for each model preset
- Provide a reproducible way to compare gemma3:1b, qwen2.5:1.5b, and llama3.2:1b against the same test cases
- Add a `//go:build integration` test layer that validates the full path (Ollama → Agent → ToolCall → Execute) without mocks
- Expose `SupportsToolCalling()` on `ModelRouter` so the agent/TUI can surface a warning if the active model is not known to support tool calling

**Non-Goals:**
- Automated CI with live Ollama (testcontainers adds Docker overhead on Pi; not worth it)
- LLM-as-judge response quality evaluation (scope to tool calling correctness only)
- Fine-tuning or model modification
- Eval of non-tool-calling scenarios (plain chat quality)

## Decisions

### 1. Eval golden cases in YAML, not Go

**Decision**: Store test cases as `testdata/eval/tool-calling-cases.yaml` — a list of `{prompt, expected_tool, min_success_rate, runs}` entries.

**Why YAML over Go test table**: Golden test cases will be edited by non-developers (e.g., adding playa-relevant prompts like "send a message to my camp"). YAML is readable and editable without recompiling. The eval runner loads cases at runtime, so cases can be updated without a rebuild.

**Why not JSON**: YAML supports inline comments for documenting why a case exists and what the expected behaviour is.

### 2. Eval runner as a `cmd/eval/` binary, not a `go test` harness

**Decision**: `cmd/eval/main.go` is a standalone binary invoked as `./dusty-eval --config configs/tool-test-qwen.toml --cases testdata/eval/tool-calling-cases.yaml --runs 10 --out eval-report.json`.

**Why not `go test -run Eval`**: `go test` output format is fixed and poor for tracking metrics over time. A binary can write structured JSON with per-case success rates, timestamps, and model metadata, making it easy to compare runs side-by-side.

**Why not an external eval tool** (promptfoo, RAGAS, etc.): Zero dependencies on Node.js or Python toolchains on Pi. The eval runner can cross-compile to `arm64` and run natively.

### 3. Integration tests use `//go:build integration` tag, not testcontainers

**Decision**: Integration tests in `internal/agent/tool_integration_test.go` require a locally running Ollama instance (checked via `OLLAMA_HOST` env var, defaulting to `http://localhost:11434`). They are skipped in normal `go test` runs.

**Why no testcontainers**: On Pi 5 (the primary target), Docker daemon adds ~300MB RAM overhead. Ollama is already installed natively for development. On CI, the integration tests are intended to run on a self-hosted Pi runner with Ollama pre-installed, not in a cloud x86 container (which wouldn't reflect real ARM64 inference behavior anyway).

**Skip condition**: If Ollama is unreachable, integration tests call `t.Skip("Ollama not available")` rather than failing. This prevents CI failures on machines without Ollama.

### 4. SupportsToolCalling() based on a known-good model list

**Decision**: `ModelRouter` gains `SupportsToolCalling() bool`. The implementation checks the configured local model name against a hardcoded allowlist: `qwen2.5:*`, `llama3.2:*`, `llama3.1:*`, `mistral:*`, `phi4*`. `gemma3:1b` is NOT on the list; `gemma3:2b` and above may be added after empirical validation.

**Why a static list over dynamic detection**: Dynamic detection (send a probe tool call and see if the model responds) adds latency at startup and is brittle. The static list is conservative: better to show a warning on a model that actually works than to silently pass through a model that silently fails.

**Why not a config flag**: Model capability is a property of the model, not the user's deployment. Requiring users to declare `supports_tool_calling = true` in TOML adds friction and is easy to get wrong.

## Risks / Trade-offs

- **[Static allowlist goes stale]** → Gemma3 variants, Phi4-mini, or future models may have improved tool calling that isn't captured. Mitigation: the integration tests and eval runner provide empirical data to update the list; include a config override `tool_calling_override = true` as an escape hatch.
- **[YAML golden cases diverge from real usage]** → Cases authored on a laptop may not reflect the natural language patterns used at Burning Man. Mitigation: design cases from actual mesh prompts observed in mesh-sim testing; add a `--record` mode to the eval runner that captures Chat() I/O for later analysis.
- **[Non-determinism makes success rates noisy]** → LLMs vary between runs. Mitigation: `min_success_rate` threshold at 0.7+ for "good" cases; run `--runs 10` minimum; track 7-day rolling average in reports.
- **[Integration tests slow the feedback loop]** → Running with a real model takes 30-90s. Mitigation: gate behind build tag; never part of the default `go test ./...` run.

## Open Questions

- Should `SupportsToolCalling()` also cover cloud (Claude)? Claude reliably supports tool calling, so it should probably always return `true` for cloud routing.
- What is the minimum acceptable success rate for a model to be "usable" in the field? Suggest 0.75 as the initial threshold, revisable after first eval run.
