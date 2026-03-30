# DUSTY

**Desert Utility System for Thinking & Yearning**

A retro-future personal AI companion built on Raspberry Pi 5. Designed for fully-offline conversations at Burning Man and beyond — but just as at home on your desk.

```
  ██████╗ ██╗   ██╗███████╗████████╗██╗   ██╗
  ██╔══██╗██║   ██║██╔════╝╚══██╔══╝╚██╗ ██╔╝
  ██║  ██║██║   ██║███████╗   ██║    ╚████╔╝
  ██║  ██║██║   ██║╚════██║   ██║     ╚██╔╝
  ██████╔╝╚██████╔╝███████║   ██║      ██║
  ╚═════╝  ╚═════╝ ╚══════╝   ╚═╝      ╚═╝
```

---

## What Is This

DUSTY is a modular AI agent platform written in Go. It runs local LLM inference via Ollama with optional cloud fallback (Claude), maintains conversation memory between sessions, and is designed to expand into a full voice + visual interface system. The whole thing ships as a single binary that cross-compiles to the Pi 5's ARM64.

The spec lives in [`SPEC-pi-agent.md`](SPEC-pi-agent.md) — a living document describing the full 5-phase roadmap from terminal chatbot to voice-activated playa companion with retro pixel art animations.

**Current status: Phase 1 complete** — conversational REPL with streaming output, conversation memory, persona system, and Ollama/Claude model routing.

---

## Quick Start

### Prerequisites

