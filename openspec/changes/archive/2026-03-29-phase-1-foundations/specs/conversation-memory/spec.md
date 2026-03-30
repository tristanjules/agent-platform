## ADDED Requirements

### Requirement: Memory stores messages with role and timestamp
The `ConversationMemory` SHALL store messages as `{Role, Content, Timestamp}` triples. Valid roles are `"user"`, `"assistant"`, and `"system"`.

#### Scenario: Add and retrieve messages
- **WHEN** `memory.Add("user", "hello")` followed by `memory.Add("assistant", "hi")` are called
- **THEN** `memory.Messages()` returns a slice of length 2 with matching Role and Content fields

### Requirement: Memory enforces a configurable sliding window
When the number of stored messages exceeds `maxHistory`, the oldest messages SHALL be dropped so that `len(messages) == maxHistory`.

#### Scenario: Sliding window trims oldest messages
- **WHEN** 5 messages are added to a memory with `maxHistory=3`
- **THEN** `memory.Messages()` returns the 3 most recently added messages

#### Scenario: Zero maxHistory means unlimited
- **WHEN** `maxHistory=0` is configured and 100 messages are added
- **THEN** all 100 messages are retained

### Requirement: Memory persists to and reloads from a JSON file
When a `persistPath` is configured, `memory.Save()` SHALL write all messages as a JSON array to that file. `NewConversationMemory(max, path)` SHALL load existing messages from the file if it exists.

#### Scenario: Round-trip persistence
- **WHEN** messages are added, `Save()` is called, and a new `ConversationMemory` is created with the same path
- **THEN** the new instance contains the same messages with identical Role and Content

#### Scenario: Missing file is not an error
- **WHEN** `NewConversationMemory` is called with a path to a non-existent file
- **THEN** no error is returned and the memory starts empty

### Requirement: Memory can be cleared at runtime
`memory.Clear()` SHALL remove all stored messages immediately.

#### Scenario: Clear empties memory
- **WHEN** `memory.Clear()` is called after adding messages
- **THEN** `memory.Len()` returns 0

### Requirement: Memory operations are thread-safe
All public methods (`Add`, `Messages`, `Clear`, `Len`, `Save`) SHALL be safe to call concurrently from multiple goroutines.

#### Scenario: Concurrent add and read
- **WHEN** multiple goroutines call `Add()` and `Messages()` simultaneously
- **THEN** no data race is detected and the slice is always internally consistent
