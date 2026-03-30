## ADDED Requirements

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
