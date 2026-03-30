package state_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/state"
)

func TestStateMachineInitialState(t *testing.T) {
	sm := state.NewStateMachine(state.StateWarmup, nil)
	if sm.Current() != state.StateWarmup {
		t.Errorf("expected Warmup, got %s", sm.Current())
	}
}

func TestValidTransitions(t *testing.T) {
	sm := state.NewStateMachine(state.StateWarmup, nil)

	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatalf("Warmup -> Idle should be valid: %v", err)
	}
	if err := sm.Transition(state.StateThinking); err != nil {
		t.Fatalf("Idle -> Thinking should be valid: %v", err)
	}
	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatalf("Thinking -> Idle should be valid: %v", err)
	}
}

func TestInvalidTransition(t *testing.T) {
	sm := state.NewStateMachine(state.StateIdle, nil)

	err := sm.Transition(state.StateSpeaking)
	if err == nil {
		t.Fatal("Idle -> Speaking should be invalid")
	}
}

func TestTransitionCallback(t *testing.T) {
	sm := state.NewStateMachine(state.StateWarmup, nil)

	var callbackFrom, callbackTo state.AgentState
	var called bool
	sm.OnTransition(func(from, to state.AgentState) {
		callbackFrom = from
		callbackTo = to
		called = true
	})

	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("callback was not called")
	}
	if callbackFrom != state.StateWarmup {
		t.Errorf("expected from Warmup, got %s", callbackFrom)
	}
	if callbackTo != state.StateIdle {
		t.Errorf("expected to Idle, got %s", callbackTo)
	}
}

func TestTransitionDoesNotChangeStateOnError(t *testing.T) {
	sm := state.NewStateMachine(state.StateIdle, nil)

	_ = sm.Transition(state.StateSpeaking) // Invalid

	if sm.Current() != state.StateIdle {
		t.Errorf("state should remain Idle after failed transition, got %s", sm.Current())
	}
}

func TestForceState(t *testing.T) {
	sm := state.NewStateMachine(state.StateIdle, nil)
	sm.ForceState(state.StateSpeaking)
	if sm.Current() != state.StateSpeaking {
		t.Errorf("expected Speaking after ForceState, got %s", sm.Current())
	}
}

func TestEventBusPublishSubscribe(t *testing.T) {
	bus := state.NewEventBus(8)
	defer bus.Close()

	ch := bus.Subscribe(state.EventUserMessage)

	payload := "hello world"
	bus.Publish(state.NewEvent(state.EventUserMessage, payload))

	evt := <-ch
	if evt.Type != state.EventUserMessage {
		t.Errorf("expected EventUserMessage, got %v", evt.Type)
	}
	if evt.Payload.(string) != payload {
		t.Errorf("expected payload %q, got %v", payload, evt.Payload)
	}
}

func TestEventBusNonBlockingOnFullChannel(t *testing.T) {
	bus := state.NewEventBus(1) // Buffer of 1
	defer bus.Close()

	bus.Subscribe(state.EventUserMessage)

	// Publish twice — second publish should not block even though channel is full.
	bus.Publish(state.NewEvent(state.EventUserMessage, "first"))
	bus.Publish(state.NewEvent(state.EventUserMessage, "second")) // Should not block.
}

func TestStateMachinePublishesToBus(t *testing.T) {
	bus := state.NewEventBus(8)
	defer bus.Close()

	ch := bus.Subscribe(state.EventStateChanged)

	sm := state.NewStateMachine(state.StateWarmup, bus)
	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatal(err)
	}

	evt := <-ch
	transition, ok := evt.Payload.(state.StateTransition)
	if !ok {
		t.Fatalf("expected StateTransition payload, got %T", evt.Payload)
	}
	if transition.From != state.StateWarmup || transition.To != state.StateIdle {
		t.Errorf("unexpected transition: %v", transition)
	}
}
