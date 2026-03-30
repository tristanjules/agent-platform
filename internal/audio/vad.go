// Package audio — voice activity detection.
// No CGo required; this is pure float32 signal processing.
package audio

import (
	"math"
	"time"
)

// VADConfig controls voice activity detection parameters.
type VADConfig struct {
	// Threshold is the RMS energy level above which frames are considered speech.
	// Range: 0.0–1.0 for float32 PCM. A value of 0.02 works well for a quiet room.
	Threshold float32
	// SilenceMs is the duration of silence that ends an utterance.
	SilenceMs int
	// MaxSegmentSec is the maximum utterance length. Longer audio is split here
	// to stay within whisper.cpp's 30s processing limit.
	MaxSegmentSec int
	// SampleRate is the audio sample rate (default 16000 Hz).
	SampleRate int
}

// DefaultVADConfig returns recommended defaults for a quiet indoor environment.
func DefaultVADConfig() VADConfig {
	return VADConfig{
		Threshold:     0.02,
		SilenceMs:     800,
		MaxSegmentSec: 30,
		SampleRate:    16000,
	}
}

// RMSEnergy calculates the root mean square energy of a PCM frame.
// Returns a value in [0, 1] for normalized float32 samples.
func RMSEnergy(samples []float32) float32 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return float32(math.Sqrt(sum / float64(len(samples))))
}

// IsSpeech reports whether the frame's energy exceeds the given threshold.
func IsSpeech(samples []float32, threshold float32) bool {
	return RMSEnergy(samples) > threshold
}

// VAD is a stateful voice activity detector that consumes a stream of PCM
// frames and emits complete utterance segments when speech boundaries are found.
//
// Usage:
//
//	vad := NewVAD(cfg)
//	for frames := range audioFrames {
//	    if seg, ok := vad.Process(frames); ok {
//	        // seg is a complete utterance ready for STT
//	    }
//	}
//	if final := vad.Flush(); final != nil {
//	    // handle any trailing speech
//	}
type VAD struct {
	cfg VADConfig

	speaking      bool
	silentFrames  int     // consecutive silent frames counted
	accumulated   []float32 // growing buffer of the current utterance
	totalSamples  int     // total samples accumulated in current utterance
}

// NewVAD creates a voice activity detector with the given config.
func NewVAD(cfg VADConfig) *VAD {
	return &VAD{cfg: cfg}
}

// Process adds frames to the detector. Returns a non-nil slice when a complete
// utterance is detected (silence timeout or max length reached), and true.
// Returns nil, false when still accumulating.
func (v *VAD) Process(frames []float32) ([]float32, bool) {
	speech := IsSpeech(frames, v.cfg.Threshold)
	silenceTimeout := v.silenceTimeoutFrames()
	maxSamples := v.cfg.MaxSegmentSec * v.cfg.SampleRate

	if speech {
		v.speaking = true
		v.silentFrames = 0
		v.accumulated = append(v.accumulated, frames...)
		v.totalSamples += len(frames)
	} else if v.speaking {
		// Silence while we were speaking.
		v.silentFrames++
		v.accumulated = append(v.accumulated, frames...)
		v.totalSamples += len(frames)
	}

	// Emit if silence timeout reached.
	if v.speaking && v.silentFrames >= silenceTimeout {
		return v.flush(), true
	}

	// Emit if max segment length reached.
	if v.totalSamples >= maxSamples {
		return v.flush(), true
	}

	return nil, false
}

// Flush returns any accumulated speech and resets state.
// Call at the end of a recording session to retrieve any trailing utterance.
func (v *VAD) Flush() []float32 {
	if !v.speaking || len(v.accumulated) == 0 {
		return nil
	}
	return v.flush()
}

// Reset clears accumulated state without emitting.
func (v *VAD) Reset() {
	v.speaking = false
	v.silentFrames = 0
	v.accumulated = nil
	v.totalSamples = 0
}

// IsSpeaking reports whether the VAD is currently inside an utterance.
func (v *VAD) IsSpeaking() bool {
	return v.speaking
}

// AccumulatedDuration returns the duration of the current accumulated audio.
func (v *VAD) AccumulatedDuration() time.Duration {
	if v.cfg.SampleRate == 0 {
		return 0
	}
	secs := float64(v.totalSamples) / float64(v.cfg.SampleRate)
	return time.Duration(secs * float64(time.Second))
}

func (v *VAD) flush() []float32 {
	result := v.accumulated
	v.accumulated = nil
	v.speaking = false
	v.silentFrames = 0
	v.totalSamples = 0
	return result
}

// silenceTimeoutFrames converts SilenceMs to a frame count based on typical
// frame size. We use a heuristic of 512 samples per frame at 16kHz ≈ 32ms/frame.
func (v *VAD) silenceTimeoutFrames() int {
	const typicalFrameSizeMs = 32
	return v.cfg.SilenceMs / typicalFrameSizeMs
}
