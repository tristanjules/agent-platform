## ADDED Requirements

### Requirement: VoiceLoop orchestrates the six-stage pipeline
`voice.VoiceLoop.Run(ctx context.Context) error` SHALL start six goroutines (capture, VAD, STT, agent, TTS, playback) connected by typed channels and block until ctx is cancelled.

#### Scenario: Speech utterance flows end-to-end to playback
- **WHEN** speech frames flow through the pipeline (capture → VAD → STT → agent → TTS)
- **THEN** audio chunks reach the playback stage and the agent is called with the transcribed text

#### Scenario: Pure silence never reaches the agent
- **WHEN** only silence frames are processed
- **THEN** the STT engine is never called and the agent Chat method is never invoked

#### Scenario: Context cancellation stops the loop cleanly
- **WHEN** ctx is cancelled
- **THEN** Run returns within a bounded time and no goroutines leak

### Requirement: The agent goroutine performs sentence-level streaming to TTS
As LLM tokens arrive, the agent goroutine SHALL accumulate them and forward a sentence to `ttsIn` whenever a sentence-ending character (`.?!\n`) is detected. Any remaining text after the token stream closes SHALL be flushed as a final sentence.

#### Scenario: Two sentences produce two TTS calls
- **WHEN** the LLM token stream contains two sentences separated by sentence-ending punctuation
- **THEN** the TTS engine is called at least twice

#### Scenario: First sentence is forwarded before all tokens arrive
- **WHEN** the first sentence ends mid-stream (more tokens still pending)
- **THEN** the first sentence is sent to TTS before the token stream closes

### Requirement: capture goroutine drops frames on buffer overflow
When `audioRaw` is full, the capture goroutine SHALL drop the incoming frame and log a debug message rather than blocking the audio callback.

#### Scenario: Drop on full buffer does not block
- **WHEN** the audioRaw channel buffer is at capacity and a new frame arrives
- **THEN** the frame is discarded and the goroutine continues without blocking

### Requirement: VAD goroutine flushes accumulated speech on shutdown
When ctx is cancelled while the VAD has accumulated speech, `vadGoroutine` SHALL call `vad.Flush()` and attempt to send the result downstream before exiting.

#### Scenario: In-flight utterance is not silently discarded on shutdown
- **WHEN** ctx is cancelled while the VAD has accumulated speech frames
- **THEN** vadGoroutine calls Flush and attempts to forward the partial utterance

### Requirement: Chatter interface decouples the voice loop from the Agent type
The voice loop SHALL accept a `voice.Chatter` interface (`Chat(ctx, text) (<-chan string, error)`) rather than a concrete `*agent.Agent`. Any type implementing this interface can be used, enabling mock-based testing without a real LLM.

#### Scenario: Mock Chatter works in tests
- **WHEN** a mock implementing Chatter is injected into VoiceLoop
- **THEN** the pipeline operates correctly and the mock's Chat method is called with the transcribed text

### Requirement: NewFromAppConfig constructs the full pipeline from application config
`voice.NewFromAppConfig(cfg *config.Config, chatter Chatter, log *slog.Logger) (*VoiceLoop, error)` SHALL create all pipeline components (AudioCapture, AudioPlayback, STTEngine, TTSEngine, PersonaRegistry) from the application config, returning a descriptive error if any component fails to initialise.

#### Scenario: Missing STT engine returns error
- **WHEN** the binary is built with nostt and NewFromAppConfig is called
- **THEN** it returns an error describing the missing STT engine
