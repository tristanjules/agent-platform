package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/tristanj/dusty/internal/state"
)

// EventMsg wraps a state.Event so it can be sent through the Bubble Tea
// message loop from goroutines that subscribe to the EventBus.
type EventMsg struct {
	Event state.Event
}

// StartBridge subscribes to relevant EventBus channels and forwards events
// to the Bubble Tea program via p.Send(). Each event type gets its own
// goroutine to avoid head-of-line blocking. The bridge stops when the
// context embedded in the subscribed channels is closed.
//
// Call this after creating the tea.Program but before calling p.Run().
func StartBridge(bus *state.EventBus, p *tea.Program) {
	eventTypes := []state.EventType{
		state.EventStateChanged,
		state.EventUserMessage,
		state.EventAgentTokens,
		state.EventAgentResponse,
		state.EventAgentError,
		state.EventSTTResult,
		state.EventTTSStarted,
		state.EventTTSDone,
	}

	for _, et := range eventTypes {
		ch := bus.Subscribe(et)
		go func(c <-chan state.Event) {
			for ev := range c {
				p.Send(EventMsg{Event: ev})
			}
		}(ch)
	}
}
