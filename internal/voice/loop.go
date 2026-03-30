// Package voice orchestrates the DUSTY voice pipeline:
//
//	AudioCapture → VAD → STT → Agent.Chat → TTS → AudioPlayback
//
// All stages are connected by typed channels and run as goroutines.
// Cancelling the context passed to Run tears the entire pipeline down cleanly.
package voice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tristanj/dusty/internal/audio"
	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/stt"
	"github.com/tristanj/dusty/internal/tts"
)

// Chatter is the interface the voice loop uses to communicate with the LLM.
// *agent.Agent satisfies this interface.
type Chatter interface {
	Chat(ctx context.Context, userMessage string) (<-chan string, error)
}

// Config bundles all dependencies for the VoiceLoop.
type Config struct {
	AppConfig config.Config
	Agent     Chatter
	Capture   audio.AudioCapture
	Playback  audio.AudioPlayback
	STT       stt.Engine
	TTS       tts.Engine
	Voice     tts.VoicePersona
	Log       *slog.Logger
}

// VoiceLoop wires together all voice pipeline stages.
type VoiceLoop struct {
	cfg     config.Config
	agent   Chatter
	capture audio.AudioCapture
	play    audio.AudioPlayback
	vad     *audio.VAD
	stt     stt.Engine
	tts     tts.Engine
	voice   tts.VoicePersona
	log     *slog.Logger
}

// New creates a VoiceLoop from the given Config. It initialises the VAD from
// the AudioConfig section but does not start any goroutines.
func New(cfg Config) *VoiceLoop {
	vadCfg := audio.VADConfig{
		Threshold:     cfg.AppConfig.Audio.VADThreshold,
		SilenceMs:     cfg.AppConfig.Audio.SilenceMs,
		MaxSegmentSec: cfg.AppConfig.Audio.MaxSegmentSec,
		SampleRate:    cfg.AppConfig.Audio.SampleRate,
	}
	return &VoiceLoop{
		cfg:     cfg.AppConfig,
		agent:   cfg.Agent,
		capture: cfg.Capture,
		play:    cfg.Playback,
		vad:     audio.NewVAD(vadCfg),
		stt:     cfg.STT,
		tts:     cfg.TTS,
		voice:   cfg.Voice,
		log:     cfg.Log,
	}
}

// Run launches the six pipeline goroutines and blocks until ctx is cancelled.
// It returns ctx.Err() on clean shutdown or a wrapped error on fatal failure.
//
// Channel buffers (from the architecture plan):
//
//	audioRaw  32 — absorbs malgo callback jitter
//	vadOut     4 — max queued utterances before backpressure
//	sttOut     4 — whisper is slow; queue prevents dropped speech
//	ttsIn      8 — absorbs burst sentences at response start
//	playIn    64 — allows piper to run ahead of playback
func (vl *VoiceLoop) Run(ctx context.Context) error {
	audioRaw := make(chan []float32, 32)
	vadOut := make(chan []float32, 4)
	sttOut := make(chan string, 4)
	ttsIn := make(chan string, 8)
	playIn := make(chan []int16, 64)

	go vl.captureGoroutine(ctx, audioRaw)
	go vl.vadGoroutine(ctx, audioRaw, vadOut)
	go vl.sttGoroutine(ctx, vadOut, sttOut)
	go vl.agentGoroutine(ctx, sttOut, ttsIn)
	go vl.ttsGoroutine(ctx, ttsIn, playIn)
	go vl.playbackGoroutine(ctx, playIn)

	<-ctx.Done()
	return ctx.Err()
}

// ---------------------------------------------------------------------------
// Stage goroutines
// ---------------------------------------------------------------------------

// captureGoroutine forwards frames from AudioCapture to audioRaw.
// It drops frames (with a debug log) if audioRaw is full to prevent the
// audio callback from blocking.
func (vl *VoiceLoop) captureGoroutine(ctx context.Context, out chan<- []float32) {
	frames, err := vl.capture.Start(ctx)
	if err != nil {
		vl.log.Error("audio capture failed to start", "err", err)
		return
	}
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				return
			}
			select {
			case out <- frame:
			case <-ctx.Done():
				return
			default:
				vl.log.Debug("audio buffer full, dropping frame")
			}
		case <-ctx.Done():
			return
		}
	}
}

