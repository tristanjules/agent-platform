## ADDED Requirements

### Requirement: User messages displayed immediately on submit
When the user submits a message, it SHALL appear in the conversation pane before the agent response begins streaming.

#### Scenario: User message visible before response
- **WHEN** user submits "What is presence?" via the input component
- **THEN** "You: What is presence?" appears in the conversation pane immediately

### Requirement: Streaming tokens append in real-time with cursor
Each `EventAgentTokens` event SHALL append the token to the current agent response line. A `▌` cursor SHALL be shown at the end of the streaming response until `EventAgentResponse` is received.

#### Scenario: Streaming token appended
- **WHEN** `EventAgentTokens` is received with "Being"
- **THEN** the conversation pane shows "DUSTY: Being▌"

#### Scenario: Cursor removed on completion
- **WHEN** `EventAgentResponse` is received with the full response text
- **THEN** the `▌` cursor is removed and the response is finalized in the history

### Requirement: EventAgentResponse reconciles streaming buffer
On `EventAgentResponse`, if the full response text is non-empty, it SHALL replace the streaming buffer as the finalized response (handling any dropped tokens from the EventBus).

#### Scenario: Full response used when available
- **WHEN** `EventAgentResponse` is received with full text "Being present is..."
- **THEN** the conversation entry shows exactly "Being present is..." regardless of what tokens arrived

### Requirement: Viewport auto-scrolls to bottom on new content
The conversation pane SHALL scroll to the bottom whenever new content is added (user message, token, finalized response, or system message).

#### Scenario: Auto-scroll on new token
- **WHEN** user has scrolled up and a new token arrives
- **THEN** the viewport scrolls to the bottom to show the latest content

### Requirement: System messages displayed distinctly
Command results and informational messages (e.g. "[memory cleared]") SHALL be displayed in the dimmed style, visually distinct from user/agent turns.

#### Scenario: System message styled differently
- **WHEN** a system message "[memory cleared]" is added
- **THEN** it renders in the Dimmed style, not the Primary or Accent style
