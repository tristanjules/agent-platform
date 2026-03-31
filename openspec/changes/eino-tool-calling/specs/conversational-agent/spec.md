## MODIFIED Requirements

### Requirement: Agent exposes streaming Chat interface
The agent SHALL accept a user message string and return a channel of string tokens that streams the LLM response incrementally. The channel SHALL be closed when the response is complete. When registered tools are available, the agent SHALL pass tool schemas to the model and execute any tool calls before streaming the final response. The tool-calling loop is transparent to callers — they receive only the final streamed text.

#### Scenario: Successful streaming chat
- **WHEN** `agent.Chat(ctx, "hello")` is called while the agent is in Idle state
- **THEN** a `<-chan string` is returned with no error, tokens arrive incrementally, and the channel is closed when the response is finished

#### Scenario: Chat while not idle
- **WHEN** `agent.Chat(ctx, message)` is called while the agent is in Thinking state
- **THEN** an error is returned and no channel is produced

#### Scenario: Chat with tool invocation
- **WHEN** `agent.Chat(ctx, "send DUSTY-B a message saying hello")` is called with mesh tools registered
- **THEN** the model receives tool schemas, may return a ToolCall, the tool is executed internally, and the final text response is streamed via the returned channel
