## ADDED Requirements

### Requirement: Peer memory store persists rich facts per peer
The peer memory store SHALL maintain a JSON file (`mesh-peer-memory.json`) with per-peer entries keyed by node ID. Each entry SHALL contain a list of facts (free-text strings with timestamps) and an interaction log (timestamped records of messages sent/received).

#### Scenario: Fact is stored for a peer
- **WHEN** `AddFact(nodeID, "Lourens is camped near Binnekring")` is called
- **THEN** the fact is appended to that peer's facts list with a timestamp and persisted to disk

#### Scenario: Facts persist across restarts
- **WHEN** the agent restarts after storing facts about DUSTY-B
- **THEN** all previously stored facts for DUSTY-B are loaded from `mesh-peer-memory.json`

### Requirement: Peer memory records interaction history
Every mesh message sent to or received from a peer SHALL be logged in that peer's interaction history. Each log entry SHALL contain: timestamp, direction (sent/received), message type, and a summary of the content (not the full payload, to keep the file manageable).

#### Scenario: Received message is logged
- **WHEN** a `msg` type message is received from node B with content "heading to Binnekring"
- **THEN** an interaction entry is added: `{time, direction: "received", type: "msg", summary: "heading to Binnekring"}`

#### Scenario: Sent message is logged
- **WHEN** a `msg` type message is sent to node B via `mesh_send`
- **THEN** an interaction entry is added: `{time, direction: "sent", type: "msg", summary: "<content>"}`

### Requirement: Peer memory supports fact queries
The store SHALL provide `GetFacts(nodeID)`, `GetInteractions(nodeID, limit)`, and `GetAllPeerSummaries()` methods. `GetAllPeerSummaries()` SHALL return a brief summary per peer (name, owner, fact count, last interaction time).

#### Scenario: Query facts for a specific peer
- **WHEN** `GetFacts(nodeB_ID)` is called and node B has 3 stored facts
- **THEN** all 3 facts are returned with their timestamps, ordered newest first

#### Scenario: Query interactions with a limit
- **WHEN** `GetInteractions(nodeB_ID, 5)` is called and node B has 20 interactions
- **THEN** the 5 most recent interactions are returned

### Requirement: Peer memory auto-seeds facts from discovery metadata
When a new peer is discovered, the peer memory store SHALL automatically create an entry and seed initial facts from the discovery message metadata (owner name, agent name, persona).

#### Scenario: Discovery seeds peer memory
- **WHEN** node B is discovered with owner "Lourens", agent "DUSTY-B", persona "Philosopher"
- **THEN** peer memory for node B is created with facts: "Owner: Lourens", "Agent name: DUSTY-B", "Persona: Philosopher"

### Requirement: Interaction history has a configurable retention limit
The interaction log per peer SHALL retain a configurable maximum number of entries (default: 100). When the limit is exceeded, the oldest entries SHALL be removed.

#### Scenario: Old interactions are pruned
- **WHEN** peer B has 100 interactions and a new one is logged with a limit of 100
- **THEN** the oldest interaction is removed and the new one is appended
