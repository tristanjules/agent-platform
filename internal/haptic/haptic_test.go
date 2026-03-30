package haptic_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/haptic"
)

func TestStubHapticKnownPatterns(t *testing.T) {
	h := haptic.NewStub()
	for _, p := range []string{haptic.PatternShort, haptic.PatternDouble, haptic.PatternLong} {
		if err := h.Vibrate(p); err != nil {
			t.Errorf("Vibrate(%q) returned error: %v", p, err)
		}
	}
}

func TestStubHapticUnknownPatternFallsBack(t *testing.T) {
	h := haptic.NewStub()
	// Should not error — falls back to PatternShort.
	if err := h.Vibrate("unknown_pattern"); err != nil {
		t.Errorf("Vibrate(unknown) should not error, got: %v", err)
	}
}

func TestStubHapticImplementsInterface(t *testing.T) {
	var _ haptic.HapticProvider = haptic.NewStub()
}
