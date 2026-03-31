## ADDED Requirements

### Requirement: State transitions displayed as transition lines
On `EventStateChanged`, the reasoning pane SHALL append a line formatted as `[<From> → <To>]` in the Dimmed style.

#### Scenario: State transition line appended
- **WHEN** `EventStateChanged` is received with transition Idle → Thinking
- **THEN** the reasoning pane shows "[Idle → Thinking]"

### Requirement: Streaming tokens echoed in dimmed style
On `EventAgentTokens`, the reasoning pane SHALL append/extend the current line with the token content rendered in the Dimmed style.

#### Scenario: Token echo appended
- **WHEN** `EventAgentTokens` is received with "Being"
- **THEN** the reasoning pane appends "Being" in the Dimmed style

### Requirement: Voice pipeline events annotated
`EventSTTResult`, `EventTTSStarted`, and `EventTTSDone` SHALL each append a bracketed annotation line (e.g. `[STT: transcribed text]`, `[TTS: synthesizing...]`, `[TTS: done]`) in the Dimmed style.

#### Scenario: STT result shown
- **WHEN** `EventSTTResult` is received with text "hey dusty"
- **THEN** reasoning pane shows "[STT: hey dusty]"

### Requirement: Line history capped at 200 entries
The reasoning pane SHALL keep at most 200 lines, dropping the oldest when the limit is exceeded.

#### Scenario: Old lines dropped at capacity
- **WHEN** the reasoning pane contains 200 lines and a new line is added
- **THEN** the oldest line is removed and the new line is appended

### Requirement: Viewport auto-scrolls to bottom on new content
The reasoning pane SHALL scroll to the bottom whenever a new line is appended.

#### Scenario: Auto-scroll on new event
- **WHEN** a new event line is appended to the reasoning pane
- **THEN** the viewport scrolls to show the latest line
