## Context

DUSTY is an offline-first AI agent running on Raspberry Pi, built in Go with an event-driven architecture (state machine + event bus). At AfrikaBurn, multiple DUSTY instances will operate in the open desert with no cellular connectivity. They need agent-to-agent communication over LoRa mesh radio using Meshtastic-compatible hardware (LILYGO T-Beam) connected via USB serial.

The existing architecture provides clean extension points: the `Tool` interface for agent-callable capabilities, the `EventBus` for decoupled pub/sub, and the `StateMachine` for managing interaction states. The communication system plugs into all three.

**Constraints:**
- LoRa throughput: ~1–3 kbps with duty cycle limits
- Message payloads must be <200 bytes
- Fire-and-forget or request/acknowledge patterns only (no streaming)
- Must work fully offline — no internet, no cloud services
- Hardware: LILYGO T-Beam connected to Pi via USB serial

## Goals / Non-Goals

**Goals:**
- Reliable agent-to-agent messaging over LoRa mesh via Meshtastic protocol
- Compact message schema for commands, state sync, presence, location, and acks
- Node discovery with persistent peer registry (who is who, last seen, location)
- Agent-callable `mesh_send` tool for composing/sending messages
- Theatrical "INCOMING TRANSMISSION" notification flow in TUI and REPL
- Stubbed GPS and haptic interfaces ready for future hardware integration
- AES-256 encrypted channels via Meshtastic PSK

**Non-Goals:**
- Streaming audio/video over mesh (bandwidth prohibitive)
- Direct Meshtastic protobuf implementation from scratch (use serial CLI or protobuf over serial)
- BLE transport in Go (iOS uses Meshtastic app natively via BLE)
- Multi-hop routing logic (Meshtastic handles this at firmware level)
- Full GPS hardware integration (stubbed for now)
- Real haptic motor driver (stubbed for now)

## Decisions

### D1: Go serial adapter over Python subprocess bridge

**Decision:** Communicate with the Meshtastic node directly via serial protobuf over USB, using a pure Go serial library (`go.bug.st/serial`) and Meshtastic's protobuf definitions compiled for Go.

**Alternatives considered:**
- **Python subprocess bridge** — Shell out to the `meshtastic` Python CLI. Simpler initially, but adds Python as a runtime dependency on the Pi, introduces IPC complexity, and is harder to manage lifecycle/errors from Go.
- **gRPC/socket bridge to Python daemon** — More robust than subprocess, but still requires Python runtime and adds operational complexity for a desert deployment.

**Rationale:** Pure Go keeps the single-binary deployment story. Meshtastic nodes expose a well-documented serial protobuf API (`meshtastic-protobuf`). The Go serial library is mature and cross-compiles to ARM64. This eliminates the Python dependency entirely.

### D2: Terse JSON message encoding (not binary protobuf)

**Decision:** Use compact JSON for the application-layer message schema (the payload inside Meshtastic packets).

**Alternatives considered:**
- **Custom protobuf schema** — Maximum compression, but adds protobuf compilation step and makes debugging harder on resource-constrained devices.
- **MessagePack/CBOR** — Good compression, but adds dependency and reduces debuggability.

**Rationale:** JSON is human-readable (critical for debugging in the desert with limited tooling), supported natively in Go, and with terse keys (`"t"`, `"a"`, `"v"`) fits within the 200-byte budget. The overhead vs binary is ~20-30 bytes — acceptable given message sparsity.

### D3: Event bus integration for incoming messages

**Decision:** The mesh receive loop publishes `EventMeshMessageReceived` on the existing `EventBus`. The TUI/REPL subscribes to this event to trigger the notification flow. No direct coupling between mesh and display.

**Rationale:** Consistent with the existing architecture. The event bus already bridges audio, agent, and display subsystems. Adding mesh events follows the same pattern and allows any future subscriber (logging, analytics, visual display) to react without modification.

### D4: State machine interrupt for incoming transmissions

**Decision:** Add `StateReceivingTransmission` to the state machine. On incoming message, transition to this state (valid from `StateIdle` and `StateProcessing`). The TUI/REPL renders the theatrical announcement. User accept → display message → return to previous state. User dismiss → return to previous state.

