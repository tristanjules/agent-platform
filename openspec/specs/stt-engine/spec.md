# stt-engine Specification

## Purpose
TBD - created by archiving change 2026-03-29-phase-2-voice-pipeline. Update Purpose after archive.
## Requirements
### Requirement: STTEngine transcribes 16 kHz mono float32 PCM to text
`stt.Engine.Transcribe(ctx context.Context, pcm []float32) (STTResult, error)` SHALL convert a complete VAD-segmented utterance to text. The entire segment is passed at once (batch mode, not streaming).

#### Scenario: Non-empty audio produces a text result
- **WHEN** a valid PCM segment containing speech is passed to Transcribe
- **THEN** `STTResult.Text` is a non-empty string and `STTResult.Language` matches the configured language

#### Scenario: Empty audio returns empty result without error
- **WHEN** `Transcribe` is called with an empty slice
- **THEN** it returns an empty `STTResult` and nil error

#### Scenario: Cancelled context is respected
- **WHEN** a cancelled context is passed to Transcribe
- **THEN** it returns immediately with the context's error

### Requirement: STTEngine.Close frees the loaded model
`stt.Engine.Close() error` SHALL release all C memory allocated for the whisper context and GGML model. After Close, the engine SHALL NOT be used for further transcription.

#### Scenario: Close can be called safely
- **WHEN** `Close()` is called on a loaded engine
- **THEN** it returns nil and no memory is leaked

### Requirement: Model path is resolved from short name or explicit path
`resolveModelPath(model string) string` SHALL map short names like `"base"` or `"tiny"` to `"assets/models/ggml-<name>.en.bin"`. Strings already containing a path separator SHALL be returned unchanged.

#### Scenario: Short name resolution
- **WHEN** the config model is `"base"`
- **THEN** the resolved path is `"assets/models/ggml-base.en.bin"`

#### Scenario: Explicit path passthrough
- **WHEN** the config model is `"/data/models/my-model.bin"`
- **THEN** the resolved path is `"/data/models/my-model.bin"` unchanged

### Requirement: STT stub returns ErrSTTNotAvailable
When compiled with the `nostt` build tag, `stt.NewEngine` SHALL return a stub whose `Transcribe` returns `ErrSTTNotAvailable` with a message directing the user to rebuild without the tag.

#### Scenario: Stub Transcribe returns the correct error
- **WHEN** `Transcribe` is called on the nostt stub
- **THEN** the returned error is `ErrSTTNotAvailable`