// vadGoroutine feeds frames into the VAD and emits complete utterances to out.
// On context cancellation any accumulated speech is flushed before exiting.
func (vl *VoiceLoop) vadGoroutine(ctx context.Context, in <-chan []float32, out chan<- []float32) {
	defer func() {
		if seg := vl.vad.Flush(); seg != nil {
			select {
			case out <- seg:
			default: // don't block on shutdown
			}
		}
	}()

	for {
		select {
		case frame, ok := <-in:
			if !ok {
				return
			}
			if seg, emitted := vl.vad.Process(frame); emitted {
				select {
				case out <- seg:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// sttGoroutine transcribes PCM utterances using the STT engine.
func (vl *VoiceLoop) sttGoroutine(ctx context.Context, in <-chan []float32, out chan<- string) {
	for {
		select {
		case pcm, ok := <-in:
			if !ok {
				return
			}
			result, err := vl.stt.Transcribe(ctx, pcm)
			if err != nil {
				vl.log.Error("STT transcription failed", "err", err)
				continue
			}
			text := strings.TrimSpace(result.Text)
			if text == "" {
				continue
			}
			vl.log.Info("transcribed utterance", "text", text, "duration", result.Duration)
			select {
			case out <- text:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// agentGoroutine calls Agent.Chat for each transcribed utterance and forwards
// LLM tokens to ttsIn sentence-by-sentence for low-latency playback.
func (vl *VoiceLoop) agentGoroutine(ctx context.Context, in <-chan string, out chan<- string) {
	for {
		select {
		case text, ok := <-in:
			if !ok {
				return
			}
			tokens, err := vl.agent.Chat(ctx, text)
			if err != nil {
				vl.log.Error("agent chat failed", "err", err)
				continue
			}
			vl.streamTokensToTTS(ctx, tokens, out)
		case <-ctx.Done():
			return
		}
	}
}

// streamTokensToTTS drains a token channel, splitting on sentence boundaries
// and forwarding each sentence to out as soon as it is complete.
// This provides first-sentence playback latency of ~1 s rather than waiting
// for the full LLM response.
func (vl *VoiceLoop) streamTokensToTTS(ctx context.Context, tokens <-chan string, out chan<- string) {
	var buf strings.Builder
	for {
		select {
		case token, ok := <-tokens:
			if !ok {
				// End of LLM stream — flush any remaining partial sentence.
				if s := strings.TrimSpace(buf.String()); s != "" {
					select {
					case out <- s:
					case <-ctx.Done():
					}
				}
				return
			}
			buf.WriteString(token)
			current := buf.String()
			if idx := firstSentenceEnd(current); idx >= 0 {
				sentence := strings.TrimSpace(current[:idx+1])
				rest := current[idx+1:]
				if sentence != "" {
					select {
					case out <- sentence:
					case <-ctx.Done():
						return
					}
				}
				buf.Reset()
				buf.WriteString(rest)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ttsGoroutine synthesizes each sentence and forwards raw audio chunks to playIn.
func (vl *VoiceLoop) ttsGoroutine(ctx context.Context, in <-chan string, out chan<- []int16) {
	for {
		select {
		case sentence, ok := <-in:
			if !ok {
				return
			}
			chunks, err := vl.tts.Synthesize(ctx, sentence, vl.voice)
			if err != nil {
				vl.log.Error("TTS synthesis failed", "err", err)
				continue
			}
			for chunk := range chunks {
				select {
				case out <- chunk:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// playbackGoroutine streams audio chunks from playIn to the AudioPlayback device.
func (vl *VoiceLoop) playbackGoroutine(ctx context.Context, in <-chan []int16) {
	if err := vl.play.Play(ctx, in); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		vl.log.Error("playback error", "err", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// NewFromAppConfig constructs a fully-wired VoiceLoop from the application
// config, creating all audio, STT, and TTS components.
//
// Returns an error if any component fails to initialise (e.g. when built with
// the noaudio/nostt/notts build tags). The error message tells the user which
// tag to drop to enable that capability.
func NewFromAppConfig(cfg *config.Config, chatter Chatter, log *slog.Logger) (*VoiceLoop, error) {
	capCfg := audio.CaptureConfig{
		SampleRate: uint32(cfg.Audio.SampleRate),
		Channels:   uint32(cfg.Audio.Channels),
		FrameSize:  512,
	}
	cap, err := audio.NewAudioCapture(capCfg)
	if err != nil {
		return nil, fmt.Errorf("audio capture: %w", err)
	}

	play, err := audio.NewAudioPlayback(audio.DefaultPlaybackConfig())
	if err != nil {
		return nil, fmt.Errorf("audio playback: %w", err)
	}

	sttEngine, err := stt.NewEngine(cfg.STT)
	if err != nil {
		return nil, fmt.Errorf("STT engine: %w", err)
	}

	ttsEngine, err := tts.NewEngine(cfg.TTS)
	if err != nil {
		return nil, fmt.Errorf("TTS engine: %w", err)
	}

	reg := tts.NewPersonaRegistry(cfg.TTS)
	persona, err := reg.Default()
	if err != nil {
		// No voices configured — create a bare-minimum placeholder.
		persona = tts.VoicePersona{Name: "default", SpeakingRate: 1.0}
	}

	return New(Config{
		AppConfig: *cfg,
		Agent:     chatter,
		Capture:   cap,
		Playback:  play,
		STT:       sttEngine,
		TTS:       ttsEngine,
		Voice:     persona,
		Log:       log,
	}), nil
}

// firstSentenceEnd returns the index of the first sentence-ending character
// (.!?\n) in s, or -1 if none is found.
func firstSentenceEnd(s string) int {
	for i, c := range s {
		if c == '.' || c == '!' || c == '?' || c == '\n' {
			return i
		}
	}
	return -1
}