**Rationale:** Using the state machine ensures only one interrupt can happen at a time and prevents conflicting states (e.g., can't receive transmission while already processing one). The accept/dismiss flow maps naturally to a state transition.

### D5: Peer registry as JSON file (not database)

**Decision:** Store known mesh peers in a JSON file (`mesh-peers.json`) alongside the existing `dusty.memory.json`.

**Alternatives considered:**
- **SQLite** — Overkill for <10 peers, adds CGo dependency.
- **In-memory only** — Loses peer knowledge across restarts.

**Rationale:** Consistent with existing memory persistence pattern. Peer count will be small (2-10 nodes). JSON is human-editable for manual provisioning.

### D6: Notification sound via audio playback subsystem

**Decision:** Reuse the existing `internal/audio/playback.go` infrastructure to play notification sounds. Fall back to terminal bell (`\a`) if audio playback is unavailable.

**Rationale:** Audio playback is already scaffolded for TTS. Notification sounds are just short PCM samples played through the same pipeline. Terminal bell provides a degraded but functional fallback for headless/REPL mode.

### D7: Separate peer memory store from connectivity registry

**Decision:** Peer facts and interaction history live in a separate `mesh-peer-memory.json` file, not in the peer registry. The registry (`mesh-peers.json`) stays lean — connectivity metadata only (nodeID, name, owner, lastSeen, location, status). The peer memory store holds rich, agent-writable facts and a per-peer interaction log.

**Alternatives considered:**
- **Extend registry with facts/history arrays** — Simple co-location, but the registry becomes overloaded with two concerns (connectivity tracking vs knowledge). Different mutation patterns (registry mutates on every heartbeat; facts mutate rarely).
- **Inject into ConversationMemory as system messages** — Agent can naturally reference peer facts, but pollutes the sliding window with mesh metadata. Facts would be evicted as conversation grows.

**Rationale:** Clean separation of concerns. The registry is a phone book that changes frequently (heartbeats every 5 min). Peer memory is a knowledge base that changes rarely (new facts from messages or agent inference). Different persistence cadences, different consumers. The agent queries peer memory via the `mesh_inbox` tool; the mesh subsystem queries the registry for routing.

### D8: Persistent message inbox with status tracking

**Decision:** All received mesh messages are persisted to `mesh-messages.json` with a status field (`unread`, `read`, `dismissed`). Messages are never deleted — only status changes. The inbox supports query by status, sender, and time range.

**Alternatives considered:**
- **In-memory only, messages lost on restart** — Unacceptable for a desert deployment where the user may be away for hours.
- **Store only in ConversationMemory** — Dismissed messages would be lost. No way to query "what did I miss?" without replaying the whole conversation.

**Rationale:** The "away" problem is real — DUSTY will be running unattended for long stretches. When the user returns, they need to see what arrived. Dismissed messages should be retrievable (you might dismiss in haste, want it later). The status model lets the TUI show an unread count and the agent can summarise what's pending.

### D9: `mesh_inbox` agent tool for conversational message/peer queries

**Decision:** A second agent-callable tool (`mesh_inbox`) lets DUSTY query the message store and peer memory conversationally. Supports actions: `unread` (list unread messages), `history` (messages from a peer), `replay` (re-display a specific message), `peer_facts` (facts about a peer), and `peers` (list known peers with status).

**Rationale:** The `mesh_send` tool handles outbound. The `mesh_inbox` tool handles inbound queries. This lets the user ask DUSTY naturally: "What did DUSTY-B send while I was gone?" or "What do you know about Lourens?" without needing slash commands.

### D10: Package structure

```
internal/mesh/
├── transport.go      # Serial adapter, send/receive loop, queuing
├── protocol.go       # Message schema, encode/decode
├── discovery.go      # Handshake, heartbeat, peer registry
├── registry.go       # Peer connectivity registry (lean: nodeID, name, lastSeen)
├── peer_memory.go    # Rich per-peer fact store and interaction history
├── message_store.go  # Persistent message inbox with status tracking
├── tool_send.go      # mesh_send Tool implementation
└── tool_inbox.go     # mesh_inbox Tool implementation

internal/notify/
├── notifier.go       # Notification orchestrator (sound + haptic + event)
└── sound.go          # Sound playback integration

internal/gps/
└── gps.go            # Stubbed GPS interface

internal/haptic/
└── haptic.go         # Stubbed haptic interface
```

Data files (alongside existing `dusty.memory.json`):
```
mesh-peers.json          # Connectivity registry (lean, frequently updated)
mesh-peer-memory.json    # Per-peer facts and interaction log (rarely updated)
mesh-messages.json       # Message inbox with status tracking
```

## Risks / Trade-offs

**[Risk] Serial protobuf API stability** — Meshtastic's serial API may change between firmware versions.
→ Mitigation: Pin to a known Meshtastic firmware version. The protobuf schema is versioned. Document the tested firmware version in config.

**[Risk] Duty cycle violations** — Sending too frequently can violate LoRa regulations and degrade mesh performance.
→ Mitigation: Message queue with configurable rate limiter (default: 1 message per 30 seconds). Queue drops oldest messages if backpressure builds.

**[Risk] Serial port reliability** — USB serial connections can drop, especially with vibration/heat in desert conditions.
→ Mitigation: Reconnection loop with exponential backoff. State machine transitions to `StateError` on serial failure. Health check via periodic ping.

**[Risk] Message size overflow** — LLM-generated content may exceed 200-byte budget.
→ Mitigation: The `mesh_send` tool truncates/summarises content before transmission. Agent system prompt instructs terse message composition.

**[Risk] Peer registry stale data** — Nodes may go offline without sending a goodbye.
→ Mitigation: TTL-based expiry on peer entries (configurable, default 30 minutes). Heartbeat absence triggers `EventMeshNodeLost`.

**[Trade-off] JSON vs binary encoding** — ~20-30 bytes overhead per message for human readability. Acceptable given message sparsity (minutes between messages, not seconds).

**[Trade-off] No BLE transport in Go** — iOS users must use the Meshtastic app for monitoring. This is acceptable since the primary interface is the Pi's TUI/REPL.
