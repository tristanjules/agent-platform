# conversational-agent Specification

## Purpose
TBD - created by archiving change phase-1-foundations. Update Purpose after archive.
## Requirements
### Requirement: Agent exposes streaming Chat interface
The agent SHALL accept a user message string and return a channel of string tokens that streams the LLM response incrementally. The channel SHALL be closed when the response is complete.

#### Scenario: Successful streaming chat
- **WHEN** `agent.Chat(ctx, "hello")` is called while the agent is in Idle state
- **THEN** a `<-chan string` is returned with no error, tokens arrive incrementally, and the channel is closed when the response is finished

#### Scenario: Chat while not idle
- **WHEN** `agent.Chat(ctx, message)` is called while the agent is in Thinking state
- **THEN** an error is returned and no channel is produced

### Requirement: Agent wires memory, persona, router, and state machine
The agent SHALL compose `ConversationMemory`, `Persona`, `ModelRouter`, and `StateMachine` into a single coherent unit. Each `Chat()` call SHALL record the user message, query the LLM with the full history, record the assistant response, and publish lifecycle events to the `EventBus`.

#### Scenario: Memory recorded after chat
- **WHEN** a chat completes successfully
- **THEN** both the user message and full assistant response are present in `agent.MemoryLen()` (incremented by 2)

#### Scenario: Events published during chat
- **WHEN** a chat call begins
- **THEN** `EventUserMessage` is published before inference starts, `EventAgentTokens` is published for each token, and `EventAgentResponse` is published when the full response is assembled

### Requirement: Agent saves memory on Close
The agent SHALL persist conversation history to disk when `agent.Close()` is called, returning an error if the write fails.

#### Scenario: Memory persisted on close
- **WHEN** `agent.Close()` is called after a conversation
- **THEN** the memory JSON file is written (or updated) at the configured path and the file contains all conversation turns

#### Scenario: Close with empty memory
- **WHEN** `agent.Close()` is called with no conversation history
- **THEN** an empty JSON array is written and no error is returned

### Requirement: Agent persona and routing are hot-swappable at runtime
The agent SHALL allow `SetPersona(name)` and `SetRoutingPreference(pref)` to be called at any time, taking effect on the next `Chat()` call without restart.

#### Scenario: Persona switch takes effect immediately
- **WHEN** `agent.SetPersona("minimal")` is called followed by `agent.Chat(ctx, message)`
- **THEN** the system prompt used for inference reflects the minimal persona

#### Scenario: Routing preference switch
- **WHEN** `agent.SetRoutingPreference(PreferCloud)` is called
- **THEN** the next `Chat()` call routes to the cloud provider

