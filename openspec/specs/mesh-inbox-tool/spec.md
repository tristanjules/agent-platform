# mesh-inbox-tool Specification

## Purpose
Agent tool for querying the mesh message inbox, peer facts, and interaction history. Enables the conversational agent to review received messages and recall peer context.

## Requirements
### Requirement: mesh_inbox tool implements the Tool interface
The `mesh_inbox` tool SHALL implement the existing `Tool` interface (`Name()`, `Description()`, `Execute(args)`) and be registered in the agent's tool set alongside `mesh_send`.

#### Scenario: Tool is discoverable
- **WHEN** the agent lists available tools
- **THEN** `mesh_inbox` appears with its name and description

### Requirement: mesh_inbox supports an action argument
`Execute(args)` SHALL accept an `action` argument with the following values: `unread` (list unread messages), `history` (messages from a peer), `replay` (re-display a specific message by ID), `peer_facts` (facts about a peer), `peers` (list known peers with status and summaries).

#### Scenario: List unread messages
- **WHEN** `Execute({"action": "unread"})` is called with 3 unread messages
- **THEN** a formatted summary of the 3 unread messages is returned (sender, time, preview)

#### Scenario: Query message history for a peer
- **WHEN** `Execute({"action": "history", "peer": "DUSTY-B"})` is called
- **THEN** recent messages to/from DUSTY-B are returned in chronological order

#### Scenario: Replay a specific message
- **WHEN** `Execute({"action": "replay", "id": "abc123"})` is called
- **THEN** the full message content is returned and the message status is set to `read`

#### Scenario: Query peer facts
- **WHEN** `Execute({"action": "peer_facts", "peer": "DUSTY-B"})` is called
- **THEN** all stored facts about DUSTY-B are returned (owner, capabilities, custom notes)

#### Scenario: List all known peers
- **WHEN** `Execute({"action": "peers"})` is called
- **THEN** a summary of all known peers is returned (name, owner, status, last seen, fact count)

### Requirement: mesh_inbox returns human-readable formatted output
All responses from `mesh_inbox` SHALL be formatted as human-readable text suitable for the agent to present conversationally. Timestamps SHALL be relative ("5 minutes ago", "2 hours ago") when under 24 hours, absolute otherwise.

#### Scenario: Unread summary format
- **WHEN** `unread` action returns results
- **THEN** each message is formatted as: `[<time>] <sender>: <preview>` with a header showing total unread count

### Requirement: mesh_inbox handles missing/unknown arguments gracefully
If `action` is missing, the tool SHALL default to `unread`. If `peer` is required but missing, the tool SHALL return a helpful error. If the peer name is not found, the tool SHALL suggest known peers.

#### Scenario: Missing action defaults to unread
- **WHEN** `Execute({})` is called
- **THEN** the unread message list is returned

#### Scenario: Unknown peer suggests alternatives
- **WHEN** `Execute({"action": "peer_facts", "peer": "DUSTY-Z"})` is called and "DUSTY-Z" is not known
- **THEN** an error is returned: "Unknown peer 'DUSTY-Z'. Known peers: DUSTY-B, DUSTY-C"
