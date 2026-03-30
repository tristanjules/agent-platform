// Package tts provides the text-to-speech interface and implementations for DUSTY.
// The primary implementation calls the Piper TTS binary via os/exec (piper.go).
// No CGo is required — Piper ARM64 binaries are downloaded separately.
package tts

import (
	"context"
	"errors"
	"fmt"

	"github.com/tristanj/dusty/internal/config"
)

// ErrTTSNotAvailable is returned when TTS is invoked on a stub build.
var ErrTTSNotAvailable = errors.New("TTS not available: rebuild without the 'notts' build tag")

// VoicePersona maps a named persona to a Piper voice model and parameters.
type VoicePersona struct {
	Name         string
	ModelPath    string  // absolute path to .onnx file
	SpeakingRate float64 // 1.0 = normal, 0.9 = slightly slower
}

// Engine synthesizes text to PCM audio using a specified voice persona.
type Engine interface {
	// Synthesize converts text to int16 PCM chunks using the given voice.
	// Returns a channel of audio chunks. The channel is closed when synthesis
	// is complete or ctx is cancelled. Callers must drain the channel.
	Synthesize(ctx context.Context, text string, voice VoicePersona) (<-chan []int16, error)
}

// NewEngine creates a TTS engine from the given config.
func NewEngine(cfg config.TTSConfig) (Engine, error) {
	switch cfg.Engine {
	case "piper", "":
		return newPiperEngine(cfg)
	default:
		return nil, fmt.Errorf("unknown TTS engine: %q (supported: piper)", cfg.Engine)
	}
}

// NewNullEngine returns a TTS engine that silently discards all synthesis requests.
// Useful when running in text-only mode.
func NewNullEngine() Engine {
	return &nullEngine{}
}

type nullEngine struct{}

func (n *nullEngine) Synthesize(_ context.Context, _ string, _ VoicePersona) (<-chan []int16, error) {
	ch := make(chan []int16)
	close(ch)
	return ch, errors.New("null TTS engine: no audio output configured")
}