- Go 1.23+
- [Ollama](https://ollama.com) running locally

```bash
# Install and start Ollama
ollama serve
ollama pull gemma3:1b        # Best for Pi 5 (~15 tok/s)
# or
ollama pull phi3.5:3b        # Better reasoning, slower
ollama pull llama3.2:1b      # Fast, good instruction-following
```

### Build & Run

```bash
git clone https://github.com/tristanj/dusty
cd dusty

make run
# or
make build && ./bin/dusty
```

### Run with cloud (Anthropic)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./bin/dusty --mode cloud
# or for auto-fallback (local first, cloud if Ollama unreachable):
./bin/dusty --mode auto
```

### Deploy to Raspberry Pi

```bash
make cross-compile-pi
# Copy bin/dusty-linux-arm64 to Pi, rename to dusty, run it.
scp bin/dusty-linux-arm64 pi@raspberrypi.local:~/dusty
```

---

## REPL Commands

Once running, type a message to chat. Slash commands are available at any time:

| Command | Description |
|---|---|
| `/clear` | Wipe conversation memory |
| `/persona <name>` | Switch personality: `philosopher` `companion` `minimal` |
| `/model [name]` | List available models, or switch to a different one |
| `/mode <mode>` | Change routing: `local` `cloud` `auto` |
| `/status` | Show current agent state, memory count, model, persona |
| `/help` | Show all commands |
| `/quit` | Save memory and exit |

Conversation history is persisted to `dusty.memory.json` and reloaded on startup. `Ctrl+C` saves before exit.

---

## Configuration

DUSTY is configured with TOML files in `configs/`. The binary loads `configs/default.toml` by default.

```bash
./bin/dusty --config configs/burning-man.toml   # Offline-only preset
./bin/dusty --config my-custom.toml
```

**Key config sections** (see [`configs/default.toml`](configs/default.toml) for the full schema):

```toml
[agent]
name = "DUSTY"
personality = "philosopher"   # philosopher | companion | minimal
max_history = 50              # sliding window for context

[inference]
mode = "local"                # local | cloud | auto

[inference.local]
provider = "ollama"
model = "gemma3:1b"
endpoint = "http://localhost:11434"

[inference.cloud]
provider = "anthropic"
model = "claude-sonnet-4-20250514"
# API key from env: ANTHROPIC_API_KEY

[memory]
persist = true
path = "dusty.memory.json"
```

**Environment variables:**

| Variable | Effect |
|---|---|
| `ANTHROPIC_API_KEY` | Enables cloud inference via Claude |
| `OLLAMA_HOST` | Overrides Ollama endpoint (useful for Pi deployment) |

### Flag overrides

Any config value can be overridden at launch:

```bash
./bin/dusty \
  --config configs/default.toml \
  --model phi3.5:3b \
  --persona companion \
  --mode auto \
  --verbose
```

---

## Architecture

### System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        DUSTY Agent                          │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐ │
│  │  Audio    │  │  STT     │  │  Brain   │  │  TTS       │ │
│  │  Capture  │──│  Engine  │──│  (Eino)  │──│  Engine    │ │
│  │  (ALSA)  │  │(whisper) │  │          │  │  (Piper)   │ │
│  └──────────┘  └──────────┘  └──────────┘  └────────────┘ │
│       │              │             │              │         │
│  ┌────▼──────────────▼─────────────▼──────────────▼──────┐ │
│  │                  Event Bus (Go Channels)               │ │
│  └────┬──────────────┬─────────────┬──────────────┬──────┘ │
│       │              │             │              │         │
│  ┌────▼────┐  ┌──────▼───┐  ┌─────▼─────┐  ┌────▼─────┐  │
│  │  Wake   │  │  State   │  │   TUI     │  │  Visual  │  │
│  │  Word   │  │  Machine │  │(BubbleTea)│  │  Engine  │  │
│  │ (onnx)  │  │          │  │           │  │ (Ebiten) │  │
│  └─────────┘  └──────────┘  └───────────┘  └──────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            Model Router / Inference Layer             │  │
│  │  ┌─────────┐  ┌──────────┐  ┌────────────────────┐  │  │
│  │  │ Ollama  │  │ Claude   │  │  OpenAI / Others   │  │  │
│  │  │ (local) │  │ (cloud)  │  │  (cloud)           │  │  │
│  │  └─────────┘  └──────────┘  └────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

*Grey boxes = Phase 1 complete. All others are scaffolded and ready for implementation.*

### Design Principles

**Event-Driven** — All components communicate through a typed `EventBus` backed by Go channels. The audio pipeline (Phase 2) streams audio → STT → LLM → TTS without blocking. Every state transition (Idle → Thinking → Idle) is broadcast as an event that any subscriber — TUI, visual engine, audio — can react to independently.

**State Machine** — A central `StateMachine` enforces valid transitions between agent modes: `Warmup → Idle → Listening → Processing → Thinking → Speaking → Idle`. Invalid transitions return an error; no component can put the agent in an inconsistent state.

**Hot-Swappable** — Voice models, LLM models, display modes, and personas can all be changed at runtime. The `ModelRouter` resolves the correct `ChatModel` on every request, so `./bin/dusty --model qwen2.5:3b` takes effect immediately.

**Single Binary** — The entire orchestration layer is one statically-linked Go binary. No Python virtualenv, no npm, no Docker. `scp` it to the Pi and run it.

### Package Structure

```
dusty/
├── cmd/dusty/
│   └── main.go                  # CLI flags, REPL loop, signal handling
│
├── internal/
│   ├── config/
│   │   ├── config.go            # Full config schema (all 5 phases defined)
│   │   └── loader.go            # TOML loading + env var resolution
│   │
│   ├── state/
│   │   ├── events.go            # EventBus, EventType constants, Event struct
│   │   └── machine.go           # StateMachine with validated transitions
│   │
│   ├── agent/
│   │   ├── agent.go             # Agent: Chat() streaming method, lifecycle
│   │   ├── router.go            # ModelRouter: Ollama / Claude via Eino
│   │   ├── memory.go            # ConversationMemory + JSON persistence
│   │   ├── personality.go       # Persona definitions and system prompts
│   │   └── tools.go             # Tool interface (placeholder for Phase 2+)
│   │
│   ├── audio/                   # Phase 2 — ALSA capture, playback, VAD
│   ├── stt/                     # Phase 2 — whisper.cpp Go bindings
│   ├── tts/                     # Phase 2 — Piper TTS, voice personas
│   ├── display/tui/             # Phase 3 — Bubble Tea v2 interface
│   └── display/visual/          # Phase 4 — Ebiten pixel art renderer
│
├── configs/
│   ├── default.toml             # Default configuration
│   └── burning-man.toml         # Offline-only, power-saving preset
│
└── assets/
    ├── voices/                  # Piper ONNX voice models (.gitignored)
    ├── visuals/                 # Pixel art animation packs
    ├── sounds/                  # Sound effects
    └── wakewords/               # OpenWakeWord ONNX models
```

### Data Flow (Phase 1)

```
stdin
  │
  ▼
main.go REPL loop
  │  reads line, checks for /commands
  ▼
agent.Chat(ctx, userMessage)
  │  saves user message to ConversationMemory
  │  calls router.Route(ctx) → Eino BaseChatModel (Ollama or Claude)
  │  builds message slice: [SystemPrompt] + [history...] + [user message]
  │  calls chatModel.Stream(ctx, messages)
  │  launches goroutine that drains StreamReader → token channel
  ▼
<-chan string  (streaming tokens)
  │
  ▼
main.go prints each token as it arrives
  │
  ▼
goroutine completes, assembles full response, saves to ConversationMemory
  │
  ▼
StateMachine: Thinking → Idle
EventBus: publishes EventAgentResponse
```

### State Machine Transitions

```
Warmup ──→ Idle ──→ Thinking ──→ Idle      (text mode, Phase 1)
                │
                └──→ Listening ──→ Processing ──→ Thinking ──→ Speaking ──→ Idle
                                                                            (voice mode, Phase 2)
```

Any state can transition to `Error`. `Error` can recover to `Idle` or `Warmup`.

---

## Key Technical Decisions

### Why Go

- **Memory footprint** — On a Pi 5 running an 8GB LLM, the orchestration layer needs to be lean. Go's runtime is dramatically smaller than Python's.
- **Concurrency** — Goroutines + channels map naturally to the concurrent audio pipeline: capture, STT, LLM inference, and TTS playback run simultaneously in Phase 2.
- **Single binary** — `GOOS=linux GOARCH=arm64 go build` produces a self-contained binary. No `pip install`, no `apt-get`, no virtualenv breakage on the playa.
- **CGo bindings exist** — whisper.cpp has official Go bindings; Piper has community wrappers; Ollama is written in Go.

### Why Eino for LLM

[Eino](https://github.com/cloudwego/eino) (CloudWeGo) provides a Go-native LLM application framework with a clean `BaseChatModel` interface, streaming support, and provider implementations for Ollama and Claude in the `eino-ext` package. Using its `BaseChatModel` as the abstraction boundary means the agent code is identical whether talking to `gemma3:1b` on a Pi or `claude-sonnet-4` in the cloud.

**Alternatives considered:**

| Option | Why not chosen |
|---|---|
| LangChainGo | Translates Python patterns poorly to Go; weaker streaming |
| Direct Ollama/Anthropic clients | Loses the provider abstraction; more code for model switching |
| Custom from scratch | Weeks of work reinventing tool calling, streaming, and agent loops |

### Why the Config Schema Is Fully Defined Now

All five phases of config (STT, TTS, display, visual) are defined as Go structs with TOML tags even though Phase 1 only uses `agent`, `inference`, and `memory`. This avoids config schema migrations and ensures `.toml` files written today are valid when Phase 2 and 3 arrive.

---

## Testing

```bash
make test          # Run all tests with verbose output
go test ./...      # Same, without verbose
```

**Test coverage:**

| Package | Tests |
|---|---|
| `internal/config` | Defaults, TOML loading, field overrides, `OLLAMA_HOST` env override |
| `internal/state` | State machine valid/invalid transitions, callback firing, no-state-change on error, `EventBus` pub/sub, non-blocking on full channel, state changes emit bus events |
| `internal/agent` | Memory add/retrieve, sliding window, clear, JSON round-trip persistence, graceful handling of missing file |

---

## Phase Roadmap

| Phase | Goal | Status |
|---|---|---|
| **1 — Foundations** | Text REPL, Eino/Ollama/Claude, conversation memory, config, state machine, event bus | ✅ Complete |
| **2 — Voice Pipeline** | whisper.cpp STT + Piper TTS + PortAudio capture, VAD, full voice loop, hot-swap voices | 🔲 Next |
| **3 — TUI** | Bubble Tea v2 split-pane interface with streaming tokens, reasoning display, retro phosphor theme | 🔲 Planned |
| **4 — Visual Mode** | Ebiten pixel art renderer, state-driven animations (idle/listen/think/speak), CRT shader effects | 🔲 Planned |
| **5 — Playa Ready** | Wake word detection, power-saving presets, error recovery, Pi 5 performance tuning | 🔲 Planned |

### Phase 2 Preview: Voice Pipeline

The voice loop will use:
- **[whisper.cpp Go bindings](https://github.com/ggml-org/whisper.cpp/bindings/go)** — official, actively maintained; `whisper-tiny` or `whisper-base` on Pi 5
- **[Piper TTS Go](https://github.com/piper-tts-go/piper)** — ONNX-based neural TTS, runs on ARM, hot-swappable `.onnx` voice models
- **[PortAudio Go](https://github.com/gordonklaus/portaudio)** — cross-platform audio I/O for microphone capture and speaker playback

The `internal/audio`, `internal/stt`, and `internal/tts` packages are already stubbed and will be filled in. The state machine transitions (`Idle → Listening → Processing → Thinking → Speaking → Idle`) and event bus are already wired for this.

### Phase 3 Preview: TUI Design

```
┌─────────────────────────────────────────┐
│  DUSTY v0.1        [LOCAL] gemma3:1b    │  ← status bar
├─────────────────────────────────────────┤
│  ┌─ Reasoning ────────────────────────┐ │
│  │ > Considering existential themes   │ │  ← dimmed, scrolling
│  │ > Drawing from Camus, Watts...     │ │
│  └────────────────────────────────────┘ │
│  ┌─ Conversation ─────────────────────┐ │
│  │ You: What does it mean to be       │ │
│  │      present in this moment?       │ │
│  │                                    │ │
│  │ DUSTY: Being present is less       │ │  ← streaming tokens
│  │ about emptying the mind...█        │ │
│  └────────────────────────────────────┘ │
│  ┌─ Input ────────────────────────────┐ │
│  │ > _                                │ │
│  └────────────────────────────────────┘ │
│  [TAB] Visual Mode  [F1] Voice  [F2] Settings │
└─────────────────────────────────────────┘
```

Built with [Bubble Tea v2](https://charm.land/bubbletea/v2) + Lip Gloss v2. Released February 2026 with 10× rendering performance improvement.

---

## Hardware Target

### Base Build (~$280)

| Component | Spec | Est. Cost |
|---|---|---|
| Raspberry Pi 5 | 8GB RAM | $80 |
| SunFounder Dual NVMe Raft HAT | PCIe Gen 2, dual M.2 (NVMe + Hailo slot) | $30 |
| Samsung PM991a NVMe | 256GB M.2 2230 | $25 |
| Hyperpixel 4.0 Square | 4" IPS, 720×720 | $50 |
| USB audio adapter + MEMS mic | Audio I/O | $15 |
| Small speaker (3W, 4Ω) | Output | $5 |
| Anker 737 Power Bank | 24000mAh, 140W — ~5hr runtime | $40 |
| 32GB MicroSD | Boot drive | $8 |
| 3D printed enclosure | Dust-sealed | $15 |

**Battery runtime:** ~4–5 hours active conversation at 14–18W draw. Bring 2–3 banks for a Burning Man day.

### AI Accelerator Upgrade Path

| Accelerator | TOPS | Price | Best For |
|---|---|---|---|
| CPU only | ~2 | $0 | Starting out, 1–3B models |
| Hailo-8L (M.2) | 13 INT8 | ~$40 | Offload STT/wake word, free CPU for LLM |
| Hailo-10H (AI HAT+ 2) | 40 INT4 | ~$110 | LLM acceleration, endgame |

---

## Dependencies

| Package | Version | Purpose |
|---|---|---|
| `github.com/cloudwego/eino` | v0.7.13 | LLM agent framework, `BaseChatModel` interface |
| `github.com/cloudwego/eino-ext/components/model/ollama` | v0.1.8 | Ollama provider for Eino |
| `github.com/cloudwego/eino-ext/components/model/claude` | v0.1.16 | Anthropic Claude provider for Eino |
| `github.com/BurntSushi/toml` | v1.6.0 | TOML config parsing |
| `github.com/anthropics/anthropic-sdk-go` | v1.26.0 | Anthropic SDK (via claude eino-ext) |
| **Phase 2** | | |
| `github.com/ggml-org/whisper.cpp/bindings/go` | — | Speech-to-text |
| `github.com/piper-tts-go/piper` | — | Text-to-speech |
| `github.com/gordonklaus/portaudio` | — | Audio I/O |
| **Phase 3** | | |
| `charm.land/bubbletea/v2` | — | TUI framework |
| `charm.land/lipgloss/v2` | — | TUI styling |
| **Phase 4** | | |
| `github.com/hajimehoshi/ebiten/v2` | — | 2D graphics engine for pixel art renderer |

---

## Makefile

```bash
make build              # Build ./bin/dusty
make run                # Build and run with configs/default.toml
make test               # Run all tests
make cross-compile-pi   # Build Linux ARM64 binary for Pi 5
make clean              # Remove ./bin/
make fmt                # gofmt all packages
make vet                # go vet all packages
```

---

## Future Directions

From the spec — open questions worth exploring as later phases mature:

- **Hailo NPU** — Compile whisper.cpp/Piper to run on Hailo-8L via HailoRT SDK; frees CPU entirely for LLM
- **Council of Voices** — Eino's multi-agent patterns for multiple sub-agents with different personalities debating, synthesized by the main agent
- **RAG** — Vector store (Qdrant/Chroma, both have Go clients) over personal journals, philosophy texts, books
- **Camera** — Moondream or LLaVA small vision model + Pi Camera Module v3 for "what do you see?" interactions
- **Mesh** — DUSTY instances discovering each other via WiFi Direct at Burning Man for agent-to-agent conversation
- **Voice cloning** — Custom ONNX model trained on your own voice via Piper's fine-tuning pipeline

---

*DUSTY is a living project. The spec evolves as hardware is acquired and design decisions are validated on the playa.*
