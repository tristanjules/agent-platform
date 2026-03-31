## Why

The agent has tools defined (`mesh_send`, `mesh_inbox`, `current_time`) but no mechanism for the LLM to actually invoke them. The system prompt describes tools textually, but `Chat()` only streams raw text — there is no tool-calling loop. Users should be able to say "send DUSTY-B a message saying hello" and have the model automatically discover the right tool, call it with correct arguments, and summarize the result. Eino already supports `ToolInfo`, `ToolCall`, and `ToolMessage` natively — we just need to wire it up.

## What Changes

- Convert the existing `Tool` interface into Eino `ToolInfo` schemas so tool definitions are passed to the model alongside each request
- Implement a tool-calling loop in `Chat()`: detect `ToolCall` responses from the model, execute the matching tool, append `ToolMessage` results, and re-invoke the model for a final natural-language response
- Pass tools to the model via Eino's `model.WithTools()` option on `Stream()` calls
- The agent's existing text-only streaming continues to work for non-tool responses — tool calling is additive

## Capabilities

### New Capabilities
- `tool-calling-loop`: The agentic tool-use loop — converting Tool interface to Eino ToolInfo, detecting ToolCall responses, executing tools, feeding results back, and streaming the final response

### Modified Capabilities
- `conversational-agent`: Chat() now supports multi-turn tool-calling before producing a final streamed response

## Impact

- `internal/agent/agent.go` — Chat() gains a tool-calling loop around Stream()
- `internal/agent/tools.go` — Tool interface gains a `Parameters()` method (or adapter) to produce Eino ToolInfo
- `internal/agent/router.go` — Route() or Stream() calls pass `model.WithTools()` option
- No external API changes. The REPL and TUI continue to work as before — they just receive the final streamed response after any tool calls complete internally
