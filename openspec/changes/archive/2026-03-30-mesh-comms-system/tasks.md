## 1. Foundation — Event Bus & State Machine Extensions

- [x] 1.1 Add new event types to `internal/state/events.go`: `EventMeshMessageReceived`, `EventMeshNodeDiscovered`, `EventMeshNodeLost`, `EventNotificationTriggered`
- [x] 1.2 Add `StateReceivingTransmission` to `internal/state/machine.go` with valid transitions from `StateIdle` and `StateProcessing`, and return transitions back to the originating state
- [x] 1.3 Write tests for new state transitions (valid and invalid) and new event type subscriptions

## 2. Stub Services — GPS & Haptic

- [x] 2.1 Create `internal/gps/gps.go` with `GPSProvider` interface and stub implementation returning configurable fixed location
- [x] 2.2 Create `internal/haptic/haptic.go` with `HapticProvider` interface, pattern constants (`PatternShort`, `PatternDouble`, `PatternLong`), and stub implementation that logs requests
- [x] 2.3 Write tests for GPS stub (configured vs unconfigured) and haptic stub (known and unknown patterns)

## 3. Mesh Protocol — Message Schema

- [x] 3.1 Create `internal/mesh/protocol.go` with message types enum (`cmd`, `ack`, `hb`, `loc`, `msg`, `syn`, `dis`), terse JSON message struct, and encode/decode functions
- [x] 3.2 Implement message ID generation (8-char alphanumeric) and deduplication tracker (sliding window of 256 IDs)
- [x] 3.3 Implement compact location encoding (fixed-point integer lat/lon)
- [x] 3.4 Implement payload size validation and word-boundary truncation for text messages
- [x] 3.5 Write tests for encode/decode round-trips, deduplication, size limits, location encoding, and unknown type rejection

## 4. Mesh Transport — Serial Adapter

- [x] 4.1 Add `go.bug.st/serial` dependency and Meshtastic protobuf definitions to the Go module
- [x] 4.2 Create `internal/mesh/transport.go` with `Transport` struct: `Open(portPath)`, `Close()`, `Send(nodeID, payload)`, `Incoming() <-chan Message`
- [x] 4.3 Implement serial protobuf read loop (decode `FromRadio` packets, deliver to incoming channel, drop malformed packets)
- [x] 4.4 Implement serial protobuf write (encode `ToRadio` packets)
- [x] 4.5 Implement outbound message queue with configurable rate limiter (default 1 msg/30s) and overflow drop-oldest policy (capacity 16)
- [x] 4.6 Implement reconnection with exponential backoff (1s start, 60s max) on serial failure
- [x] 4.7 Implement graceful shutdown: drain queue (5s timeout), close serial, close channels
- [x] 4.8 Write tests for queue rate limiting, overflow behavior, and graceful shutdown (use mock serial)

## 5. Mesh Discovery — Peer Registry

- [x] 5.1 Create `internal/mesh/registry.go` with `PeerRegistry` struct and lean peer entry type (nodeID, agentName, ownerName, capabilities, lastSeen, location, status — no facts/history)
- [x] 5.2 Implement JSON persistence (`mesh-peers.json`): load on init, save on mutation
- [x] 5.3 Implement lookup methods: `GetPeer(nodeID)`, `ListPeers()`, `ListActivePeers()`, `GetPeerByName(agentName)`
- [x] 5.4 Implement TTL-based stale peer detection (default 30 min) with `EventMeshNodeLost` publishing
- [x] 5.5 Create `internal/mesh/discovery.go` with startup discovery handshake (broadcast `dis`, listen for responses, configurable timeout 30s)
- [x] 5.6 Implement periodic heartbeat sender (default every 5 min) with current state and optional location
- [x] 5.7 Implement heartbeat receiver that updates peer `lastSeen` and publishes `EventMeshNodeDiscovered` for new peers
- [x] 5.8 Write tests for registry CRUD, persistence round-trip, stale detection, and discovery handshake flow

## 6. Peer Memory Store

- [x] 6.1 Create `internal/mesh/peer_memory.go` with `PeerMemory` struct: per-peer facts list (text + timestamp) and interaction log (timestamp, direction, type, summary)
- [x] 6.2 Implement JSON persistence (`mesh-peer-memory.json`): load on init, save on mutation
- [x] 6.3 Implement `AddFact(nodeID, fact)`, `GetFacts(nodeID)`, `GetInteractions(nodeID, limit)`, `GetAllPeerSummaries()`
- [x] 6.4 Implement auto-seeding of facts from discovery metadata (owner, agent name, persona) on `EventMeshNodeDiscovered`
- [x] 6.5 Implement interaction logging: hook into message send/receive to record entries with direction and summary
- [x] 6.6 Implement configurable interaction history retention limit (default 100 per peer) with oldest-first pruning
- [x] 6.7 Write tests for fact CRUD, interaction logging, auto-seeding, retention pruning, and persistence round-trip

## 7. Message Store — Inbox

- [x] 7.1 Create `internal/mesh/message_store.go` with `MessageStore` struct: message entries with ID, sender, type, content, receivedAt, status (unread/read/dismissed/sent), readAt
- [x] 7.2 Implement JSON persistence (`mesh-messages.json`): load on init, save on mutation
- [x] 7.3 Implement query methods: `ListUnread()`, `ListAll(limit)`, `ListFromPeer(nodeID, limit)`, `GetMessage(id)`, `UnreadCount()`
- [x] 7.4 Implement status transitions: `MarkRead(id)`, `MarkDismissed(id)` with valid transition enforcement (unread→read, unread→dismissed, dismissed→read)
- [x] 7.5 Implement outbound message storage: store sent messages with status `sent` and sentAt timestamp
- [x] 7.6 Implement configurable retention limit (default 200): prune oldest read first, then dismissed, never prune unread
- [x] 7.7 Wire message store into receive loop (auto-store on `EventMeshMessageReceived`) and send path (store on `mesh_send` enqueue)
- [x] 7.8 Write tests for status transitions, query methods, retention pruning, and persistence round-trip

