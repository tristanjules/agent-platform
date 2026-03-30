# state-machine Specification

## Purpose
TBD - created by archiving change phase-1-foundations. Update Purpose after archive.
## Requirements
### Requirement: State machine enforces valid transitions
The `StateMachine.Transition(to)` method SHALL return an error if the transition from the current state to `to` is not in the valid transition table. The state SHALL NOT change on an invalid transition.

#### Scenario: Valid transition succeeds
- **WHEN** `Transition(StateIdle)` is called while in `StateWarmup`
- **THEN** no error is returned and `Current()` returns `StateIdle`

#### Scenario: Invalid transition returns error and preserves state
- **WHEN** `Transition(StateSpeaking)` is called while in `StateIdle`
- **THEN** an error is returned and `Current()` still returns `StateIdle`

### Requirement: Transition callbacks fire on every successful transition
Any function registered via `OnTransition(fn)` SHALL be called synchronously with `(from, to)` arguments immediately after a successful transition.

#### Scenario: Callback fires with correct states
- **WHEN** a callback is registered and `Transition(StateIdle)` is called from `StateWarmup`
- **THEN** the callback receives `from=StateWarmup, to=StateIdle`

### Requirement: ForceState bypasses the transition table
`ForceState(s)` SHALL set the state unconditionally, without firing validation or callbacks. It SHALL be used only for initialization and error recovery.

#### Scenario: Force state ignores table
- **WHEN** `ForceState(StateSpeaking)` is called while in `StateIdle`
- **THEN** `Current()` returns `StateSpeaking` with no error

### Requirement: State machine publishes transitions to the EventBus
When an `EventBus` is provided at construction, every successful `Transition()` SHALL publish an `EventStateChanged` event with a `StateTransition{From, To}` payload.

#### Scenario: Transition event published to bus
- **WHEN** `Transition(StateIdle)` succeeds and an `EventBus` was provided
- **THEN** an `EventStateChanged` event is available on a subscribed channel with the correct `From` and `To` values

### Requirement: All state machine methods are thread-safe
`Transition`, `Current`, `ForceState`, and `OnTransition` SHALL be safe to call concurrently from multiple goroutines.

#### Scenario: Concurrent transitions don't race
- **WHEN** multiple goroutines call `Transition()` simultaneously
- **THEN** no data race is detected and the state machine remains consistent

### Requirement: State machine includes StateReceivingTransmission state
The state machine SHALL include a `StateReceivingTransmission` state. Valid transitions TO this state SHALL be from `StateIdle` and `StateProcessing`. Valid transitions FROM this state SHALL be back to the originating state (the state before the interrupt).

#### Scenario: Transition from Idle to ReceivingTransmission
- **WHEN** `Transition(StateReceivingTransmission)` is called while in `StateIdle`
- **THEN** the transition succeeds and `Current()` returns `StateReceivingTransmission`

#### Scenario: Transition from Processing to ReceivingTransmission
- **WHEN** `Transition(StateReceivingTransmission)` is called while in `StateProcessing`
- **THEN** the transition succeeds and `Current()` returns `StateReceivingTransmission`

#### Scenario: Return to previous state after transmission handled
- **WHEN** the user accepts or dismisses a transmission while in `StateReceivingTransmission`
- **THEN** the state machine transitions back to the state it was in before the interrupt

#### Scenario: Invalid transition from Speaking
- **WHEN** `Transition(StateReceivingTransmission)` is called while in `StateSpeaking`
- **THEN** an error is returned and the state remains `StateSpeaking`

