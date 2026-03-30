package state

import (
	"fmt"
	"sync"
)

// AgentState represents the current mode of the DUSTY agent.
type AgentState int

const (
	StateIdle       AgentState = iota
	StateListening             // Microphone active, capturing audio.
	StateProcessing            // STT running on captured audio.
	StateThinking              // LLM generating a response.
	StateSpeaking              // TTS playing audio output.
	StateError                 // Recoverable error state.
	StateWarmup                // System initializing.
)

// String returns a human-readable name for the state.
func (s AgentState) String() string {
	names := [...]string{
		"Idle",
		"Listening",
		"Processing",
		"Thinking",
		"Speaking",
		"Error",
		"Warmup",
	}
	if int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

// validTransitions defines which state transitions are allowed.
// Key is the current state, value is the set of states it can transition to.
var validTransitions = map[AgentState]map[AgentState]bool{
	StateWarmup: {
		StateIdle:  true,
		StateError: true,
	},
	StateIdle: {
		StateListening: true,
		StateThinking:  true, // Direct text input skips listening/processing.
		StateError:     true,
		StateWarmup:    true,
	},
	StateListening: {
		StateProcessing: true,
		StateIdle:       true, // Cancelled / silence timeout.
		StateError:      true,
	},
	StateProcessing: {
		StateThinking: true,
		StateIdle:     true, // STT produced no usable text.
		StateError:    true,
	},
	StateThinking: {
		StateSpeaking: true,
		StateIdle:     true, // Text-only mode, no TTS.
		StateError:    true,
	},
	StateSpeaking: {
		StateIdle:      true,
		StateListening: true, // Continuous conversation mode.
		StateError:     true,
	},
	StateError: {
		StateIdle:   true,
		StateWarmup: true,
	},
}

// StateTransition records a state change.
type StateTransition struct {
	From AgentState
	To   AgentState
}

// TransitionCallback is called when the state machine transitions.
type TransitionCallback func(from, to AgentState)

// StateMachine manages the agent's current state with thread-safe transitions
// and enforces the valid transition table.
type StateMachine struct {
	current   AgentState
	callbacks []TransitionCallback
	bus       *EventBus
	mu        sync.RWMutex
}

// NewStateMachine creates a state machine starting in the given state.
// If an EventBus is provided, state transitions are published as events.
func NewStateMachine(initial AgentState, bus *EventBus) *StateMachine {
	return &StateMachine{
		current: initial,
		bus:     bus,
	}
}

// Current returns the current state (thread-safe).
func (sm *StateMachine) Current() AgentState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Transition attempts to move to the given state. Returns an error if the
// transition is not valid from the current state.
func (sm *StateMachine) Transition(to AgentState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	from := sm.current

	allowed, exists := validTransitions[from]
	if !exists || !allowed[to] {
		return fmt.Errorf("invalid state transition: %s -> %s", from, to)
	}

	sm.current = to

	// Notify callbacks.
	for _, cb := range sm.callbacks {
		cb(from, to)
	}

	// Publish to event bus.
	if sm.bus != nil {
		sm.bus.Publish(NewEvent(EventStateChanged, StateTransition{From: from, To: to}))
	}

	return nil
}

// ForceState sets the state without checking the transition table.
// Use only during initialization or error recovery.
func (sm *StateMachine) ForceState(s AgentState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current = s
}

// OnTransition registers a callback that fires on every state transition.
func (sm *StateMachine) OnTransition(fn TransitionCallback) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.callbacks = append(sm.callbacks, fn)
}
