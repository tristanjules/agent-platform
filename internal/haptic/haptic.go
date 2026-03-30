// Package haptic provides a vibration interface for DUSTY.
// The initial implementation is a stub that logs requests without hardware action.
// Future implementations can drive GPIO motors or BLE vibration hardware.
package haptic

import "log"

// Vibration pattern constants.
const (
	PatternShort  = "short"  // Single short pulse.
	PatternDouble = "double" // Two short pulses.
	PatternLong   = "long"   // One long pulse.
)

// HapticProvider triggers vibration patterns.
type HapticProvider interface {
	Vibrate(pattern string) error
}

// StubHaptic logs vibration requests without performing any hardware action.
type StubHaptic struct{}

// NewStub creates a StubHaptic.
func NewStub() *StubHaptic {
	return &StubHaptic{}
}

// Vibrate logs the requested pattern. Unknown patterns fall back to PatternShort.
func (h *StubHaptic) Vibrate(pattern string) error {
	switch pattern {
	case PatternShort, PatternDouble, PatternLong:
		log.Printf("[haptic] vibrate: %s", pattern)
	default:
		log.Printf("[haptic] vibrate: unknown pattern %q, using %s", pattern, PatternShort)
	}
	return nil
}
