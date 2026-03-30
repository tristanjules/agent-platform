// Package audio — wake word detection placeholder.
// Full implementation deferred to Phase 5 (OpenWakeWord ONNX).
package audio

import "errors"

// ErrWakeWordNotImplemented is returned when wake word detection is invoked
// before Phase 5 is implemented.
var ErrWakeWordNotImplemented = errors.New("wake word detection not yet implemented (Phase 5)")

// WakeWordDetector listens for a trigger phrase in a PCM audio stream.
// This interface is defined now so Phase 5 can implement it without
// changing downstream consumers.
type WakeWordDetector interface {
	// Detected returns a channel that receives the matched wake word string
	// each time it is detected in the audio stream.
	Detected() <-chan string
	// Close stops the detector.
	Close() error
}

// NullWakeWordDetector is a stub that never triggers.
// Used in Phase 2–4 where wake word is not yet active.
type NullWakeWordDetector struct {
	ch chan string
}

// NewNullWakeWordDetector returns a wake word detector that never fires.
func NewNullWakeWordDetector() *NullWakeWordDetector {
	return &NullWakeWordDetector{ch: make(chan string)}
}

func (n *NullWakeWordDetector) Detected() <-chan string { return n.ch }
func (n *NullWakeWordDetector) Close() error            { return nil }
