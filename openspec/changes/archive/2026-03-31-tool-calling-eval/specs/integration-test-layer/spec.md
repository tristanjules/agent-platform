## ADDED Requirements

### Requirement: Integration tests validate tool calling end-to-end with real Ollama
The `internal/agent/tool_integration_test.go` file SHALL carry a `//go:build integration` build tag and SHALL test the complete tool-calling path (Chat() → model Generate() → ToolCall → executeTool() → result) using a real Ollama instance. Tests SHALL skip (not fail) if Ollama is unreachable at the configured endpoint.

#### Scenario: Integration test skips when Ollama is unavailable
- **WHEN** the integration test suite runs and Ollama is not reachable at `OLLAMA_HOST` (or `http://localhost:11434`)
- **THEN** all integration tests call `t.Skip("Ollama not available: ...")` and the suite exits with a skip, not a failure

#### Scenario: Tool call emitted for mesh send prompt
- **WHEN** the agent receives `"send DUSTY-B the message hello"` with `mesh_send` registered and the real Ollama model loaded
- **THEN** the model emits at least one `ToolCall` with `Function.Name == "mesh_send"` within the configured max rounds

#### Scenario: Tool call emitted for inbox check prompt
- **WHEN** the agent receives `"check my unread messages"` with `mesh_inbox` registered
- **THEN** the model emits at least one `ToolCall` with `Function.Name == "mesh_inbox"`

#### Scenario: Final response is non-empty after tool execution
- **WHEN** a tool call is executed successfully
- **THEN** the final streamed response text is non-empty (the model produced a summary, not silence)

### Requirement: Integration tests are excluded from default test runs
Integration tests SHALL NOT run when `go test ./...` is executed without build tags. They SHALL only run when explicitly opted in with `-tags integration`.

#### Scenario: Default test run excludes integration tests
- **WHEN** `go test ./internal/agent/...` is executed without `-tags integration`
- **THEN** `tool_integration_test.go` is not compiled and integration tests do not run

#### Scenario: Integration tests run with explicit tag
- **WHEN** `go test -tags integration ./internal/agent/...` is executed with Ollama available
- **THEN** integration tests run and report results

### Requirement: Integration test model is configurable via environment variable
The integration tests SHALL read the Ollama model name from `DUSTY_TEST_MODEL` (defaulting to `gemma3:1b`) and the endpoint from `OLLAMA_HOST` (defaulting to `http://localhost:11434`). This allows different models to be tested without code changes.

#### Scenario: Custom model used in integration test
- **WHEN** `DUSTY_TEST_MODEL=qwen2.5:1.5b go test -tags integration ./internal/agent/...` is executed
- **THEN** integration tests run against `qwen2.5:1.5b`
