## Context

The DUSTY agent currently streams text-only responses from the LLM. Tools exist (`mesh_send`, `mesh_inbox`, `current_time`) with an `Execute(args)` interface, but they are never passed to the model as callable functions. The system prompt describes them textually, relying on the model to output formatted pseudo-calls that are never parsed.

Eino natively supports tool calling: `schema.ToolInfo` defines tool schemas, `model.WithTools()` passes them to the model, and the model returns `schema.ToolCall` entries in its response `Message.ToolCalls`. Results are fed back via `schema.ToolMessage()`. Both the Ollama and Claude Eino adapters support this protocol.

## Goals / Non-Goals

**Goals:**
- Enable the LLM to discover and invoke registered tools via standard tool-calling protocol
- Natural language like "send DUSTY-B hello" resolves to `mesh_send(target="DUSTY-B", message="hello")` automatically
- Maintain streaming UX — the final natural-language response after tool execution still streams token-by-token
- Work with both Ollama (Llama 3.1+, Mistral, Qwen) and Claude models

**Non-Goals:**
- Parallel tool calling (multiple tools in one turn) — single sequential tool call per turn is sufficient for now
- Tool approval/confirmation UX — tools execute immediately without user confirmation
- Adding new tools — this change only wires up existing tools
- Streaming tool call arguments — we collect the full response before parsing tool calls

## Decisions

### 1. Adapt Tool interface to produce Eino ToolInfo

**Decision**: Add a `ToolInfoAdapter` function that converts a `Tool` into `schema.ToolInfo` using `schema.NewParamsOneOfByParams()`. Each tool defines its parameter schema via a new `Params() map[string]*schema.ParameterInfo` method on the Tool interface.

**Why not generate JSON schemas automatically**: The Tool interface is simple and hand-authored. Auto-reflection would add complexity for 2-3 tools. Explicit parameter definitions are clearer and give precise descriptions/types to the model.

### 2. Tool-calling loop inside Chat()

**Decision**: After the initial `Stream()` call, if the assembled response message contains `ToolCalls`, execute each tool, append `ToolMessage` results, and call `Stream()` again. Loop until the model returns a response with no tool calls (pure text). Cap at 5 iterations to prevent runaway loops.

**Why loop in Chat() rather than a separate orchestrator**: Chat() is the single entry point. Adding a separate orchestrator layer would split streaming logic across two places. The loop is simple (under 30 lines) and keeps all inference logic co-located.

**Why not use Eino's built-in agent/chain abstractions**: They add abstraction layers we don't need. Our loop is straightforward and keeps us in control of streaming, memory, and state transitions.

### 3. Use Generate() for tool rounds, Stream() for final response

**Decision**: During tool-calling rounds (model → tool call → tool result → model), use `Generate()` (non-streaming) since the intermediate responses aren't shown to the user. Only the final text response uses `Stream()` for token-by-token delivery.

**Why**: Streaming intermediate tool-call responses adds complexity (assembling partial JSON tool call arguments) with no UX benefit — the user doesn't see these turns.

### 4. Tool registry as a map keyed by name

**Decision**: Build a `map[string]Tool` from `DefaultTools()` at agent construction time. When the model returns a `ToolCall`, look up by `Function.Name`. Unknown tool names return an error message to the model (not a Go error).

**Why return errors to the model**: The model can self-correct ("I called a non-existent tool, let me try the right one"). Failing the whole Chat() call for a model mistake is too harsh.

## Risks / Trade-offs

- **[Model capability variance]** → Not all Ollama models support tool calling well. Mitigation: document recommended models (Llama 3.1+, Qwen 2.5+, Mistral). If a model doesn't support tools, Eino will likely error on the WithTools option — we surface that cleanly.
- **[Runaway tool loops]** → A confused model could call tools indefinitely. Mitigation: hard cap of 5 tool-call rounds per Chat() invocation.
- **[Argument parsing]** → Models may produce malformed JSON arguments. Mitigation: tools already accept `map[string]any` and handle missing/wrong types gracefully with error strings.
- **[Memory bloat]** → Tool call/result messages added to conversation history increase token usage. Mitigation: only the final assistant text response is stored in ConversationMemory, not intermediate tool call/result turns. The full tool-augmented context is ephemeral to the single Chat() call.
