package audio_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/audio"
)

// silence returns a frame of zeroed samples (below any reasonable threshold).
func silence(n int) []float32 {
	return make([]float32, n)
}

// speech returns a frame of loud samples (above typical threshold).
func speech(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 0.5 // well above 0.02 threshold
	}
	return s
}

func TestRMSEnergy(t *testing.T) {
	tests := []struct {
		name    string
		samples []float32
		wantGt  float32 // result must be greater than this
		wantLt  float32 // result must be less than this
	}{
		{"silence", silence(512), -1, 0.001},
		{"speech", speech(512), 0.4, 1.0},
		{"empty", []float32{}, -1, 0.001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := audio.RMSEnergy(tt.samples)
			if got <= tt.wantGt || got >= tt.wantLt {
				t.Errorf("RMSEnergy() = %f, want (%f, %f)", got, tt.wantGt, tt.wantLt)
			}
		})
	}
}

func TestIsSpeech(t *testing.T) {
	threshold := float32(0.02)
	if audio.IsSpeech(silence(512), threshold) {
		t.Error("silence should not be detected as speech")
	}
	if !audio.IsSpeech(speech(512), threshold) {
		t.Error("loud audio should be detected as speech")
	}
}

func TestVAD_SpeechThenSilence(t *testing.T) {
	cfg := audio.VADConfig{
		Threshold:     0.02,
		SilenceMs:     320, // 10 frames at 32ms each
		MaxSegmentSec: 30,
		SampleRate:    16000,
	}
	vad := audio.NewVAD(cfg)

	// Feed speech frames — no segment emitted yet.
	for i := 0; i < 5; i++ {
		seg, ok := vad.Process(speech(512))
		if ok {
			t.Errorf("frame %d: expected no segment during speech, got %d samples", i, len(seg))
		}
	}

	// Feed silence — segment should emit after ~10 silent frames.
	var emitted []float32
	for i := 0; i < 15; i++ {
		seg, ok := vad.Process(silence(512))
		if ok {
			emitted = seg
			break
		}
	}

	if emitted == nil {
		t.Fatal("expected a segment after silence timeout, got none")
	}
	if len(emitted) == 0 {
		t.Error("emitted segment should contain samples")
	}
}

func TestVAD_MaxSegmentLength(t *testing.T) {
	cfg := audio.VADConfig{
		Threshold:     0.02,
		SilenceMs:     800,
		MaxSegmentSec: 1, // Force a split at 1 second
		SampleRate:    16000,
	}
	vad := audio.NewVAD(cfg)

	frameSize := 512
	totalFrames := (cfg.SampleRate * cfg.MaxSegmentSec / frameSize) + 5 // slightly over limit

	var emitted []float32
	for i := 0; i < totalFrames; i++ {
		seg, ok := vad.Process(speech(frameSize))
		if ok {
			emitted = seg
			break
		}
	}

	if emitted == nil {
		t.Fatal("expected segment to be emitted at max length")
	}
	// Should contain approximately 1 second of audio.
	minExpected := cfg.SampleRate * cfg.MaxSegmentSec / 2
	if len(emitted) < minExpected {
		t.Errorf("emitted segment too short: got %d samples, want >= %d", len(emitted), minExpected)
	}
}

func TestVAD_PureSilenceDoesNotEmit(t *testing.T) {
	cfg := audio.DefaultVADConfig()
	vad := audio.NewVAD(cfg)

	for i := 0; i < 100; i++ {
		seg, ok := vad.Process(silence(512))
		if ok {
			t.Errorf("pure silence should never emit a segment, got %d samples at frame %d", len(seg), i)
		}
	}
}

func TestVAD_Flush(t *testing.T) {
	cfg := audio.VADConfig{
		Threshold:     0.02,
		SilenceMs:     10000, // Very long silence timeout — won't auto-emit.
		MaxSegmentSec: 30,
		SampleRate:    16000,
	}
	vad := audio.NewVAD(cfg)

	// Feed some speech.
	for i := 0; i < 5; i++ {
		vad.Process(speech(512))
	}

	// Flush should return accumulated samples.
	result := vad.Flush()
	if result == nil {
		t.Fatal("Flush() should return accumulated speech")
	}
	if len(result) != 5*512 {
		t.Errorf("expected %d samples, got %d", 5*512, len(result))
	}

	// Second flush returns nothing.
	if vad.Flush() != nil {
		t.Error("second Flush() should return nil after reset")
	}
}

func TestVAD_Reset(t *testing.T) {
	cfg := audio.DefaultVADConfig()
	vad := audio.NewVAD(cfg)

	for i := 0; i < 5; i++ {
		vad.Process(speech(512))
	}
	if !vad.IsSpeaking() {
		t.Error("expected IsSpeaking() true after speech frames")
	}

	vad.Reset()
	if vad.IsSpeaking() {
		t.Error("expected IsSpeaking() false after Reset()")
	}
	if vad.Flush() != nil {
		t.Error("expected nil Flush() after Reset()")
	}
}

func TestVAD_AccumulatedDuration(t *testing.T) {
	cfg := audio.VADConfig{
		Threshold:     0.02,
		SilenceMs:     10000,
		MaxSegmentSec: 30,
		SampleRate:    16000,
	}
	vad := audio.NewVAD(cfg)

	// Feed exactly 16000 samples = 1 second.
	frames := 16000 / 512
	for i := 0; i < frames; i++ {
		vad.Process(speech(512))
	}

	d := vad.AccumulatedDuration()
	if d < 900_000_000 || d > 1_100_000_000 { // ~1 second ± 100ms
		t.Errorf("expected ~1s duration, got %v", d)
	}
}
