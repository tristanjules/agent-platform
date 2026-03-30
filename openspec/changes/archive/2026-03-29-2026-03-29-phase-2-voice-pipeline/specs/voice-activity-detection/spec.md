## ADDED Requirements

### Requirement: RMSEnergy returns a normalized energy value for a PCM frame
`audio.RMSEnergy(samples []float32) float32` SHALL compute the root mean square of the sample values, returning 0.0 for an empty slice.

#### Scenario: Silence produces near-zero energy
- **WHEN** `RMSEnergy` is called with a frame of all-zero samples
- **THEN** the result is less than 0.001

#### Scenario: Loud audio produces high energy
- **WHEN** `RMSEnergy` is called with samples all set to 0.5
- **THEN** the result is between 0.4 and 1.0

#### Scenario: Empty input returns zero
- **WHEN** `RMSEnergy([]float32{})` is called
- **THEN** the result is 0.0

### Requirement: IsSpeech classifies a frame as speech or silence
`audio.IsSpeech(samples []float32, threshold float32) bool` SHALL return true if and only if `RMSEnergy(samples) > threshold`.

#### Scenario: Silence is not classified as speech
- **WHEN** `IsSpeech` is called with a zero-energy frame and threshold 0.02
- **THEN** the result is false

#### Scenario: Loud audio is classified as speech
- **WHEN** `IsSpeech` is called with a frame of samples at 0.5 amplitude and threshold 0.02
- **THEN** the result is true

### Requirement: VAD accumulates speech and emits complete utterances on silence timeout
`audio.VAD.Process(frames []float32) ([]float32, bool)` SHALL accumulate frames while speech is detected. Once silence is detected for `SilenceMs` milliseconds, it SHALL emit all accumulated frames and reset.

#### Scenario: Speech frames alone do not trigger emission
- **WHEN** five speech frames are processed
- **THEN** no segment is emitted (returns nil, false)

#### Scenario: Silence after speech triggers emission
- **WHEN** speech frames are followed by enough silence frames to exceed the silence timeout
- **THEN** a non-nil segment containing the speech samples is emitted

#### Scenario: Pure silence never triggers emission
- **WHEN** only silence frames are processed (no preceding speech)
- **THEN** no segment is ever emitted regardless of frame count

### Requirement: VAD emits a segment when max segment length is reached
If `totalSamples >= MaxSegmentSec * SampleRate` while accumulating, `Process` SHALL emit immediately without waiting for a silence timeout.

#### Scenario: Max-length segment is emitted mid-speech
- **WHEN** the accumulated samples reach or exceed `MaxSegmentSec * SampleRate`
- **THEN** a segment is emitted even if speech is still ongoing

### Requirement: Flush returns any accumulated speech and resets state
`VAD.Flush() []float32` SHALL return accumulated samples if the VAD is currently speaking, then reset all state. It SHALL return nil if no speech has been accumulated.

#### Scenario: Flush after speech returns samples
- **WHEN** five speech frames are processed and Flush is called before the silence timeout
- **THEN** Flush returns a slice with exactly 5 * frameSize samples

#### Scenario: Second Flush returns nil
- **WHEN** Flush is called twice consecutively
- **THEN** the second call returns nil

### Requirement: Reset clears all VAD state
`VAD.Reset()` SHALL set `speaking = false`, clear accumulated samples, and reset frame counters. Subsequent `IsSpeaking()` calls SHALL return false and `Flush()` SHALL return nil.

#### Scenario: IsSpeaking is false after Reset
- **WHEN** five speech frames are processed and then Reset is called
- **THEN** `IsSpeaking()` returns false and `Flush()` returns nil

### Requirement: AccumulatedDuration reflects current buffer size
`VAD.AccumulatedDuration() time.Duration` SHALL return the duration of accumulated samples as `totalSamples / SampleRate` seconds, with zero returned if `SampleRate == 0`.

#### Scenario: Duration matches sample count
- **WHEN** exactly `SampleRate` samples (1 second at 16000 Hz) are accumulated
- **THEN** `AccumulatedDuration()` returns a value within 100 ms of 1 second
