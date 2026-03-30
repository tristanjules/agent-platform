## 1. Project Scaffolding

- [x] 1.1 Run `git init` and `go mod init github.com/tristanj/dusty`
- [x] 1.2 Create full directory structure: `cmd/dusty`, `internal/{agent,audio,stt,tts,display/tui,display/visual,state,config}`, `configs`, `assets/{voices,visuals,sounds,wakewords}`
- [x] 1.3 Add `.gitignore` (binaries, model files, memory JSON, .env)
- [x] 1.4 Add `Makefile` with `build`, `run`, `test`, `cross-compile-pi`, `clean`, `fmt`, `vet` targets
- [x] 1.5 Add `configs/default.toml` with all sections defined
- [x] 1.6 Add `configs/burning-man.toml` offline-only preset
- [x] 1.7 Add `.gitkeep` files in each `assets/` subdirectory

## 2. Config System

- [x] 2.1 Define `Config` struct hierarchy in `internal/config/config.go` with TOML tags covering all 5 phases: `AgentConfig`, `InferenceConfig`, `LocalInferenceConfig`, `CloudInferenceConfig`, `STTConfig`, `TTSConfig`, `DisplayConfig`, `VisualConfig`, `MemoryConfig`
- [x] 2.2 Implement `config.Defaults()` returning sensible default values for every field
- [x] 2.3 Implement `config.Load(path)` in `internal/config/loader.go`: read file → `toml.Unmarshal` onto defaults → `resolveEnv()`
- [x] 2.4 Implement `resolveEnv()`: apply `OLLAMA_HOST` override to endpoint, load `ANTHROPIC_API_KEY` into package-level `cloudAPIKeyStore`
- [x] 2.5 Expose `config.CloudAPIKey()` accessor

## 3. State Machine & Event Bus

- [x] 3.1 Define `EventType` constants (StateChanged, UserMessage, AgentTokens, AgentResponse, AgentError, AudioCaptured, STTResult, TTSStarted, TTSDone, WakeWordDetected) in `internal/state/events.go`
- [x] 3.2 Implement `Event` struct with `Type`, `Payload any`, `Timestamp`; implement `NewEvent(type, payload)`
- [x] 3.3 Implement `EventBus` with `Subscribe(EventType) <-chan Event`, `Publish(Event)` (non-blocking), and `Close()`
- [x] 3.4 Define `AgentState` enum (Idle, Listening, Processing, Thinking, Speaking, Error, Warmup) in `internal/state/machine.go`
- [x] 3.5 Implement `validTransitions` map encoding all legal state transitions
- [x] 3.6 Implement `StateMachine` with `Transition(to)`, `Current()`, `ForceState(s)`, and `OnTransition(fn)`
- [x] 3.7 Wire state machine to publish `EventStateChanged` with `StateTransition{From, To}` payload on every successful transition

## 4. Conversation Memory

- [x] 4.1 Define `Message` struct (`Role`, `Content`, `Timestamp`) in `internal/agent/memory.go`
- [x] 4.2 Implement `ConversationMemory` with mutex-protected `[]Message` slice
- [x] 4.3 Implement `Add(role, content)` with sliding-window trimming to `maxHistory`
- [x] 4.4 Implement `Messages()` returning a defensive copy of the slice
- [x] 4.5 Implement `Save()` writing `json.MarshalIndent` to the configured path
- [x] 4.6 Implement `load()` in `NewConversationMemory` — load existing JSON on init, treat missing file as empty (not error)

## 5. Persona System

- [x] 5.1 Define `Persona` struct (`Name`, `Description`, `SystemPrompt`) in `internal/agent/personality.go`
- [x] 5.2 Write system prompt content for `philosopher` persona (contemplative, curious, concise, dry humor)
- [x] 5.3 Write system prompt content for `companion` persona (warm, supportive, matches energy)
- [x] 5.4 Write system prompt content for `minimal` persona (direct, no filler)
- [x] 5.5 Implement `GetPersona(name)` with fallback to philosopher
- [x] 5.6 Implement `ListPersonas()` returning all registered names

## 6. Model Router

