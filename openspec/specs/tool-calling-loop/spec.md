## ADDED Requirements

### Requirement: Tools are converted to Eino ToolInfo schemas
Each registered `Tool` SHALL be convertible to an `schema.ToolInfo` struct containing the tool's name, description, and parameter schema. The parameter schema SHALL use `schema.NewParamsOneOfByParams()` with typed `ParameterInfo` entries.

#### Scenario: mesh_send tool produces valid ToolInfo
- **WHEN** the `mesh_send` tool is converted to `ToolInfo`
- **THEN** the resulting `ToolInfo` has Name="mesh_send", a non-empty Desc, and parameters for "target" (string, required), "message" (string, required), and "type" (string, optional)

#### Scenario: Tool with no parameters produces ToolInfo with nil params
- **WHEN** the `current_time` tool is converted to `ToolInfo`
- **THEN** the resulting `ToolInfo` has Name="current_time", a non-empty Desc, and nil ParamsOneOf

### Requirement: Agent passes tool schemas to the model on each inference call
The agent SHALL pass all registered tools as `schema.ToolInfo` slices to the model via `model.WithTools()` on every Generate/Stream call. Tool choice SHALL be set to `ToolChoiceAllowed` so the model can choose whether to call a tool or respond with text.

#### Scenario: Model receives tool definitions
- **WHEN** `Chat()` is called with tools registered
- **THEN** the model's Generate/Stream call includes `WithTools()` containing all registered tool schemas

#### Scenario: No tools registered
- **WHEN** `Chat()` is called with no tools registered (empty DefaultTools)
- **THEN** the model's Generate/Stream call does not include `WithTools()`

### Requirement: Agent executes tool calls returned by the model
When the model's response contains `ToolCalls`, the agent SHALL look up each tool by name in the tool registry, execute it with the provided arguments, and append a `schema.ToolMessage` with the result. The agent SHALL then re-invoke the model with the updated message history.

#### Scenario: Model calls mesh_send successfully
- **WHEN** the model returns a ToolCall with Function.Name="mesh_send" and arguments `{"target":"DUSTY-B","message":"hello"}`
- **THEN** the agent executes `mesh_send.Execute(args)`, appends a ToolMessage with the result string, and calls the model again

#### Scenario: Model calls unknown tool
- **WHEN** the model returns a ToolCall with Function.Name="nonexistent_tool"
- **THEN** the agent appends a ToolMessage with an error string "Unknown tool: nonexistent_tool" and calls the model again so it can self-correct

#### Scenario: Tool execution returns an error
- **WHEN** a tool's `Execute()` returns a non-nil error
- **THEN** the agent appends a ToolMessage with the error string and calls the model again

### Requirement: Tool-calling loop terminates
The agent SHALL loop through tool-call/result rounds until the model returns a response with no ToolCalls (pure text), OR until a maximum of 5 rounds is reached. If the maximum is reached, the agent SHALL use whatever text content the model has produced so far.

#### Scenario: Model responds with text after one tool call
- **WHEN** the model first returns a ToolCall, the tool is executed, and the model's second response is pure text
- **THEN** the pure text response is streamed to the user and the loop ends

#### Scenario: Maximum rounds exceeded
- **WHEN** the model continues returning ToolCalls for 5 consecutive rounds
- **THEN** the loop terminates and any text content from the last response is used as the final response

### Requirement: Final response is streamed
The final model response (after all tool-call rounds complete) SHALL be streamed token-by-token via `Stream()`. Intermediate tool-calling rounds SHALL use `Generate()` (non-streaming) since their output is not displayed to the user.

#### Scenario: Streaming preserved for final response
- **WHEN** the model produces a final text response after tool calling
- **THEN** tokens are delivered incrementally via the returned `<-chan string`, identical to the existing streaming behavior

### Requirement: Only final response is stored in conversation memory
Intermediate tool-call and tool-result messages SHALL NOT be persisted in `ConversationMemory`. Only the original user message and the final assistant text response SHALL be stored, maintaining the existing memory contract.

#### Scenario: Memory after tool-assisted chat
- **WHEN** a Chat() call involves 2 tool-call rounds before a final text response
- **THEN** `MemoryLen()` increases by exactly 2 (one user message, one assistant response)
