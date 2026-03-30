// Package stt provides the speech-to-text interface and implementations for DUSTY.
// The primary implementation uses whisper.cpp via CGo (whisper.go).
// A stub is available for builds without CGo support (whisper_stub.go).
package stt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tristanj/dusty/internal/config"
)

// ErrSTTNotAvailable is returned when STT is invoked on a stub build.
var ErrSTTNotAvailable = errors.New("STT not available: rebuild without the 'nostt' build tag and ensure libwhisper.a is built")

// STTResult contains the output of a transcription.
type STTResult struct {
	Text     string
	Language string        // detected language code, e.g. "en"
	Duration time.Duration // duration of the audio segment processed
}

// Engine transcribes PCM audio to text.
type Engine interface {
	// Transcribe converts a complete audio segment (16kHz mono float32 PCM)
	// into text. The entire VAD-segmented utterance is passed at once.
	// whisper.cpp operates in batch mode — use VAD to segment before calling.
	Transcribe(ctx context.Context, pcm []float32) (STTResult, error)
	// Close frees the loaded model and any resources.
	Close() error
}

// NewEngine creates an STT engine from the given config.
// Returns an error if the requested engine is not available in this build.
func NewEngine(cfg config.STTConfig) (Engine, error) {
	switch cfg.Engine {
	case "whisper", "":
		return newWhisperEngine(cfg)
	default:
		return nil, fmt.Errorf("unknown STT engine: %q (supported: whisper)", cfg.Engine)
	}
}
