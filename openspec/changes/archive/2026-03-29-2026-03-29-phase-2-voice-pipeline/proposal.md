## Why

DUSTY is a voice-first AI agent for Burning Man. Phase 1 delivered a solid text REPL, but the core user experience — speaking to DUSTY and hearing it respond — requires a complete voice pipeline. Phase 2 closes that gap: microphone input is captured, segmented by a voice activity detector, transcribed by whisper.cpp, passed through the existing LLM agent, synthesized by Piper TTS, and played back through the speaker. The state machine, event bus, and config schema scaffolded in Phase 1 are all ready to receive this pipeline — Phase 2 simply fills in the empty packages.

## What Changes

- Add `internal/audio/vad.go`: energy-based Voice Activity Detector — pure Go, no CGo, testable without hardware
- Add `internal/audio/capture.go` + stub: `AudioCapture` interface backed by `gen2brain/malgo` (miniaudio bindings); build-tagged `!noaudio` / `noaudio`
- Add `internal/audio/playback.go` + stub: `AudioPlayback` interface backed by `gen2brain/malgo`; build-tagged `!noaudio` / `noaudio`
- Add `internal/stt/engine.go` + `whisper.go` + stub: `STTEngine` interface; whisper.cpp CGo implementation build-tagged `!nostt` / `nostt`
- Add `internal/tts/engine.go` + `piper.go` + stub: `TTSEngine` interface; Piper TTS via `os/exec` build-tagged `!notts` / `notts`
- Add `internal/tts/personas.go`: `PersonaRegistry` mapping config voice names to `VoicePersona` structs with fallback
- Add `internal/voice/loop.go`: `VoiceLoop` — six-goroutine pipeline orchestrator connecting all stages; sentence-level TTS streaming for low latency
- Extend `internal/config/config.go`: add `AudioConfig` struct and `PiperBin` field to `TTSConfig`
- Extend `cmd/dusty/main.go`: add `/voice start|stop` REPL commands via `voiceState` lifecycle manager
- Extend `Makefile`: add `build-voice`, `build-pi`, `build-whisper`, `download-whisper-model`, `download-piper` targets
- Add `.vscode/settings.json`: configure gopls to use stub build tags so analysis tools work without CGo headers

## Capabilities

### New Capabilities

- `voice-activity-detection`: Stateful energy-based VAD that accumulates PCM frames and emits complete utterances on silence timeout or max-length cap; pure Go, fully unit-tested without hardware
- `audio-capture`: Microphone capture interface (16 kHz mono float32) backed by malgo; build-tagged so text-only builds incur no CGo dependency
- `audio-playback`: Speaker playback interface (22050 Hz int16) backed by malgo; same build-tag pattern as capture
- `stt-engine`: Speech-to-text interface; whisper.cpp CGo implementation with GGML model loading and full-segment transcription; graceful stub for non-CGo builds
- `tts-engine`: Text-to-speech interface; Piper TTS via `os/exec` with sentence-level streaming, `--length_scale` speaking rate control, and raw PCM output; `PersonaRegistry` maps config voice names to model paths
- `voice-loop`: Six-goroutine pipeline (capture → VAD → STT → agent → TTS → playback) connected by typed channels; sentence streaming delivers first-word playback ~1 s after utterance ends; context cancellation tears down cleanly at any sentence boundary

### Modified Capabilities

- `config-system`: Extended with `AudioConfig` (sample_rate, channels, vad_threshold, silence_ms, max_segment_sec) and `TTSConfig.PiperBin` field; existing TOML files remain valid

## Impact

- **New CGo dependency**: `github.com/gen2brain/malgo` (only `-ldl` on Linux, no extra system packages); whisper.cpp built from source via `make build-whisper`
- **New runtime dependency**: Piper TTS binary downloaded via `make download-piper`; whisper.cpp GGML model downloaded via `make download-whisper-model`
- **Build tag isolation**: Default `make build` gains `noaudio nostt notts` tags — zero CGo, zero hardware requirement; `make build-voice` enables the full pipeline
- **extern/ directory**: `extern/whisper.cpp` holds the vendored C library source (gitignored); avoids conflict with Go's `vendor/` directory
- **No breaking changes** to Phase 1 text REPL — existing `/clear`, `/persona`, `/model`, `/mode`, `/status` commands unchanged
