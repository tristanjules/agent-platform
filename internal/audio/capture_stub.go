//go:build noaudio

package audio

import (
	"context"
	"errors"
)

// ErrAudioNotAvailable is returned when the binary was built without audio support.
var ErrAudioNotAvailable = errors.New("audio not available: rebuild without the 'noaudio' build tag")

type AudioCapture interface {
	Start(ctx context.Context) (<-chan []float32, error)
	Close() error
}

type CaptureConfig struct {
	SampleRate  uint32
	Channels    uint32
	FrameSize   uint32
	DeviceID    string
}

func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{SampleRate: 16000, Channels: 1, FrameSize: 512}
}

type stubCapture struct{}

func NewAudioCapture(_ CaptureConfig) (AudioCapture, error) {
	return &stubCapture{}, nil
}

func (s *stubCapture) Start(_ context.Context) (<-chan []float32, error) {
	return nil, ErrAudioNotAvailable
}

func (s *stubCapture) Close() error { return nil }
