//go:build notts

package tts

import "github.com/tristanj/dusty/internal/config"

func newPiperEngine(_ config.TTSConfig) (Engine, error) {
	return nil, ErrTTSNotAvailable
}
