# mesh-message-store Specification

## Purpose
Persistent message store for all mesh communications. Tracks message status (unread/read/dismissed/sent), supports inbox queries, and manages retention limits.

## Requirements
### Requirement: Message store persists all received mesh messages
The message store SHALL persist all received mesh messages to `mesh-messages.json`. Each entry SHALL contain: message ID, sender node ID, message type, content, received timestamp, and status.

#### Scenario: Received message is stored
- **WHEN** a mesh message is received and decoded
- **THEN** an entry is created in the message store with status `unread` and persisted to disk

#### Scenario: Messages persist across restarts
- **WHEN** the agent restarts with 5 previously stored messages
- **THEN** all 5 messages are loaded with their original statuses

### Requirement: Messages have read/unread/dismissed status
Each stored message SHALL have a status field with one of three values: `unread` (received but not yet viewed), `read` (user accepted and viewed), or `dismissed` (user explicitly dismissed without viewing). Status transitions SHALL be: `unread -> read`, `unread -> dismissed`, `dismissed -> read`.

#### Scenario: Message accepted transitions to read
- **WHEN** the user accepts an incoming transmission notification
- **THEN** the message status transitions from `unread` to `read` with a `readAt` timestamp

#### Scenario: Message dismissed transitions to dismissed
- **WHEN** the user dismisses an incoming transmission notification
- **THEN** the message status transitions from `unread` to `dismissed`

#### Scenario: Dismissed message can be replayed to read
- **WHEN** the user replays a previously dismissed message
- **THEN** the message status transitions from `dismissed` to `read`

### Requirement: Message store supports inbox queries
The store SHALL provide: `ListUnread()`, `ListAll(limit)`, `ListFromPeer(nodeID, limit)`, `GetMessage(id)`, `MarkRead(id)`, `MarkDismissed(id)`, and `UnreadCount()`.

#### Scenario: List unread messages
- **WHEN** `ListUnread()` is called with 3 unread and 2 read messages
- **THEN** only the 3 unread messages are returned, ordered by received time (newest first)

#### Scenario: List messages from a specific peer
- **WHEN** `ListFromPeer(nodeB_ID, 10)` is called
- **THEN** the 10 most recent messages from node B are returned regardless of status

#### Scenario: Unread count
- **WHEN** `UnreadCount()` is called with 3 unread messages
- **THEN** the integer 3 is returned

### Requirement: Message store has a configurable retention limit
The store SHALL retain a configurable maximum number of messages (default: 200). When the limit is exceeded, the oldest `read` messages SHALL be removed first, then oldest `dismissed`. `Unread` messages SHALL never be auto-removed.

#### Scenario: Old read messages are pruned
- **WHEN** the store has 200 messages and a new one arrives, with 50 read messages older than all unread/dismissed
- **THEN** the oldest read message is removed and the new message is stored

#### Scenario: Unread messages are never pruned
- **WHEN** the store has 200 messages and all are unread
- **THEN** the new message is stored and the limit is temporarily exceeded rather than dropping unread messages

### Requirement: Outbound messages are also stored
Messages sent via `mesh_send` SHALL also be stored in the message store with status `sent` and a `sentAt` timestamp. This allows full conversation threading per peer.

#### Scenario: Sent message is stored
- **WHEN** a message is sent via `mesh_send` to node B
- **THEN** an entry is created with status `sent`, the sender as self, and the target as node B
