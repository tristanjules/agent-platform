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
The agent SHALL pass all registered tools as `schema.ToolInfo` slices to the model via `model.WithTools()` on Generate/Stream calls based on the classifier result. When `ClassifyResult.ShouldUseTool` is true (at any confidence), tools SHALL be injected. When `ShouldUseTool` is false and `Confidence >= confidence_threshold`, tools SHALL NOT be injected. When `Confidence < confidence_threshold`, tools SHALL be injected as a conservative fallback. When no classifier is configured, the existing `SupportsToolCalling()` gate applies.

#### Scenario: High-confidence tool classification includes tool schemas
- **WHEN** `Chat()` is called with tools registered and classifier returns `ShouldUseTool: true, Confidence: 0.95`
- **THEN** the model's Generate/Stream call includes `WithTools()` containing all registered tool schemas

#### Scenario: High-confidence conversation classification skips tool schemas
- **WHEN** `Chat()` is called with tools registered and classifier returns `ShouldUseTool: false, Confidence: 0.95`
- **THEN** the model's Generate/Stream call does NOT include `WithTools()`

#### Scenario: Low-confidence classification includes tool schemas as fallback
- **WHEN** `Chat()` is called with tools registered and classifier returns `Confidence: 0.4` (below threshold)
- **THEN** the model's Generate/Stream call includes `WithTools()` regardless of `ShouldUseTool` value

#### Scenario: No classifier configured preserves current behavior
- **WHEN** `Chat()` is called with no classifier configured (nil)
- **THEN** tool schema injection uses the existing `SupportsToolCalling()` gate (current behavior)

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

### Requirement: Agent accepts an optional Classifier
The `Agent` struct SHALL accept an optional `Classifier` via constructor or setter. When present, `Chat()` SHALL classify the user message before routing. When absent, the agent SHALL behave identically to the current implementation.

#### Scenario: Agent with classifier
- **WHEN** `NewAgent()` is called with a classifier configured
- **THEN** the agent stores the classifier and uses it in `Chat()`

#### Scenario: Agent without classifier
- **WHEN** `NewAgent()` is called without a classifier (nil or not configured)
- **THEN** `Chat()` skips classification and uses `SupportsToolCalling()` as before
