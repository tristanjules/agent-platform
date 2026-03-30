//go:build !noaudio

// Package audio provides microphone capture, speaker playback, and voice
// activity detection for the DUSTY voice pipeline.
package audio

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// AudioCapture captures PCM audio from the default input device.
// Samples are 16kHz mono float32, matching whisper.cpp's expected format.
type AudioCapture interface {
	// Start begins capturing audio. Returns a channel of PCM frame slices.
	// The channel is closed when ctx is cancelled or a fatal error occurs.
	Start(ctx context.Context) (<-chan []float32, error)
	// Close releases the audio device.
	Close() error
}

type malgoCapture struct {
	ctx     *malgo.AllocatedContext
	device  *malgo.Device
	frameCh chan []float32
	cfg     CaptureConfig
}

// CaptureConfig controls microphone capture parameters.
type CaptureConfig struct {
	SampleRate  uint32
	Channels    uint32
	FrameSize   uint32 // samples per callback
	DeviceID    string // "" = system default
}

// DefaultCaptureConfig returns sensible defaults for whisper.cpp compatibility.
func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{
		SampleRate: 16000,
		Channels:   1,
		FrameSize:  512,
	}
}

// NewAudioCapture creates a malgo-backed microphone capture.
func NewAudioCapture(cfg CaptureConfig) (AudioCapture, error) {
	mCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {})
	if err != nil {
		return nil, fmt.Errorf("initializing malgo context: %w", err)
	}

	return &malgoCapture{
		ctx:     mCtx,
		frameCh: make(chan []float32, 32),
		cfg:     cfg,
	}, nil
}

func (c *malgoCapture) Start(ctx context.Context) (<-chan []float32, error) {
	deviceCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceCfg.Capture.Format = malgo.FormatF32
	deviceCfg.Capture.Channels = c.cfg.Channels
	deviceCfg.SampleRate = c.cfg.SampleRate
	deviceCfg.Alsa.NoMMap = 1

	onRecvFrames := func(pSample2, pSample []byte, frameCount uint32) {
		// Convert raw bytes to float32 slice.
		samples := bytesToFloat32(pSample, frameCount*c.cfg.Channels)
		// Non-blocking send — drop if consumer is behind.
		select {
		case c.frameCh <- samples:
		default:
		}
	}

	callbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(c.ctx.Context, deviceCfg, callbacks)
	if err != nil {
		return nil, fmt.Errorf("initializing capture device: %w", err)
	}
	c.device = device

	if err := device.Start(); err != nil {
		return nil, fmt.Errorf("starting capture device: %w", err)
	}

	// Stop device when context is cancelled.
	go func() {
		<-ctx.Done()
		device.Stop()
		close(c.frameCh)
	}()

	return c.frameCh, nil
}

func (c *malgoCapture) Close() error {
	if c.device != nil {
		c.device.Uninit()
	}
	return c.ctx.Uninit()
}

// bytesToFloat32 reinterprets a little-endian float32 byte slice.
func bytesToFloat32(b []byte, count uint32) []float32 {
	out := make([]float32, count)
	for i := uint32(0); i < count; i++ {
		offset := i * 4
		if offset+4 > uint32(len(b)) {
			break
		}
		bits := uint32(b[offset]) |
			uint32(b[offset+1])<<8 |
			uint32(b[offset+2])<<16 |
			uint32(b[offset+3])<<24
		// Reinterpret IEEE 754 bits as float32.
		out[i] = *(*float32)(unsafe.Pointer(&bits))
	}
	return out
}
