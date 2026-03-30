## ADDED Requirements

### Requirement: AudioPlayback consumes a channel of int16 PCM chunks
`audio.AudioPlayback.Play(ctx context.Context, audio <-chan []int16) error` SHALL read int16 PCM chunks from the channel and write them to the audio output device. It SHALL return when the channel is closed or ctx is cancelled.

#### Scenario: Play returns nil when channel closes normally
- **WHEN** the audio channel is closed after all chunks are sent
- **THEN** Play returns nil

#### Scenario: Play returns on context cancellation
- **WHEN** ctx is cancelled while Play is blocking on the audio channel
- **THEN** Play returns promptly with ctx.Err()

### Requirement: AudioPlayback stub drains the channel to prevent goroutine leaks
When compiled with the `noaudio` build tag, the stub `Play` implementation SHALL drain the audio channel using a for/range loop before returning `ErrAudioNotAvailable`.

#### Scenario: Stub prevents goroutine leak
- **WHEN** the stub Play is called with a channel that has pending items
- **THEN** all items are consumed and the goroutine that called Play exits

### Requirement: PlaybackConfig defaults match Piper's output sample rate
`audio.DefaultPlaybackConfig()` SHALL return `SampleRate=22050`, `Channels=1` to match the default output rate of the Piper TTS binary.

#### Scenario: Default playback config has Piper-compatible sample rate
- **WHEN** `DefaultPlaybackConfig()` is called
- **THEN** `SampleRate == 22050` and `Channels == 1`
