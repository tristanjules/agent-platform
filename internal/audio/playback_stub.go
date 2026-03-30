//go:build noaudio

package audio

import "context"

type AudioPlayback interface {
	Play(ctx context.Context, audio <-chan []int16) error
	Close() error
}

type PlaybackConfig struct {
	SampleRate uint32
	Channels   uint32
	DeviceID   string
}

func DefaultPlaybackConfig() PlaybackConfig {
	return PlaybackConfig{SampleRate: 22050, Channels: 1}
}

type stubPlayback struct{}

func NewAudioPlayback(_ PlaybackConfig) (AudioPlayback, error) {
	return &stubPlayback{}, nil
}

func (s *stubPlayback) Play(_ context.Context, audio <-chan []int16) error {
	// Drain the channel to avoid goroutine leaks in tests.
	for range audio {
	}
	return ErrAudioNotAvailable
}

func (s *stubPlayback) Close() error { return nil }
