## ADDED Requirements

### Requirement: Built-in personas provide named system prompts
The system SHALL ship with three built-in personas: `philosopher`, `companion`, and `minimal`. Each persona SHALL define a `Name`, `Description`, and `SystemPrompt` string. `GetPersona(name)` SHALL return the named persona, falling back to `philosopher` for unknown names.

#### Scenario: Known persona lookup
- **WHEN** `GetPersona("companion")` is called
- **THEN** the returned `Persona` has a non-empty `SystemPrompt` and `Name == "Companion"`

#### Scenario: Unknown persona falls back to philosopher
- **WHEN** `GetPersona("nonexistent")` is called
- **THEN** the philosopher persona is returned (no error, no panic)

### Requirement: Persona list is discoverable
`ListPersonas()` SHALL return the names of all registered personas.

#### Scenario: List includes all built-ins
- **WHEN** `ListPersonas()` is called
- **THEN** the returned slice contains at least `"philosopher"`, `"companion"`, and `"minimal"`

### Requirement: System prompt is injected as the first message in every LLM call
The `Agent` SHALL prepend `schema.SystemMessage(persona.SystemPrompt)` as the first element of the message slice passed to the LLM on every `Chat()` call. History messages follow.

#### Scenario: System message is first
- **WHEN** `agent.buildMessages()` is called with a non-empty history
- **THEN** the first message in the returned slice has `Role == "system"` and content matching the active persona's system prompt
