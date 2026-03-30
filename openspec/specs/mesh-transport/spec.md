# mesh-transport Specification

## Purpose
Serial transport layer for LoRa mesh communication via Meshtastic nodes. Handles USB serial connection, protobuf encoding/decoding, rate-limited message queuing, and reconnection logic.

## Requirements
### Requirement: Serial adapter connects to Meshtastic node via USB
The mesh transport SHALL open a serial connection to a Meshtastic-compatible LoRa node at the configured serial port path and baud rate. It SHALL use Meshtastic's serial protobuf API for all communication.

#### Scenario: Successful serial connection
- **WHEN** `Transport.Open()` is called with a valid serial port path (e.g., `/dev/ttyUSB0`)
- **THEN** a serial connection is established and the transport reports `Connected` status

#### Scenario: Serial port not found
- **WHEN** `Transport.Open()` is called with a non-existent serial port
- **THEN** an error is returned and the transport reports `Disconnected` status

### Requirement: Transport sends messages via serial protobuf
`Transport.Send(nodeID, payload)` SHALL encode the payload into a Meshtastic protobuf `ToRadio` packet and write it to the serial connection. It SHALL return an error if the serial connection is not open.

#### Scenario: Successful send
- **WHEN** `Send(nodeID, payload)` is called with a valid connection and payload under 200 bytes
- **THEN** the payload is written to the serial port as a Meshtastic protobuf packet

#### Scenario: Send fails when disconnected
- **WHEN** `Send()` is called while the serial connection is closed
- **THEN** an error is returned indicating the transport is disconnected

### Requirement: Transport receives messages via serial protobuf
The transport SHALL continuously read from the serial connection, decode incoming Meshtastic `FromRadio` protobuf packets, and deliver received messages to a channel.

#### Scenario: Incoming message delivered
- **WHEN** a Meshtastic packet arrives on the serial port containing a mesh payload
- **THEN** the decoded message is available on the `Transport.Incoming()` channel

#### Scenario: Malformed packet is dropped
- **WHEN** a corrupted or unparseable packet is received on serial
- **THEN** the packet is logged and discarded without crashing the receive loop

### Requirement: Message queue respects duty cycle limits
The transport SHALL maintain an outbound message queue with a configurable rate limiter (default: 1 message per 30 seconds). Messages SHALL be dequeued and sent at the rate limit. If the queue exceeds its capacity, the oldest message SHALL be dropped.

#### Scenario: Messages are rate-limited
- **WHEN** three messages are enqueued within 5 seconds with a 30-second rate limit
- **THEN** the first message is sent immediately, the second after 30 seconds, and the third after 60 seconds

#### Scenario: Queue overflow drops oldest
- **WHEN** the queue is at capacity (default 16) and a new message is enqueued
- **THEN** the oldest queued message is dropped and the new message is added

### Requirement: Transport reconnects on serial failure
If the serial connection drops, the transport SHALL attempt to reconnect with exponential backoff (starting at 1 second, max 60 seconds). It SHALL publish a reconnection event when the connection is restored.

#### Scenario: Reconnection after USB disconnect
- **WHEN** the serial connection is lost (USB cable disconnected)
- **THEN** the transport retries with exponential backoff and reconnects when the port becomes available again

### Requirement: Transport supports graceful shutdown
`Transport.Close()` SHALL drain the outbound queue (best-effort, with a timeout), close the serial connection, and close the incoming message channel.

#### Scenario: Graceful shutdown drains queue
- **WHEN** `Close()` is called with messages in the outbound queue
- **THEN** queued messages are sent (up to a 5-second timeout), then the serial port and channels are closed
