## Why

DUSTY instances at AfrikaBurn will be offline (no cellular), running on separate Raspberry Pis in the open desert. They need a way to discover each other, exchange terse messages (commands, state, locations), and alert their operators to incoming transmissions — all over LoRa mesh radio via Meshtastic. This is a core capability for the playa deployment: two agents that can coordinate, share intent, and maintain awareness of each other's existence and ownership.

## What Changes

- **Mesh communication adapter** — A Go package wrapping Meshtastic serial/protobuf communication for sending and receiving terse agent-to-agent messages over LoRa radio, with message queuing to respect duty cycle limits.
- **Message schema** — A compact binary/JSON message format (<200 bytes) supporting commands, state sync, acknowledgements, presence heartbeats, and location payloads.
- **Node discovery & mesh registry** — Startup handshake protocol and a persistent peer registry that tracks known mesh nodes, their owners, capabilities, and last-seen status.
- **Peer memory store** — A separate persistent store for rich facts and interaction history per peer (e.g. "Lourens is camped at grid 4-7", "DUSTY-B has a solar charger"). Queryable by the agent for conversational context about mesh peers.
- **Message inbox/store** — Persistent storage for all received mesh messages with read/unread/dismissed status. Supports replay, unread counts, and retrieval of dismissed messages. Solves the "away" problem when messages arrive while the user isn't at the terminal.
- **Agent tool: `mesh_send`** — An agent-callable tool that lets DUSTY compose and send messages to other mesh nodes, available via the existing `Tool` interface.
- **Agent tool: `mesh_inbox`** — An agent-callable tool for querying message history, listing unread transmissions, replaying messages, and looking up peer facts conversationally.
- **Incoming transmission notification system** — On message receipt: play a notification sound, trigger a haptic vibration stub, and interrupt the TUI/REPL with a retro sci-fi "INCOMING TRANSMISSION" announcement, prompting the user to accept/dismiss. Unread messages are queued for review on return.
- **GPS location service (stub)** — A stubbed GPS interface for future integration, allowing nodes to share and query location data.
- **Haptic feedback service (stub)** — A stubbed vibration/haptic interface for future hardware integration.
- **New event types** — `EventMeshMessageReceived`, `EventMeshNodeDiscovered`, `EventMeshNodeLost`, `EventNotificationTriggered` added to the event bus.
- **New state machine states** — `StateReceivingTransmission` added to handle the incoming message interrupt flow.

## Capabilities

### New Capabilities
- `mesh-transport`: LoRa/Meshtastic serial adapter, message queuing, duty cycle management, send/receive loop
- `mesh-protocol`: Message schema, encoding/decoding, node addressing, encryption (PSK via Meshtastic)
- `mesh-discovery`: Node discovery handshake, peer registry, presence heartbeats
- `mesh-peer-memory`: Rich per-peer fact store and interaction history, separate from connectivity registry
- `mesh-message-store`: Persistent message inbox with read/unread/dismissed status, replay, and query support
- `mesh-send-tool`: Agent-callable tool for composing and sending mesh messages
- `mesh-inbox-tool`: Agent-callable tool for querying message history, unread messages, and peer facts
- `mesh-notifications`: Incoming transmission alerts — sound, haptic stub, TUI/REPL interrupt with accept/dismiss flow, unread queue on return
- `gps-service`: Stubbed GPS location interface for future hardware integration
- `haptic-service`: Stubbed haptic/vibration interface for future hardware integration

### Modified Capabilities
- `event-bus`: New event types for mesh communication and notifications
- `state-machine`: New `StateReceivingTransmission` state and transitions

## Impact

- **New packages**: `internal/mesh/`, `internal/notify/`, `internal/gps/`, `internal/haptic/`
- **Modified packages**: `internal/state/` (new events + states), `internal/agent/` (new tool registration), `internal/display/tui/` (transmission interrupt UI), `cmd/dusty/` (mesh subsystem init)
- **New dependencies**: Meshtastic protobuf definitions, serial port library (e.g. `go.bug.st/serial`)
- **Hardware**: Requires LILYGO T-Beam or compatible Meshtastic node connected via USB serial
- **Config**: New `[mesh]` section in TOML config for serial port, node ID, PSK, peer list
- **Data files**: `mesh-peers.json` (registry), `mesh-peer-memory.json` (facts/history), `mesh-messages.json` (inbox)
- **Assets**: Notification sound file(s) in `assets/sounds/`