## 8. Mesh Send Tool

- [x] 8.1 Create `internal/mesh/tool_send.go` implementing the `Tool` interface: `Name()` → "mesh_send", `Description()`, `Execute(args)`
- [x] 8.2 Implement target resolution (agent name → node ID via peer registry, or direct node ID)
- [x] 8.3 Implement payload size check — return error with character budget if message too long
- [x] 8.4 Log sent message to peer memory interaction history and message store
- [x] 8.5 Register `mesh_send` tool in `internal/agent/tools.go` `DefaultTools()` (conditional on mesh being enabled)
- [x] 8.6 Write tests for target resolution, size validation, successful enqueue, and unknown target error

## 9. Mesh Inbox Tool

- [x] 9.1 Create `internal/mesh/tool_inbox.go` implementing the `Tool` interface: `Name()` → "mesh_inbox", `Description()`, `Execute(args)`
- [x] 9.2 Implement `unread` action: query message store, format summary with relative timestamps
- [x] 9.3 Implement `history` action: query message store by peer (resolve name via registry), return chronological thread
- [x] 9.4 Implement `replay` action: retrieve message by ID, mark as read, return full content
- [x] 9.5 Implement `peer_facts` action: query peer memory for facts about a specific peer
- [x] 9.6 Implement `peers` action: combine registry (status, lastSeen) with peer memory (fact count, last interaction) for a rich peer summary
- [x] 9.7 Implement graceful error handling: default action to `unread`, suggest known peers on unknown peer name
- [x] 9.8 Register `mesh_inbox` tool in `internal/agent/tools.go` alongside `mesh_send`
- [x] 9.9 Write tests for all actions, default behavior, and unknown peer suggestions

## 10. Notification System

- [x] 10.1 Create `internal/notify/notifier.go` with `Notifier` struct that orchestrates sound + haptic + event publishing on incoming mesh messages
- [x] 10.2 Create `internal/notify/sound.go` with sound playback via existing audio subsystem, falling back to terminal bell (`\a`)
- [x] 10.3 Add `assets/sounds/transmission.wav` notification sound file
- [x] 10.4 Subscribe `Notifier` to `EventMeshMessageReceived` on the event bus and trigger notification sequence
- [x] 10.5 Implement idle detection (no user input for configurable period, default 5 min) — suppress overlay when idle, store messages as unread silently
- [x] 10.6 Write tests for notification orchestration (mock sound + haptic providers) and idle suppression logic

## 11. TUI Integration — Transmission Overlay & Unread Queue

- [x] 11.1 Create transmission overlay component in `internal/display/tui/` — full-width retro sci-fi ">>> INCOMING TRANSMISSION <<<" banner with sender info and accept/dismiss keybindings
- [x] 11.2 Wire `EventMeshMessageReceived` into the TUI event bridge (`bridge.go`) to trigger the overlay
- [x] 11.3 Implement accept flow: display message in conversation pane, mark as read in message store, trigger TTS if available, return to previous state
- [x] 11.4 Implement dismiss flow: close overlay, mark as dismissed in message store, return to previous state
- [x] 11.5 Implement unread banner on activity resume: "⚡ N unread transmissions from <senders> [R] Review [C] Continue"
- [x] 11.6 Implement review flow: present unread messages one at a time chronologically with accept/dismiss per message
- [x] 11.7 Add subtle system messages for `EventMeshNodeDiscovered` ("entered mesh range") and `EventMeshNodeLost` ("signal lost") in conversation pane
- [x] 11.8 Write tests for overlay rendering, accept/dismiss state transitions, and unread banner flow

## 12. REPL Integration

- [x] 12.1 Add inline transmission notification to REPL loop in `cmd/dusty/main.go` — print ">>> INCOMING TRANSMISSION <<<" banner and `[y/n]` prompt
- [x] 12.2 Implement accept (print message, mark read) and dismiss (mark dismissed) flows in REPL mode
- [x] 12.3 Implement unread summary on activity resume in REPL mode

## 13. Configuration & Initialization

- [x] 13.1 Add `[mesh]` section to config schema in `internal/config/config.go`: enabled, serial port, baud rate, node ID, PSK, heartbeat interval, discovery timeout, peer TTL, queue size, rate limit, idle timeout, message retention limit, interaction history limit
- [x] 13.2 Add mesh config to `configs/` TOML template file with documented defaults
- [x] 13.3 Wire mesh subsystem initialization in `cmd/dusty/main.go`: create transport, registry, peer memory, message store, discovery, notifier; start receive loop; register mesh_send and mesh_inbox tools (gated on `mesh.enabled`)
- [x] 13.4 Inject GPS stub and haptic stub into mesh and notification subsystems via config

## 14. Integration Testing & Documentation

- [x] 14.1 Write integration test: full send/receive cycle using mock serial (encode → queue → send → receive → decode → store → notify)
- [x] 14.2 Write integration test: discovery handshake between two mock nodes with peer memory auto-seeding
- [x] 14.3 Write integration test: message lifecycle (receive → unread → dismiss → replay via inbox → read)
- [x] 14.4 Write integration test: idle period → messages queued → activity resume → unread banner
- [x] 14.5 Add mesh system prompt additions to persona system — instruct agent on terse message composition and mesh_inbox usage
