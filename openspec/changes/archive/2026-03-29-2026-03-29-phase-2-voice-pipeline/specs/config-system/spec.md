## ADDED Requirements

### Requirement: AudioConfig captures all voice pipeline tuning parameters
The `Config` struct SHALL include an `Audio AudioConfig` field with sub-fields `SampleRate int`, `Channels int`, `InputDevice string`, `OutputDevice string`, `VADThreshold float32`, `SilenceMs int`, and `MaxSegmentSec int`.

#### Scenario: Audio defaults are correct for indoor quiet use
- **WHEN** `config.Defaults()` is called
- **THEN** `cfg.Audio.SampleRate == 16000`, `cfg.Audio.Channels == 1`, `cfg.Audio.VADThreshold == 0.02`, `cfg.Audio.SilenceMs == 800`, `cfg.Audio.MaxSegmentSec == 30`

#### Scenario: Audio section in TOML overrides defaults
- **WHEN** a TOML file contains `[audio] vad_threshold = 0.025 silence_ms = 1000`
- **THEN** the loaded config has `VADThreshold == 0.025` and `SilenceMs == 1000`

### Requirement: TTSConfig exposes the Piper binary path
`TTSConfig` SHALL include a `PiperBin string` field (`toml:"piper_bin"`) defaulting to `"piper"`.

#### Scenario: PiperBin default is "piper"
- **WHEN** `config.Defaults()` is called
- **THEN** `cfg.TTS.PiperBin == "piper"`

#### Scenario: PiperBin can be overridden to a relative path
- **WHEN** a TOML file contains `[tts] piper_bin = "bin/piper"`
- **THEN** `cfg.TTS.PiperBin == "bin/piper"`
