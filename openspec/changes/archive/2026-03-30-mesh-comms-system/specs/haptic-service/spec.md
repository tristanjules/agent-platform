## ADDED Requirements

### Requirement: Haptic service exposes a vibration interface
The haptic service SHALL expose a `Vibrate(pattern)` method that triggers a vibration pattern. The initial implementation SHALL be a stub that logs the vibration request without performing any hardware action.

#### Scenario: Stub logs vibration
- **WHEN** `Vibrate("short")` is called on the stub haptic service
- **THEN** the request is logged and no error is returned

### Requirement: Haptic interface supports future hardware implementations
The haptic interface SHALL be defined as a Go interface (`HapticProvider`) with `Vibrate(pattern string) error` so that future implementations (e.g., GPIO motor driver) can be swapped in without changing consumers.

#### Scenario: Interface is implementable
- **WHEN** a new haptic implementation satisfies the `HapticProvider` interface
- **THEN** it can be injected into the notification system without modifying notification code

### Requirement: Haptic patterns are predefined constants
The service SHALL define pattern constants: `PatternShort` (single short pulse), `PatternDouble` (two short pulses), `PatternLong` (one long pulse). Unknown patterns SHALL fall back to `PatternShort`.

#### Scenario: Unknown pattern falls back
- **WHEN** `Vibrate("unknown_pattern")` is called
- **THEN** the `PatternShort` behavior is used as fallback
