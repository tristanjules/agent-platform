# audio-capture Specification

## Purpose
TBD - created by archiving change 2026-03-29-phase-2-voice-pipeline. Update Purpose after archive.
## Requirements
### Requirement: AudioCapture exposes a channel-based PCM frame stream
`audio.AudioCapture.Start(ctx context.Context) (<-chan []float32, error)` SHALL return a channel of 16 kHz mono float32 PCM frames. The channel SHALL be closed when ctx is cancelled or a fatal device error occurs.

#### Scenario: Start returns a readable channel
- **WHEN** `Start(ctx)` is called on a valid capture device
- **THEN** a non-nil channel is returned and frames can be read from it

#### Scenario: Channel closes on context cancellation
- **WHEN** the context passed to Start is cancelled
- **THEN** the returned channel is closed

### Requirement: AudioCapture drops frames on buffer overflow rather than blocking
The implementation SHALL use a non-blocking send to the internal frame channel. If the consumer is behind and the buffer is full, the incoming frame SHALL be silently dropped.

#### Scenario: Slow consumer does not block the audio callback
- **WHEN** the frame channel buffer is full and the audio callback fires
- **THEN** the callback returns immediately (frame dropped) and does not stall the audio device

### Requirement: AudioCapture stub returns ErrAudioNotAvailable
When compiled with the `noaudio` build tag, `NewAudioCapture` SHALL return a stub whose `Start` returns `nil, ErrAudioNotAvailable`.

#### Scenario: Stub build fails gracefully
- **WHEN** the binary is built with `-tags noaudio` and `Start` is called
- **THEN** the error `ErrAudioNotAvailable` is returned

### Requirement: CaptureConfig defaults match whisper.cpp requirements
`audio.DefaultCaptureConfig()` SHALL return `SampleRate=16000`, `Channels=1`, `FrameSize=512`.

#### Scenario: Default config has correct sample rate
- **WHEN** `DefaultCaptureConfig()` is called
- **THEN** `SampleRate == 16000`, `Channels == 1`, and `FrameSize == 512`

