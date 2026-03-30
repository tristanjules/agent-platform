## ADDED Requirements

### Requirement: mesh_send tool implements the Tool interface
The `mesh_send` tool SHALL implement the existing `Tool` interface (`Name()`, `Description()`, `Execute(args)`) and be registered in the agent's tool set.

#### Scenario: Tool is discoverable
- **WHEN** the agent lists available tools
- **THEN** `mesh_send` appears with its name and description

### Requirement: mesh_send accepts target and message arguments
`Execute(args)` SHALL accept `target` (node ID or agent name) and `message` (text content) as required arguments. It SHALL optionally accept `type` (message type, default "msg").

#### Scenario: Send text message by agent name
- **WHEN** `Execute({"target": "DUSTY-B", "message": "meet at grid 4-7"})` is called
- **THEN** the peer registry resolves "DUSTY-B" to a node ID and the message is enqueued for transmission

#### Scenario: Send command by node ID
- **WHEN** `Execute({"target": "!abcd1234", "message": "nav", "type": "cmd"})` is called
- **THEN** a command message is enqueued for the specified node ID

#### Scenario: Unknown target returns error
- **WHEN** `Execute({"target": "DUSTY-Z", "message": "hello"})` is called and "DUSTY-Z" is not in the peer registry
- **THEN** an error string is returned indicating the target is unknown

### Requirement: mesh_send summarises long messages before transmission
If the `message` argument would exceed the payload budget after encoding, the tool SHALL return an error instructing the agent to shorten the message, rather than silently truncating.

#### Scenario: Message too long
- **WHEN** `Execute()` is called with a 500-character message
- **THEN** an error is returned: "Message too long for mesh transmission. Please summarise to under N characters."

### Requirement: mesh_send returns transmission status
The tool SHALL return a human-readable string indicating success (message queued) or failure (disconnected, unknown target, message too long).

#### Scenario: Successful enqueue
- **WHEN** a valid message is enqueued
- **THEN** the return string includes "Message queued for transmission to <target>"
