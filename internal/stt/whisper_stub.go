//go:build nostt

package stt

import (
	"context"

	"github.com/tristanj/dusty/internal/config"
)

type whisperEngine struct{}

func newWhisperEngine(_ config.STTConfig) (Engine, error) {
	return &whisperEngine{}, nil
}

func (w *whisperEngine) Transcribe(_ context.Context, _ []float32) (STTResult, error) {
	return STTResult{}, ErrSTTNotAvailable
}

func (w *whisperEngine) Close() error { return nil }
