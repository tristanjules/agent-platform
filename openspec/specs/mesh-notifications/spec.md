# mesh-notifications Specification

## Purpose
Multi-modal notification system for incoming mesh communications. Handles sound, haptic feedback, TUI/REPL transmission overlays, unread message queuing, and subtle peer status announcements.

## Requirements
### Requirement: Incoming messages trigger a multi-modal notification
When a mesh message is received, the notification system SHALL: (1) play a notification sound, (2) trigger the haptic service, and (3) publish an `EventNotificationTriggered` event.

#### Scenario: Full notification sequence
- **WHEN** a mesh message of type `msg` is received from a known peer
- **THEN** the notification sound plays, the haptic service is triggered, and `EventNotificationTriggered` is published

### Requirement: TUI displays retro "INCOMING TRANSMISSION" announcement
When the TUI receives an `EventMeshMessageReceived` event, it SHALL display a theatrical announcement overlay styled as a retro sci-fi incoming transmission. The overlay SHALL show the sender's agent name, owner name, and a prompt to accept or dismiss.

#### Scenario: TUI shows transmission announcement
- **WHEN** `EventMeshMessageReceived` is received in TUI mode with sender "DUSTY-B" (owner: "Lourens")
- **THEN** a full-width overlay appears with text like ">>> INCOMING TRANSMISSION <<< From: DUSTY-B [Lourens]" and "[Enter] Accept / [Esc] Dismiss"

#### Scenario: User accepts transmission
- **WHEN** the user presses Enter on the transmission overlay
- **THEN** the message content is displayed in the conversation pane and read aloud (if TTS is available)

#### Scenario: User dismisses transmission
- **WHEN** the user presses Escape on the transmission overlay
- **THEN** the overlay closes without displaying the message content, and the message is logged silently

### Requirement: REPL displays inline transmission notification
When the REPL receives an `EventMeshMessageReceived` event, it SHALL print an inline notification with the sender info and prompt the user with `[y/n]` to display the message.

#### Scenario: REPL shows transmission prompt
- **WHEN** `EventMeshMessageReceived` is received in REPL mode
- **THEN** the REPL prints `\n>>> INCOMING TRANSMISSION from DUSTY-B [Lourens] <<<\nDisplay message? [y/n]: `

#### Scenario: REPL user accepts
- **WHEN** the user types `y` at the REPL transmission prompt
- **THEN** the message content is printed to stdout

### Requirement: Notification sound plays on incoming message
The notification system SHALL play a short audio file (`assets/sounds/transmission.wav`) when an incoming message arrives. If audio playback is unavailable, it SHALL fall back to the terminal bell character (`\a`).

#### Scenario: Sound plays successfully
- **WHEN** an incoming message triggers a notification and audio playback is available
- **THEN** `transmission.wav` is played through the audio output

#### Scenario: Fallback to terminal bell
- **WHEN** an incoming message triggers a notification and audio playback is unavailable
- **THEN** the terminal bell character is written to stdout

### Requirement: Unread messages are queued for review on return
When messages arrive while no user interaction is detected (no input for a configurable idle period, default 5 minutes), they SHALL be stored as `unread` in the message store without triggering the full transmission overlay. When user activity resumes, the TUI/REPL SHALL display an unread summary banner showing the count and senders.

#### Scenario: Messages arrive while user is away
- **WHEN** 3 messages arrive from DUSTY-B during a 30-minute idle period
- **THEN** all 3 are stored as `unread` and no transmission overlay is shown

#### Scenario: Unread banner on return
- **WHEN** the user resumes interaction with 3 unread messages from DUSTY-B
- **THEN** the TUI displays a banner: "3 unread transmissions from DUSTY-B [R] Review [C] Continue"

#### Scenario: User reviews unread queue
- **WHEN** the user presses R on the unread banner
- **THEN** messages are presented one at a time in chronological order with accept/dismiss per message

#### Scenario: User continues past unread banner
- **WHEN** the user presses C on the unread banner
- **THEN** the banner is dismissed and messages remain `unread` in the store for later query via `mesh_inbox`

### Requirement: Discovery and heartbeat events are announced subtly
Discovery (`EventMeshNodeDiscovered`) and node-lost (`EventMeshNodeLost`) events SHALL be displayed as system messages in the conversation pane, not as full transmission overlays.

#### Scenario: Node discovery announcement
- **WHEN** a new node "DUSTY-B" is discovered
- **THEN** a system message "DUSTY-B [Lourens] has entered mesh range" is added to the conversation pane

#### Scenario: Node lost announcement
- **WHEN** node "DUSTY-B" is marked as lost
- **THEN** a system message "DUSTY-B [Lourens] signal lost" is added to the conversation pane
