## 1. Interfaces and Stubs

- [x] 1.1 Define `AudioCapture` interface (`Start(ctx) (<-chan []float32, error)`, `Close() error`) in `internal/audio/capture.go` with `//go:build !noaudio`
- [x] 1.2 Write `internal/audio/capture_stub.go` (`//go:build noaudio`) returning `ErrAudioNotAvailable`
- [x] 1.3 Define `AudioPlayback` interface (`Play(ctx, <-chan []int16) error`, `Close() error`) in `internal/audio/playback.go` with `//go:build !noaudio`
- [x] 1.4 Write `internal/audio/playback_stub.go` (`//go:build noaudio`) draining channel and returning `ErrAudioNotAvailable`
- [x] 1.5 Define `STTEngine` interface (`Transcribe(ctx, []float32) (STTResult, error)`, `Close() error`) and `STTResult` struct in `internal/stt/engine.go`
- [x] 1.6 Write `internal/stt/whisper_stub.go` (`//go:build nostt`) returning `ErrSTTNotAvailable`
- [x] 1.7 Define `TTSEngine` interface (`Synthesize(ctx, text, VoicePersona) (<-chan []int16, error)`) and `VoicePersona` struct in `internal/tts/engine.go`
- [x] 1.8 Write `internal/tts/piper_stub.go` (`//go:build notts`) with `newPiperEngine` returning `ErrTTSNotAvailable`
- [x] 1.9 Add `internal/audio/wakeword.go` stub (`NullWakeWordDetector`) for Phase 5 placeholder
- [x] 1.10 Verify `go build -tags "noaudio nostt notts" ./...` passes with all stubs

## 2. Config Extension

- [x] 2.1 Add `AudioConfig` struct to `internal/config/config.go`: `SampleRate int`, `Channels int`, `InputDevice string`, `OutputDevice string`, `VADThreshold float32`, `SilenceMs int`, `MaxSegmentSec int`
- [x] 2.2 Add `PiperBin string` field to `TTSConfig`
- [x] 2.3 Add `Audio AudioConfig` field to top-level `Config` struct
- [x] 2.4 Set defaults in `config.Defaults()`: SampleRate=16000, Channels=1, VADThreshold=0.02, SilenceMs=800, MaxSegmentSec=30, PiperBin="piper"
- [x] 2.5 Add `[audio]` section to `configs/default.toml` and `configs/burning-man.toml`
- [x] 2.6 Update `internal/config/config_test.go` to cover new `AudioConfig` defaults and `PiperBin`

## 3. Voice Activity Detection

- [x] 3.1 Implement `RMSEnergy(samples []float32) float32` in `internal/audio/vad.go`
- [x] 3.2 Implement `IsSpeech(samples []float32, threshold float32) bool`
- [x] 3.3 Define `VADConfig` struct (`Threshold`, `SilenceMs`, `MaxSegmentSec`, `SampleRate`) with `DefaultVADConfig()`
- [x] 3.4 Implement `VAD` struct: `speaking bool`, `silentFrames int`, `accumulated []float32`, `totalSamples int`
- [x] 3.5 Implement `VAD.Process(frames []float32) ([]float32, bool)` — accumulate speech, emit on silence timeout or max-length
- [x] 3.6 Implement `VAD.Flush() []float32` — emit any accumulated speech, reset state
- [x] 3.7 Implement `VAD.Reset()`, `VAD.IsSpeaking() bool`, `VAD.AccumulatedDuration() time.Duration`
- [x] 3.8 Write `internal/audio/vad_test.go` with 7 tests covering all methods and edge cases

## 4. Piper TTS

- [x] 4.1 Implement `splitSentences(text string) []string` in `internal/tts/piper.go` splitting on `.?!\n` runs
- [x] 4.2 Implement `buildPiperArgs(voice VoicePersona) []string` producing `--model`, `--output_raw`, and optional `--length_scale`
- [x] 4.3 Implement `rawBytesToInt16(data []byte) []int16` converting little-endian PCM bytes
- [x] 4.4 Implement `piperEngine.synthesizeSentence` via `exec.CommandContext` with stdin/stdout
- [x] 4.5 Implement `piperEngine.Synthesize` — split text, synthesize each sentence, stream chunks to returned channel
- [x] 4.6 Use `exec.LookPath` in `newPiperEngine` and return descriptive error with install instructions if binary not found
- [x] 4.7 Implement `PersonaRegistry` in `internal/tts/personas.go`: `Get(name)` with default fallback, `Default()`, `List()`
- [x] 4.8 Write `internal/tts/piper_test.go` (`//go:build !notts`) covering `splitSentences`, `buildPiperArgs`, `rawBytesToInt16`, `truncate`
- [x] 4.9 Write `internal/tts/personas_test.go` covering lookup, fallback, error on missing, zero-rate normalisation

## 5. Voice Loop

