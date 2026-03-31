## ADDED Requirements

### Requirement: Bridge forwards EventBus events into Bubble Tea loop
`StartBridge(bus, program)` SHALL subscribe to all display-relevant EventBus channels and forward each received event to the Bubble Tea program via `program.Send(EventMsg{Event: ev})`.

#### Scenario: State change event forwarded
- **WHEN** the agent transitions state and publishes `EventStateChanged`
- **THEN** the Bubble Tea program receives an `EventMsg` containing the `EventStateChanged` event

#### Scenario: Token event forwarded
- **WHEN** the agent publishes `EventAgentTokens` with a token string
- **THEN** the Bubble Tea program receives an `EventMsg` containing the token

### Requirement: One goroutine per event type
The bridge SHALL spawn one goroutine per subscribed event type to prevent head-of-line blocking between unrelated event streams.

#### Scenario: Slow token consumer does not block state events
- **WHEN** the token event channel is slow to drain
- **THEN** state change events are still forwarded without delay

### Requirement: Subscribed event types
The bridge SHALL subscribe to: EventStateChanged, EventUserMessage, EventAgentTokens, EventAgentResponse, EventAgentError, EventSTTResult, EventTTSStarted, EventTTSDone.

#### Scenario: All required event types subscribed
- **WHEN** StartBridge is called
- **THEN** goroutines are started for all 8 event types listed above

### Requirement: Bridge stops when bus channels are closed
Bridge goroutines SHALL terminate naturally when the EventBus is closed (channel range loop exits on close).

#### Scenario: Clean shutdown on bus close
- **WHEN** `bus.Close()` is called during shutdown
- **THEN** all bridge goroutines exit without goroutine leaks
