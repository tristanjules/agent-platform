## ADDED Requirements

### Requirement: Handle returns structured CommandResult
`Handle(input string) CommandResult` SHALL process the slash command and return a `CommandResult` with `Output` (display text), `Quit` (exit signal), and `SettingsChanged` (state mutation flag). It SHALL NOT print to stdout.

#### Scenario: /clear returns structured result
- **WHEN** Handle("/clear") is called
- **THEN** returns CommandResult{Output: "[memory cleared]", SettingsChanged: true}

#### Scenario: /quit sets Quit flag
- **WHEN** Handle("/quit") is called
- **THEN** returns CommandResult{Quit: true}

### Requirement: All REPL commands supported
The handler SHALL support: `/clear`, `/persona`, `/model`, `/mode`, `/status`, `/voice`, `/help`, `/quit` (and aliases `/exit`, `/q`).

#### Scenario: /status returns current state
- **WHEN** Handle("/status") is called
- **THEN** returns CommandResult with Output containing state, memory count, model, and persona

#### Scenario: /persona sets persona
- **WHEN** Handle("/persona philosopher") is called
- **THEN** agent persona is updated, cfg.Agent.Personality set, and CommandResult{SettingsChanged: true} returned

### Requirement: Unknown command returns error message
An unrecognized slash command SHALL return a CommandResult with a descriptive Output and no side effects.

#### Scenario: Unknown command handled gracefully
- **WHEN** Handle("/foobar") is called
- **THEN** returns CommandResult{Output: "Unknown command: /foobar (try /help)"}

### Requirement: Voice toggle via VoiceController interface
`/voice start` and `/voice stop` SHALL operate through the `VoiceController` interface, not directly on any concrete voice struct.

#### Scenario: Voice toggled via interface
- **WHEN** Handle("/voice start") is called
- **THEN** VoiceController.ToggleVoice() is called
