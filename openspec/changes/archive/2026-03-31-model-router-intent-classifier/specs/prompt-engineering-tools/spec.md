## ADDED Requirements

### Requirement: MeshSystemPromptAddendum includes few-shot tool-call examples
The `MeshSystemPromptAddendum` string SHALL include at least one few-shot example for each registered mesh tool (mesh_send, mesh_inbox) and for current_time. Each example SHALL show a user message and the expected tool call with arguments.

#### Scenario: Few-shot example for mesh_send
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains an example showing a user asking to send a message to another node and the corresponding mesh_send tool invocation with target and message arguments

#### Scenario: Few-shot example for mesh_inbox
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains an example showing a user asking about unread messages and the corresponding mesh_inbox tool invocation

#### Scenario: Few-shot example for current_time
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains an example showing a user asking what time it is and the corresponding current_time tool invocation

### Requirement: System prompt includes negative examples for no-tool scenarios
The `MeshSystemPromptAddendum` SHALL include explicit guidance that the model MUST NOT call tools for philosophical questions, general knowledge, emotional conversations, or any prompt that does not require mesh communication or time lookup.

#### Scenario: Negative guidance present
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains text instructing the model to respond with plain text (no tool calls) for philosophical, emotional, or general-knowledge questions

### Requirement: System prompt instructs case-insensitive node name matching
The `MeshSystemPromptAddendum` SHALL instruct the model to treat node names case-insensitively. "dusty-b", "DUSTY-B", and "Dusty-B" SHALL all be recognized as valid targets for mesh_send.

#### Scenario: Case-insensitive instruction present
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains text instructing the model that node names are case-insensitive

### Requirement: System prompt addresses broadcast limitation
The `MeshSystemPromptAddendum` SHALL include guidance that there is no broadcast tool. When users ask to "let everyone know" or "tell all agents", the model SHALL explain the limitation and offer to send individual messages.

#### Scenario: Broadcast guidance present
- **WHEN** the MeshSystemPromptAddendum is rendered
- **THEN** it contains text explaining that no broadcast tool exists and the model should offer to send messages individually
