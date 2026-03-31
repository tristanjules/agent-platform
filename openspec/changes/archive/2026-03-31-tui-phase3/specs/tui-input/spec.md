## ADDED Requirements

### Requirement: Enter key submits non-empty input
When the user presses Enter with non-empty text in the input field, the component SHALL emit a `SubmitMsg{Text: value}` and clear the field.

#### Scenario: Enter submits and clears field
- **WHEN** input field contains "hello" and user presses Enter
- **THEN** a SubmitMsg with Text="hello" is emitted and the input field is empty

#### Scenario: Empty Enter is ignored
- **WHEN** input field is empty and user presses Enter
- **THEN** no SubmitMsg is emitted

### Requirement: Slash prefix routes to command handler
Input beginning with `/` SHALL be submitted as a command and handled by the CommandHandler, not sent to the agent.

#### Scenario: Slash command detected
- **WHEN** user submits "/clear"
- **THEN** the CommandHandler receives "/clear" and the agent is NOT called

### Requirement: Input width adapts to terminal width
The input component SHALL use the full terminal width minus a small margin for prompt and padding.

#### Scenario: Width updates on resize
- **WHEN** a WindowSizeMsg is received with width=120
- **THEN** the input field width is updated accordingly
