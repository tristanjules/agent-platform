## 1. Tool Interface & Schema Conversion

- [x] 1.1 Add `Params() map[string]*schema.ParameterInfo` method to the `Tool` interface in `internal/agent/tools.go`
- [x] 1.2 Implement `Params()` on `TimeTool` (returns nil — no parameters)
- [x] 1.3 Implement `Params()` on `SendTool` (target: string required, message: string required, type: string optional)
- [x] 1.4 Implement `Params()` on `InboxTool` (action: string optional with enum, peer: string optional, id: string optional)
- [x] 1.5 Add `ToolInfoAdapter(Tool) *schema.ToolInfo` function that converts a Tool to Eino ToolInfo using `schema.NewParamsOneOfByParams()`

## 2. Tool Registry

- [x] 2.1 Add a `toolRegistry map[string]Tool` field to the `Agent` struct, populated from `DefaultTools()` in `NewAgent()`
- [x] 2.2 Add a `toolInfos() []*schema.ToolInfo` method on Agent that converts all registered tools to Eino ToolInfo slices

## 3. Tool-Calling Loop in Chat()

- [x] 3.1 Modify `Chat()` to pass `model.WithTools(toolInfos)` and `model.WithToolChoice(schema.ToolChoiceAllowed)` on inference calls when tools are registered
- [x] 3.2 Implement the tool-calling loop: after `Generate()`, check `Message.ToolCalls`, execute matching tools, append `schema.ToolMessage` results, and re-invoke `Generate()` — up to 5 rounds
- [x] 3.3 Handle unknown tool names by returning an error string in the ToolMessage (not a Go error)
- [x] 3.4 Handle tool execution errors by returning the error string in the ToolMessage
- [x] 3.5 Use `Generate()` (non-streaming) for intermediate tool-calling rounds, and `Stream()` only for the final text response
- [x] 3.6 Ensure only the final assistant text response is stored in ConversationMemory (not intermediate tool call/result messages)

## 4. Tests

- [x] 4.1 Test `ToolInfoAdapter` produces correct ToolInfo for each tool type (TimeTool, SendTool, InboxTool)
- [x] 4.2 Test tool registry lookup — known tool found, unknown tool returns error string
- [x] 4.3 Test tool-calling loop terminates after model returns pure text
- [x] 4.4 Test tool-calling loop terminates at max rounds (5)
- [x] 4.5 Test that ConversationMemory only contains user + final assistant messages after tool-assisted chat
