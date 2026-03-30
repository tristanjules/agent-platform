//go:build !noaudio

package audio

import (
	"context"
	"fmt"

	"github.com/gen2brain/malgo"
)

// AudioPlayback writes PCM audio to the default output device.
type AudioPlayback interface {
	// Play reads int16 PCM chunks from the channel and writes them to the
	// speaker. Blocks until the channel is closed or ctx is cancelled.
	Play(ctx context.Context, audio <-chan []int16) error
	Close() error
}

// PlaybackConfig controls speaker output parameters.
type PlaybackConfig struct {
	SampleRate uint32
	Channels   uint32
	DeviceID   string
}

// DefaultPlaybackConfig returns defaults compatible with Piper TTS output.
func DefaultPlaybackConfig() PlaybackConfig {
	return PlaybackConfig{
		SampleRate: 22050, // Piper default output rate
		Channels:   1,
	}
}

type malgoPlayback struct {
	ctx *malgo.AllocatedContext
	cfg PlaybackConfig
}

// NewAudioPlayback creates a malgo-backed speaker output.
func NewAudioPlayback(cfg PlaybackConfig) (AudioPlayback, error) {
	mCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {})
	if err != nil {
		return nil, fmt.Errorf("initializing malgo context: %w", err)
	}
	return &malgoPlayback{ctx: mCtx, cfg: cfg}, nil
}

func (p *malgoPlayback) Play(ctx context.Context, audio <-chan []int16) error {
	// Collect all audio first (Piper provides it in chunks via channel).
	// For Phase 2 we buffer the full response then play it.
	// Phase 3+ can switch to true streaming playback.
	var allSamples []int16
	for {
		select {
		case chunk, ok := <-audio:
			if !ok {
				goto play
			}
			allSamples = append(allSamples, chunk...)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

play:
	if len(allSamples) == 0 {
		return nil
	}

	// Convert int16 to bytes for malgo.
	raw := int16SliceToBytes(allSamples)

	deviceCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceCfg.Playback.Format = malgo.FormatS16
	deviceCfg.Playback.Channels = p.cfg.Channels
	deviceCfg.SampleRate = p.cfg.SampleRate

	pos := 0
	onSendFrames := func(pSample2, pSample []byte, frameCount uint32) {
		bytesPerFrame := int(p.cfg.Channels * 2) // int16 = 2 bytes
		n := int(frameCount) * bytesPerFrame
		if pos+n > len(raw) {
			n = len(raw) - pos
		}
		if n <= 0 {
			return
		}
		copy(pSample, raw[pos:pos+n])
		pos += n
	}

	device, err := malgo.InitDevice(p.ctx.Context, deviceCfg, malgo.DeviceCallbacks{Data: onSendFrames})
	if err != nil {
		return fmt.Errorf("initializing playback device: %w", err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		return fmt.Errorf("starting playback device: %w", err)
	}

	// Wait until all samples are played or context cancelled.
	for pos < len(raw) {
		select {
		case <-ctx.Done():
			device.Stop()
			return ctx.Err()
		default:
			// Busy-wait in small increments — malgo drives playback via callbacks.
			// A channel-based approach would require malgo's async API.
		}
	}
	device.Stop()
	return nil
}

func (p *malgoPlayback) Close() error {
	return p.ctx.Uninit()
}

func int16SliceToBytes(s []int16) []byte {
	b := make([]byte, len(s)*2)
	for i, v := range s {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return b
}
