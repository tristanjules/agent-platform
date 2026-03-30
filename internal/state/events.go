// Package state provides the central event bus and state machine for DUSTY.
// All components communicate through the event bus using typed events.
package state

import (
	"sync"
	"time"
)

// EventType identifies the kind of event flowing through the bus.
type EventType int

const (
	// State transition events.
	EventStateChanged EventType = iota

	// Agent lifecycle events.
	EventUserMessage
	EventAgentTokens    // Streaming tokens from LLM.
	EventAgentResponse  // Complete agent response.
	EventAgentError

	// Audio pipeline events (Phase 2).
	EventAudioCaptured
	EventSTTResult
	EventTTSStarted
	EventTTSDone

	// Wake word events (Phase 5).
	EventWakeWordDetected

	// Mesh communication events.
	EventMeshMessageReceived  // Incoming mesh message from a peer.
	EventMeshNodeDiscovered   // New peer seen for the first time.
	EventMeshNodeLost         // Peer has gone stale (TTL expired).
	EventNotificationTriggered // Notification sound/haptic fired.
)

// String returns a human-readable name for the event type.
func (et EventType) String() string {
	names := [...]string{
		"StateChanged",
		"UserMessage",
		"AgentTokens",
		"AgentResponse",
		"AgentError",
		"AudioCaptured",
		"STTResult",
		"TTSStarted",
		"TTSDone",
		"WakeWordDetected",
		"MeshMessageReceived",
		"MeshNodeDiscovered",
		"MeshNodeLost",
		"NotificationTriggered",
	}
	if int(et) < len(names) {
		return names[et]
	}
	return "Unknown"
}

// Event is a message flowing through the event bus.
type Event struct {
	Type      EventType
	Payload   any
	Timestamp time.Time
}

// NewEvent creates an event with the current timestamp.
func NewEvent(eventType EventType, payload any) Event {
	return Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// EventBus provides a publish/subscribe mechanism using Go channels.
// Subscribers receive events on dedicated channels, decoupling producers
// from consumers across the system.
type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
	bufferSize  int
}

// NewEventBus creates an event bus with the given channel buffer size.
func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
		bufferSize:  bufferSize,
	}
}

// Subscribe returns a channel that receives events of the given type.
// The caller should read from the channel to avoid blocking publishers.
func (eb *EventBus) Subscribe(eventType EventType) <-chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, eb.bufferSize)
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

// Publish sends an event to all subscribers of that event type.
// Non-blocking: if a subscriber's channel is full, the event is dropped
// for that subscriber (prevents slow consumers from blocking the system).
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, ch := range eb.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
			// Subscriber channel full — drop event to prevent blocking.
		}
	}
}

// Close closes all subscriber channels. Call during shutdown.
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for eventType, channels := range eb.subscribers {
		for _, ch := range channels {
			close(ch)
		}
		delete(eb.subscribers, eventType)
	}
}
