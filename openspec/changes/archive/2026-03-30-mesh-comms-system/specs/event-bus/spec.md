## ADDED Requirements

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
