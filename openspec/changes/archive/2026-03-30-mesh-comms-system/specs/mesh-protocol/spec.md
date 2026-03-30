## ADDED Requirements

### Requirement: Message schema uses compact JSON with terse keys
All mesh messages SHALL use a JSON schema with single-character or abbreviated keys to fit within the 200-byte payload budget. The schema SHALL include at minimum: message type (`t`), action (`a`), value (`v`), sender node ID (`s`), and message ID (`id`).

#### Scenario: Encode a command message
- **WHEN** a command message is encoded with type "cmd", action "nav", value "grid-4-7"
- **THEN** the JSON output is `{"t":"cmd","a":"nav","v":"grid-4-7","s":"<nodeID>","id":"<msgID>"}` and is under 200 bytes

#### Scenario: Decode a received message
- **WHEN** a valid JSON payload `{"t":"ack","id":"abc123","s":"node2"}` is received
- **THEN** the decoded message has Type "ack", ID "abc123", and Sender "node2"

### Requirement: Protocol defines message types
The protocol SHALL support the following message types: `cmd` (command), `ack` (acknowledgement), `hb` (heartbeat/presence), `loc` (location), `msg` (text message), `syn` (state sync), and `dis` (discovery handshake).

#### Scenario: Each message type is recognized
- **WHEN** a message with type `hb` is decoded
- **THEN** it is recognized as a heartbeat message and processed accordingly

#### Scenario: Unknown message type is rejected
- **WHEN** a message with type `xyz` is decoded
- **THEN** an error is returned indicating an unknown message type

### Requirement: Messages include a unique ID for deduplication
Every outbound message SHALL include a unique `id` field (8-character alphanumeric). The receiver SHALL track recently seen IDs (sliding window of 256) and discard duplicates.

#### Scenario: Duplicate message is dropped
- **WHEN** a message with the same `id` is received twice within the deduplication window
- **THEN** the second message is silently discarded

### Requirement: Text messages truncate LLM output to fit payload budget
When encoding a `msg` type message, the protocol SHALL truncate the value field to ensure the total encoded JSON is under 200 bytes. The truncation point SHALL be at a word boundary when possible.

#### Scenario: Long text is truncated
- **WHEN** a text message with a 300-character value is encoded
- **THEN** the value is truncated to fit within 200 bytes total, ending at a word boundary

### Requirement: Location messages encode coordinates compactly
Location messages (`loc` type) SHALL encode latitude and longitude as fixed-point integers (6 decimal places × 1,000,000) to save bytes compared to float strings.

#### Scenario: GPS coordinates are encoded as integers
- **WHEN** a location message is created with lat -31.416668 and lon 19.233334
- **THEN** the encoded payload contains `"la":-31416668,"lo":19233334`
