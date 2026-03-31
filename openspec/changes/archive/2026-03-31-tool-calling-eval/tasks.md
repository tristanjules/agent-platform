## 1. Model Config Presets

- [x] 1.1 Create `configs/tool-test-gemma.toml` — copy of `default.toml` with `model = "gemma3:1b"` and inline `# Tool calling: UNRELIABLE — baseline comparison` comment
- [x] 1.2 Create `configs/tool-test-qwen.toml` — same structure with `model = "qwen2.5:1.5b"` and `# Tool calling: EXCELLENT` comment
- [x] 1.3 Create `configs/tool-test-llama.toml` — same structure with `model = "llama3.2:1b"` and `# Tool calling: GOOD` comment

## 2. ModelRouter SupportsToolCalling

- [x] 2.1 Add `SupportsToolCalling() bool` to the `ModelRouter` interface in `internal/agent/router.go`
- [x] 2.2 Define package-level `toolCallingAllowlist []string` with prefixes: `qwen2.5`, `llama3.1`, `llama3.2`, `mistral`, `phi4`, `phi3.5`
- [x] 2.3 Implement `SupportsToolCalling()` on `router` struct: return `true` if `PreferCloud`, or if local model name has a prefix in the allowlist
- [x] 2.4 Add unit tests for `SupportsToolCalling()` — gemma3:1b returns false, qwen2.5:1.5b returns true, cloud routing returns true, unknown model returns false, prefix matching works for long model names

## 3. Integration Test Layer

- [x] 3.1 Create `internal/agent/tool_integration_test.go` with `//go:build integration` tag
- [x] 3.2 Add `ollamaAvailable(t)` helper that dials `OLLAMA_HOST` (default `http://localhost:11434`) and calls `t.Skip` if unreachable
- [x] 3.3 Add `integrationAgent(t)` helper that builds a real `Agent` using `DUSTY_TEST_MODEL` (default `gemma3:1b`) and `OLLAMA_HOST` env vars
- [x] 3.4 Write `TestIntegration_MeshSendToolCall` — sends "send DUSTY-B the message hello", asserts at least one `ToolCall` with name `mesh_send` was invoked (check via a recording InboxTool stub)
- [x] 3.5 Write `TestIntegration_MeshInboxToolCall` — sends "check my unread messages", asserts `mesh_inbox` ToolCall emitted
- [x] 3.6 Write `TestIntegration_FinalResponseNonEmpty` — asserts the streaming final response is non-empty after tool execution
- [x] 3.7 Update `Makefile` with `make test-integration` target: `go test -tags integration -timeout 120s ./internal/agent/...`

## 4. Golden Test Case Data

- [x] 4.1 Create `testdata/eval/` directory
- [x] 4.2 Create `testdata/eval/tool-calling-cases.yaml` with at least 8 cases covering: mesh_send (direct, informal, with punctuation), mesh_inbox (unread check, peer history, peer list), current_time, and an ambiguous prompt that should NOT trigger a tool call
- [x] 4.3 Set `min_success_rate` conservatively: 0.8 for mesh_send/inbox cases, 0.9 for current_time, 0.0 for no-tool case (inverted: measure that no tool is called)

## 5. Eval Runner Binary

- [x] 5.1 Create `cmd/eval/main.go` with CLI flags: `--config`, `--cases`, `--runs` (default 5), `--out` (default `eval-report.json`)
- [x] 5.2 Implement YAML case loader: parse `testdata/eval/tool-calling-cases.yaml` into `[]EvalCase` struct
- [x] 5.3 Implement `runCase(agent, case, runs)` — runs Chat() N times, counts how many produced a ToolCall matching `expected_tool`, returns success count and rate
- [x] 5.4 Implement tool call recorder: a `RecordingTool` wrapper that wraps any `Tool`, records whether it was called, and delegates Execute() — inject into the eval agent instead of real mesh tools
- [x] 5.5 Implement JSON report writer: serialize results to `EvalReport` struct with model name, config path, timestamp, per-case results, overall pass rate
- [x] 5.6 Add stdout live progress: print `[1/8] PASS (0.90) send DUSTY-B hello` after each case
- [x] 5.7 Exit with code 1 if any case's rate < `min_success_rate`
- [x] 5.8 Add `cmd/eval/` to `Makefile`: `make eval` target runs `go run ./cmd/eval/... --config configs/tool-test-gemma.toml --cases testdata/eval/tool-calling-cases.yaml`
- [x] 5.9 Add `make eval-compare` target that runs eval against all three tool-test configs in sequence and prints a summary table

## 6. Documentation

- [x] 6.1 Add `## Testing Tool Calling` section to `README.md` documenting: integration test invocation, eval runner usage, how to add new golden cases, and the model comparison workflow
