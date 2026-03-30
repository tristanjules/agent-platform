# mesh-discovery Specification

## Purpose
Peer discovery, heartbeat, and registry management for the LoRa mesh network. Handles node handshakes, periodic presence announcements, and peer lifecycle tracking.

## Requirements
### Requirement: Nodes perform discovery handshake on startup
On startup, the mesh subsystem SHALL broadcast a discovery message (`dis` type) containing the node's ID, agent name, and owner name. It SHALL listen for discovery responses for a configurable timeout (default: 30 seconds).

#### Scenario: Discovery finds a peer
- **WHEN** node A broadcasts a discovery message and node B responds with its own discovery message
- **THEN** both nodes add each other to their peer registries

#### Scenario: Discovery finds no peers
- **WHEN** a discovery message is broadcast and no response is received within the timeout
- **THEN** the peer registry remains empty and the node operates in standalone mode

### Requirement: Nodes send periodic heartbeats
The mesh subsystem SHALL send a heartbeat message (`hb` type) at a configurable interval (default: every 5 minutes) to all known peers. The heartbeat SHALL include the node's current state and optional location.

#### Scenario: Heartbeat updates peer last-seen time
- **WHEN** a heartbeat from node B is received by node A
- **THEN** node A's peer registry updates node B's `lastSeen` timestamp

### Requirement: Peer registry persists known nodes
The peer registry SHALL store known peers in a JSON file (`mesh-peers.json`). Each entry SHALL contain: node ID, agent name, owner name, capabilities, last-seen timestamp, and last-known location.

#### Scenario: Peer data persists across restarts
- **WHEN** the agent restarts after previously discovering node B
- **THEN** node B is present in the loaded peer registry with its stored metadata

#### Scenario: New peer is added to registry
- **WHEN** a discovery message is received from a previously unknown node
- **THEN** a new entry is created in the peer registry and saved to disk

### Requirement: Stale peers are marked as lost
If a peer's `lastSeen` timestamp exceeds a configurable TTL (default: 30 minutes), the peer SHALL be marked as `lost` and an `EventMeshNodeLost` event SHALL be published.

#### Scenario: Peer goes stale
- **WHEN** node B's last heartbeat was 31 minutes ago with a 30-minute TTL
- **THEN** node B is marked as `lost` in the registry and `EventMeshNodeLost` is published

### Requirement: Peer registry stores connectivity metadata only
Each peer entry SHALL contain: node ID, agent name, owner name, capabilities list, last-seen timestamp, last-known location, and status (active/lost). Rich facts and interaction history are stored separately in the peer memory store.

#### Scenario: Discovery populates registry entry
- **WHEN** node B's discovery message includes `owner: "Lourens"` and `agent: "DUSTY-B"`
- **THEN** the peer registry entry for node B contains `Owner: "Lourens"` and `AgentName: "DUSTY-B"`

### Requirement: Registry exposes peer lookup methods
The registry SHALL provide `GetPeer(nodeID)`, `ListPeers()`, `ListActivePeers()` (non-lost only), and `GetPeerByName(agentName)` methods.

#### Scenario: List active peers excludes lost nodes
- **WHEN** `ListActivePeers()` is called with one active and one lost peer
- **THEN** only the active peer is returned
