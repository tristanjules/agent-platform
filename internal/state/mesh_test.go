package state_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/state"
)

// TestReceivingTransmissionFromIdle verifies the interrupt flow from Idle.
func TestReceivingTransmissionFromIdle(t *testing.T) {
	sm := state.NewStateMachine(state.StateIdle, nil)

	if err := sm.Transition(state.StateReceivingTransmission); err != nil {
		t.Fatalf("Idle -> ReceivingTransmission should be valid: %v", err)
	}
	if sm.Current() != state.StateReceivingTransmission {
		t.Errorf("expected ReceivingTransmission, got %s", sm.Current())
	}
}

// TestReceivingTransmissionFromProcessing verifies interrupt from Processing.
func TestReceivingTransmissionFromProcessing(t *testing.T) {
	sm := state.NewStateMachine(state.StateProcessing, nil)

	if err := sm.Transition(state.StateReceivingTransmission); err != nil {
		t.Fatalf("Processing -> ReceivingTransmission should be valid: %v", err)
	}
}

// TestReceivingTransmissionReturnsToIdle verifies accept/dismiss returns to Idle.
func TestReceivingTransmissionReturnsToIdle(t *testing.T) {
	sm := state.NewStateMachine(state.StateIdle, nil)

	if err := sm.Transition(state.StateReceivingTransmission); err != nil {
		t.Fatal(err)
	}
	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatalf("ReceivingTransmission -> Idle should be valid: %v", err)
	}
	if sm.Current() != state.StateIdle {
		t.Errorf("expected Idle, got %s", sm.Current())
	}
}

// TestReceivingTransmissionReturnsToProcessing verifies return to Processing.
func TestReceivingTransmissionReturnsToProcessing(t *testing.T) {
	sm := state.NewStateMachine(state.StateProcessing, nil)

	if err := sm.Transition(state.StateReceivingTransmission); err != nil {
		t.Fatal(err)
	}
	if err := sm.Transition(state.StateProcessing); err != nil {
		t.Fatalf("ReceivingTransmission -> Processing should be valid: %v", err)
	}
}

// TestInvalidTransmissionFromSpeaking verifies Speaking cannot be interrupted.
func TestInvalidTransmissionFromSpeaking(t *testing.T) {
	sm := state.NewStateMachine(state.StateSpeaking, nil)

	err := sm.Transition(state.StateReceivingTransmission)
	if err == nil {
		t.Fatal("Speaking -> ReceivingTransmission should be invalid")
	}
	if sm.Current() != state.StateSpeaking {
		t.Errorf("state should remain Speaking, got %s", sm.Current())
	}
}

// TestMeshEventTypes verifies new event types are subscribable on the bus.
func TestMeshEventTypes(t *testing.T) {
	bus := state.NewEventBus(8)
	defer bus.Close()

	tests := []struct {
		name      string
		eventType state.EventType
		payload   any
	}{
		{"MeshMessageReceived", state.EventMeshMessageReceived, "test-message"},
		{"MeshNodeDiscovered", state.EventMeshNodeDiscovered, "node-id-123"},
		{"MeshNodeLost", state.EventMeshNodeLost, "node-id-123"},
		{"NotificationTriggered", state.EventNotificationTriggered, "transmission"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := bus.Subscribe(tt.eventType)
			bus.Publish(state.NewEvent(tt.eventType, tt.payload))

			evt := <-ch
			if evt.Type != tt.eventType {
				t.Errorf("expected %v, got %v", tt.eventType, evt.Type)
			}
		})
	}
}

// TestMeshEventTypeStrings verifies new event types have readable names.
func TestMeshEventTypeStrings(t *testing.T) {
	tests := []struct {
		eventType state.EventType
		want      string
	}{
		{state.EventMeshMessageReceived, "MeshMessageReceived"},
		{state.EventMeshNodeDiscovered, "MeshNodeDiscovered"},
		{state.EventMeshNodeLost, "MeshNodeLost"},
		{state.EventNotificationTriggered, "NotificationTriggered"},
	}
	for _, tt := range tests {
		if got := tt.eventType.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.eventType, got, tt.want)
		}
	}
}
