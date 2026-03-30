# event-bus Specification

## Purpose
TBD - created by archiving change phase-1-foundations. Update Purpose after archive.
## Requirements
### Requirement: EventBus supports typed pub/sub via Go channels
`EventBus.Subscribe(eventType)` SHALL return a `<-chan Event` that receives all events published with that `EventType`. Multiple subscribers for the same type SHALL each receive their own copy of every event.

#### Scenario: Single subscriber receives published event
- **WHEN** a subscriber channel is obtained via `Subscribe(EventUserMessage)` and an event of that type is published
- **THEN** the event is available on the channel with the correct `Type` and `Payload`

#### Scenario: Multiple subscribers each receive event
- **WHEN** two channels are subscribed to the same event type and an event is published
- **THEN** both channels receive the event independently

### Requirement: Publish is non-blocking
`EventBus.Publish(event)` SHALL NOT block the caller if a subscriber's channel is full. The event SHALL be silently dropped for that subscriber.

#### Scenario: Publish to a full channel does not block
- **WHEN** a subscriber's channel buffer is full (capacity 1, one unconsumed event)
- **THEN** a second `Publish()` call returns immediately without panicking or blocking

### Requirement: EventBus.Close cleans up all subscriber channels
`EventBus.Close()` SHALL close all registered subscriber channels, signaling end-of-stream to all consumers.

#### Scenario: Channels closed after bus close
- **WHEN** `bus.Close()` is called
- **THEN** all subscribed channels are closed and drainable

### Requirement: Events carry a timestamp
Every `Event` created with `NewEvent(type, payload)` SHALL have a non-zero `Timestamp` field set to the time of creation.

#### Scenario: NewEvent sets timestamp
- **WHEN** `NewEvent(EventUserMessage, "hello")` is called
- **THEN** the returned event has `Timestamp` set to approximately the current time

### Requirement: EventBus methods are thread-safe
`Subscribe`, `Publish`, and `Close` SHALL be safe to call concurrently from multiple goroutines.

#### Scenario: Concurrent subscribe and publish
- **WHEN** goroutines call `Subscribe()` and `Publish()` simultaneously
- **THEN** no data race is detected

### Requirement: EventBus supports mesh communication event types
The EventBus SHALL support the following new event types: `EventMeshMessageReceived`, `EventMeshNodeDiscovered`, `EventMeshNodeLost`, and `EventNotificationTriggered`. These SHALL follow the same pub/sub mechanics as existing event types.

#### Scenario: Subscribe to mesh message events
- **WHEN** `Subscribe(EventMeshMessageReceived)` is called and a mesh message event is published
- **THEN** the subscriber channel receives the event with the mesh message as payload

#### Scenario: Subscribe to node discovery events
- **WHEN** `Subscribe(EventMeshNodeDiscovered)` is called and a discovery event is published
- **THEN** the subscriber channel receives the event with the peer info as payload

#### Scenario: Subscribe to node lost events
- **WHEN** `Subscribe(EventMeshNodeLost)` is called and a node-lost event is published
- **THEN** the subscriber channel receives the event with the lost peer's node ID as payload

#### Scenario: Subscribe to notification events
- **WHEN** `Subscribe(EventNotificationTriggered)` is called and a notification event is published
- **THEN** the subscriber channel receives the event with notification metadata as payload

