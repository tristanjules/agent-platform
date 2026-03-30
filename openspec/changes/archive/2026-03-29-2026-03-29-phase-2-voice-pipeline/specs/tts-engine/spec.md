## ADDED Requirements

### Requirement: TTSEngine synthesizes text to a channel of int16 PCM chunks
`tts.Engine.Synthesize(ctx context.Context, text string, voice VoicePersona) (<-chan []int16, error)` SHALL split text into sentences and synthesize each sentence using the Piper TTS binary. Audio chunks SHALL be streamed to the returned channel. The channel SHALL be closed when all sentences are synthesized or ctx is cancelled.

#### Scenario: Synthesize returns a closed channel for each sentence
- **WHEN** text containing a sentence-ending character is passed
- **THEN** at least one chunk of int16 samples is sent to the returned channel before it closes

#### Scenario: Context cancellation stops synthesis between sentences
- **WHEN** ctx is cancelled mid-synthesis
- **THEN** no further sentences are synthesized and the channel is closed promptly

### Requirement: splitSentences splits text on .!?\n boundaries
`splitSentences(text string) []string` SHALL split on runs of `.`, `!`, `?`, or `\n` characters, returning each fragment (including its trailing punctuation) trimmed of leading/trailing whitespace.

#### Scenario: Two-sentence text produces two entries
- **WHEN** `splitSentences("Hello. World.")` is called
- **THEN** `["Hello.", "World."]` is returned

#### Scenario: Ellipsis is treated as a single boundary
- **WHEN** `splitSentences("Hmm... interesting.")` is called
- **THEN** `["Hmm...", "interesting."]` is returned

#### Scenario: Empty string returns nil
- **WHEN** `splitSentences("")` is called
- **THEN** nil is returned

#### Scenario: Text without punctuation returns the whole string
- **WHEN** `splitSentences("Hello world")` is called
- **THEN** `["Hello world"]` is returned

### Requirement: Speaking rate maps to Piper --length_scale
When `VoicePersona.SpeakingRate != 1.0`, `buildPiperArgs` SHALL append `--length_scale <1.0/SpeakingRate>` (3 decimal places) to the argument list.

#### Scenario: Rate 0.9 produces correct length_scale
- **WHEN** `buildPiperArgs` is called with `SpeakingRate=0.9`
- **THEN** `--length_scale 1.111` appears in the returned args

#### Scenario: Rate 1.0 omits --length_scale
- **WHEN** `buildPiperArgs` is called with `SpeakingRate=1.0`
- **THEN** `--length_scale` does NOT appear in the returned args

### Requirement: newPiperEngine returns a descriptive error if the binary is not found
`newPiperEngine` SHALL use `exec.LookPath` to resolve the binary. If not found, it SHALL return an error message including the binary name and instructions to run `make download-piper`.

#### Scenario: Missing binary produces actionable error
- **WHEN** the configured piper binary is not on PATH and `newPiperEngine` is called
- **THEN** the error message references the binary name and instructs the user to run `make download-piper`

### Requirement: PersonaRegistry maps config voice names to VoicePersona structs
`tts.NewPersonaRegistry(cfg TTSConfig) *PersonaRegistry` SHALL build a registry from `cfg.Voices`. `Get(name)` SHALL return the named persona or fall back to the default voice. Zero `SpeakingRate` values SHALL be normalised to 1.0.

#### Scenario: Known voice is returned directly
- **WHEN** `Get("philosopher")` is called and "philosopher" is in the registry
- **THEN** the correct `VoicePersona` is returned with no error

#### Scenario: Unknown voice falls back to default
- **WHEN** `Get("oracle")` is called and "oracle" is not in the registry
- **THEN** the default voice persona is returned without error

#### Scenario: Missing both name and default returns error
- **WHEN** the registry is empty and `Get("any")` is called
- **THEN** an error is returned

#### Scenario: List returns alphabetically sorted names
- **WHEN** `List()` is called on a registry with multiple voices
- **THEN** the names are returned in ascending alphabetical order

#### Scenario: Zero SpeakingRate is normalised to 1.0
- **WHEN** a voice config has `SpeakingRate == 0`
- **THEN** the resulting `VoicePersona.SpeakingRate == 1.0`