- [x] 5.1 Define `Chatter` interface (`Chat(ctx, text) (<-chan string, error)`) in `internal/voice/loop.go`
- [x] 5.2 Define `voice.Config` struct bundling all pipeline dependencies
- [x] 5.3 Implement `VoiceLoop.Run(ctx)` launching six goroutines with typed channels
- [x] 5.4 Implement `captureGoroutine` — forward frames from AudioCapture with drop-on-full backpressure
- [x] 5.5 Implement `vadGoroutine` — feed VAD, emit utterances, flush on shutdown
- [x] 5.6 Implement `sttGoroutine` — transcribe utterances, skip empty results
- [x] 5.7 Implement `agentGoroutine` — call Agent.Chat, split tokens on `.?!\n` via `streamTokensToTTS`
- [x] 5.8 Implement `ttsGoroutine` — synthesize each sentence, forward audio chunks
- [x] 5.9 Implement `playbackGoroutine` — call AudioPlayback.Play; suppress context-cancellation errors
- [x] 5.10 Implement `voice.NewFromAppConfig(cfg, chatter, log)` factory constructing all pipeline components from config
- [x] 5.11 Write `internal/voice/voice_test.go` with 4 mock-based tests: E2E speech flow, silence never reaches agent, context cancellation, sentence streaming

## 6. main.go Integration

- [x] 6.1 Add `voiceState` struct with `start(ctx, cfg, agent, log)`, `stop()`, `active()` methods and `sync.Mutex`
- [x] 6.2 Add `/voice start` command — call `voice.NewFromAppConfig`, launch `Run` goroutine with panic recovery
- [x] 6.3 Add `/voice stop` command — cancel voice context
- [x] 6.4 Add `/voice` (no subcommand) — show active/inactive status and usage
- [x] 6.5 Update `/help` output to document `/voice start|stop`
- [x] 6.6 Show "DUSTY is busy with voice mode" instead of raw error when text REPL chat conflicts with active voice

## 7. malgo Audio Implementation

- [x] 7.1 Implement `malgoCapture` in `internal/audio/capture.go` (`//go:build !noaudio`): `malgo.InitContext`, `DefaultDeviceConfig(Capture)`, `FormatF32`, callback sending to buffered channel
- [x] 7.2 Implement `bytesToFloat32` reinterpreting IEEE 754 bytes via `unsafe.Pointer`
- [x] 7.3 Implement `malgoPlayback` in `internal/audio/playback.go` (`//go:build !noaudio`): collect int16 chunks, write to playback device

## 8. whisper.cpp STT + Build System

- [x] 8.1 Implement `whisperEngine` in `internal/stt/whisper.go` (`//go:build !nostt`)
- [x] 8.2 Set `#cgo CFLAGS` to include `extern/whisper.cpp/include` and `extern/whisper.cpp/ggml/include`
- [x] 8.3 Set `#cgo LDFLAGS` for all six static libraries (`libwhisper.a`, `libggml.a`, `libggml-base.a`, `libggml-cpu.a`, `libggml-blas.a`, `libggml-metal.a`)
- [x] 8.4 Set OS-specific frameworks: `darwin` gets Accelerate, Foundation, Metal, MetalKit; `linux` gets ldl, pthread
- [x] 8.5 Use `whisper_init_from_file_with_params` + `whisper_context_default_params()` (non-deprecated API)
- [x] 8.6 Implement `resolveModelPath` mapping short names ("base", "tiny") to `assets/models/ggml-<name>.en.bin`
- [x] 8.7 Add `make build-whisper` target: cmake check, clone `extern/whisper.cpp`, cmake configure + build
- [x] 8.8 Add `make download-whisper-model` target: curl ggml-base.en.bin from HuggingFace
- [x] 8.9 Add `make download-piper` target: curl piper_linux_aarch64.tar.gz from rhasspy releases
- [x] 8.10 Add `make build-voice` (CGO_ENABLED=1, no stub tags) and `make build-pi` (cross-compile ARM64) targets
- [x] 8.11 Update `make test` to use `-tags "noaudio nostt notts"` by default; add `make test-all` without tags

## 9. Hardening and Polish

- [x] 9.1 Add `.vscode/settings.json` with `go.buildFlags` and `gopls.build.buildFlags` using stub tags
- [x] 9.2 Add `extern/` and `assets/models/` to `.gitignore`
- [x] 9.3 Suppress `context.DeadlineExceeded` in `playbackGoroutine` error log (same as Canceled)
- [x] 9.4 Add panic recovery in `voiceState.start` goroutine
- [x] 9.5 Fix `fmt.Println("\n")` redundant-newline lint error in `main.go`
- [x] 9.6 Verify `make build` and `make test` pass cleanly (7 packages, zero CGo)
- [x] 9.7 Verify `make build-voice` produces `bin/dusty-voice` (~39 MB on macOS with Metal)
