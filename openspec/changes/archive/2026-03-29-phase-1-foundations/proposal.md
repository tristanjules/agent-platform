## Why

DUSTY needs a solid Go foundation before voice, TUI, or visual features can be built. Phase 1 establishes the minimal viable conversational loop — text in, streaming LLM response out — along with all the scaffolding (state machine, event bus, config schema, memory persistence) that later phases plug into. Building this foundation correctly now prevents painful rewrites at Phase 3+.

## What Changes

- Introduce the `github.com/tristanj/dusty` Go module with the full directory structure for all 5 phases
- Add TOML configuration system with complete schema covering agent, inference, STT, TTS, display, and memory
- Add a typed `EventBus` (channel pub/sub) and `StateMachine` (FSM with enforced valid transitions) in `internal/state`
- Add a `ConversationMemory` with sliding-window history and JSON file persistence in `internal/agent`
- Add a `Persona` system with three built-in personalities (philosopher, companion, minimal) and TOML-loadable system prompts
- Add a `ModelRouter` that creates Eino `BaseChatModel` instances for Ollama (local) and Anthropic Claude (cloud) with local/cloud/auto routing modes
- Add an `Agent` type that wires router + memory + persona + state machine into a single `Chat(ctx, message) → <-chan string` streaming interface
- Add `cmd/dusty/main.go`: a CLI REPL with streaming token output, slash commands, graceful shutdown, and memory save on exit
- Add `configs/default.toml` and `configs/burning-man.toml` presets
- Add `Makefile` with build, test, cross-compile-pi, fmt, and vet targets

## Capabilities

### New Capabilities

- `conversational-agent`: Core agent that manages LLM inference, conversation history, and persona — exposes a streaming `Chat` interface
- `model-router`: Routing layer that selects between local Ollama and cloud Anthropic Claude based on config mode (local/cloud/auto), abstracted behind Eino's `BaseChatModel`
- `conversation-memory`: In-memory conversation history with configurable sliding-window and JSON persistence to disk across sessions
- `persona-system`: Named personality presets that set the agent's system prompt, tone, and behavior; switchable at runtime
- `state-machine`: Central finite state machine governing agent lifecycle states (Warmup/Idle/Listening/Processing/Thinking/Speaking/Error) with enforced valid transitions
- `event-bus`: Typed publish/subscribe event bus using Go channels; connects agent, state machine, and future display/audio components
- `config-system`: TOML configuration loader covering all 5 phases of DUSTY, with env-var overrides for secrets and deployment

### Modified Capabilities

## Impact

- **New module**: `github.com/tristanj/dusty` at `go 1.25.6`
- **New dependencies**: `github.com/cloudwego/eino@v0.7.13`, `github.com/cloudwego/eino-ext/components/model/ollama@v0.1.8`, `github.com/cloudwego/eino-ext/components/model/claude@v0.1.16`, `github.com/BurntSushi/toml@v1.6.0`
- **Runtime dependencies**: Ollama process for local inference; `ANTHROPIC_API_KEY` env var for cloud inference
- **Binary**: `bin/dusty` (macOS) and `bin/dusty-linux-arm64` (Raspberry Pi 5)
- **No breaking changes** — this is the initial implementation