- [x] 6.1 Define `RoutingPreference` enum (`PreferLocal`, `PreferCloud`, `PreferAuto`) and `ModelInfo` struct in `internal/agent/router.go`
- [x] 6.2 Define `ModelRouter` interface: `Route(ctx) (model.BaseChatModel, error)`, `SetPreference(pref)`, `ListAvailable() []ModelInfo`
- [x] 6.3 Implement `router.localModel()` using `einoollama.NewChatModel` with `BaseURL` and `Model` from config
- [x] 6.4 Implement `router.cloudModel()` using `einoclaude.NewChatModel` with `APIKey`, `Model`, and `MaxTokens=4096`; return error if `config.CloudAPIKey()` is empty
- [x] 6.5 Implement `router.Route(ctx)` dispatching on preference with auto-fallback logic
- [x] 6.6 Implement `NewModelRouter(cfg)` deriving preference from `cfg.Inference.Mode` string

## 7. Agent Core

- [x] 7.1 Define `Agent` struct in `internal/agent/agent.go` composing config, router, memory, persona, state machine, bus, logger
- [x] 7.2 Implement `NewAgent(cfg, bus, log)`: create state machine, load memory, get persona, create router, transition Warmup→Idle
- [x] 7.3 Implement `agent.buildMessages()`: prepend `schema.SystemMessage(persona.SystemPrompt)`, append history as user/assistant messages
- [x] 7.4 Implement `agent.Chat(ctx, userMessage)`: transition to Thinking, publish EventUserMessage, add to memory, call `router.Route`, call `chatModel.Stream`, launch goroutine draining `StreamReader` → token channel
- [x] 7.5 In streaming goroutine: accumulate full response, add to memory as assistant turn, publish EventAgentResponse, transition back to Idle
- [x] 7.6 Implement `agent.SetPersona(name)`, `agent.SetRoutingPreference(pref)`, `agent.ClearMemory()`, `agent.State()`, `agent.MemoryLen()`, `agent.ListModels()`
- [x] 7.7 Implement `agent.Close()` calling `memory.Save()`
- [x] 7.8 Add placeholder `Tool` interface and `TimeTool` in `internal/agent/tools.go`

## 8. CLI Entry Point

- [x] 8.1 Add flags in `cmd/dusty/main.go`: `--config`, `--model`, `--mode`, `--persona`, `--verbose`
- [x] 8.2 Load config, apply flag overrides, initialize `slog` logger at Warn (or Debug if `--verbose`)
- [x] 8.3 Initialize `EventBus` and `Agent`; wire `os.Signal` handler for SIGINT/SIGTERM → `agent.Close()` + exit
- [x] 8.4 Print ASCII banner and startup status (agent name, persona, model, mode, memory count)
- [x] 8.5 Implement `bufio.Scanner` REPL: read line → handle slash command or call `agent.Chat` → stream tokens to stdout
- [x] 8.6 Implement `handleCommand()` for `/clear`, `/persona`, `/model`, `/mode`, `/status`, `/help`, `/quit`
- [x] 8.7 On EOF (Ctrl+D) call `agent.Close()` and exit cleanly

## 9. Dependencies & Build

- [x] 9.1 Run `go get github.com/cloudwego/eino-ext/components/model/ollama@v0.1.8`
- [x] 9.2 Run `go get github.com/cloudwego/eino-ext/components/model/claude@v0.1.16`
- [x] 9.3 Run `go get github.com/BurntSushi/toml`
- [x] 9.4 Run `go mod tidy` to resolve all transitive dependencies
- [x] 9.5 Verify `go build ./...` produces no errors
- [x] 9.6 Verify `GOOS=linux GOARCH=arm64 go build ./cmd/dusty` produces `bin/dusty-linux-arm64`

## 10. Tests

- [x] 10.1 Write config tests: defaults, missing file, field override, `OLLAMA_HOST` env override
- [x] 10.2 Write state machine tests: initial state, valid transitions, invalid transition (preserves state), callback firing, ForceState, EventBus publication
- [x] 10.3 Write EventBus tests: pub/sub, multiple subscribers, non-blocking on full channel, state change publishes event
- [x] 10.4 Write memory tests: add/retrieve, sliding window, clear, JSON round-trip persistence, missing file not an error
- [x] 10.5 Verify `go test ./...` passes with 18/18 tests green
