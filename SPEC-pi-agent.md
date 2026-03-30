# DUSTY: Portable On-Device AI Agent Platform

## Project Codename: DUSTY (Desert Utility System for Thinking & Yearning)

*A retro-future personal AI companion built on Raspberry Pi 5, designed for offline conversations at Burning Man and beyond.*

---

## 1. Vision & Goals

Build a modular, extensible personal AI agent platform that runs entirely on a Raspberry Pi 5. The system must operate fully offline (for Burning Man / playa conditions), but support cloud model routing when connectivity is available. The platform is an experimentation harness — not a fixed product — designed to evolve into many things over time.

### Core Experiences

**Written Mode (TUI)** — A stylish terminal interface built with Bubble Tea v2 showing real-time agent reasoning chains, streaming token output, and conversation history. Think retro-futuristic terminal aesthetic: scanlines, phosphor glow, box-drawing characters, typing animations.

**Visual Mode (Graphical)** — Dynamic retro bitmap-style animations rendered to a display. State-driven visuals for listening, thinking, speaking, idle. Pixel art, dithered gradients, CRT effects. Inspired by be-more-agent's reactive face system but evolved into something more abstract and generative.

**Voice Mode** — Full conversational loop: wake word → speech-to-text → LLM inference → text-to-speech → audio output. Hot-swappable voice personas via Piper TTS ONNX models.

---

## 2. Critical Analysis: be-more-agent Stack

The [be-more-agent](https://github.com/brenpoly/be-more-agent) project (504 stars, MIT license) provides an excellent proof-of-concept for offline Pi agents. Here's what it does well and where we diverge:

### What be-more-agent Gets Right

- **Fully offline architecture** — Zero cloud dependencies, Ollama + Whisper.cpp + Piper TTS is the right core stack for Pi inference.
- **State-driven animations** — The faces/ directory approach (idle, listening, thinking, speaking) is a clean state machine pattern.
- **Hardware abstraction** — Automatic microphone sample rate detection and resampling is smart for Pi audio variability.
- **Piper TTS voice fine-tuning** — Custom ONNX voice model (BMO voice v1.0) proves the voice customization pipeline works on Pi.

### Where We Improve

| Area | be-more-agent | DUSTY |
|------|---------------|-------|
| **Language** | Python 3.9+ | **Go** — lower memory footprint, better concurrency for real-time audio pipelines, single binary deployment, no virtualenv headaches on Pi |
| **Agent Framework** | Custom Python orchestration loop | **Eino (Go ADK)** — structured agent patterns (ReAct, DeepAgent), tool use, interrupt/resume, composable workflows |
| **TUI** | PNG image sequences only | **Bubble Tea v2** — full interactive TUI with real-time reasoning display, split panes, animations, plus a separate visual/graphical mode |
| **Model Routing** | Ollama only | **Abstraction layer** — Ollama primary, cloud fallback (Claude, OpenAI) via unified interface |
| **Voice System** | Single Piper voice | **Hot-swappable voice personas** — multiple ONNX models, runtime switching |
| **Memory** | Simple chat_memory.json | **Structured memory** — conversation history + semantic memory + personality persistence via Eino's context management |
| **Architecture** | Monolithic agent.py | **Modular services** — separate goroutines for audio capture, STT, LLM, TTS, display, all communicating via channels |

### Why Go Over Python for This

- **Memory** — Python's runtime overhead matters on a 8GB Pi running an LLM. Go's memory footprint is dramatically smaller for the orchestration layer.
- **Concurrency** — Goroutines and channels are a natural fit for the concurrent audio pipeline (capture → STT → LLM → TTS → playback all running simultaneously with streaming).
- **Deployment** — Single statically-linked binary vs. Python virtualenv + pip dependencies. Critical for reliability in harsh playa conditions.
- **Latency** — Go's startup time is near-instant. No Python interpreter warmup.
- **CGo bindings exist** — whisper.cpp has official Go bindings, Piper has community Go wrappers, Ollama is Go-native.

### Trade-off Acknowledged

Go has a thinner ML/AI ecosystem than Python. We mitigate this by using Go for orchestration and TUI while delegating inference to dedicated engines (Ollama, whisper.cpp, Piper) via their native APIs and CGo bindings.

---

## 3. Architecture

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
│       │              │             │              │         │
│  ┌────▼──────────────▼─────────────▼──────────────▼──────┐ │
│  │                  Event Bus (Go Channels)               │ │
│  └────┬──────────────┬─────────────┬──────────────┬──────┘ │
│       │              │             │              │         │
│  ┌────▼────┐  ┌──────▼───┐  ┌─────▼─────┐  ┌────▼─────┐  │
│  │  Wake   │  │  State   │  │   TUI     │  │  Visual  │  │
│  │  Word   │  │  Machine │  │(BubbleTea)│  │  Engine  │  │
│  │(onnx)   │  │          │  │           │  │ (bitmap) │  │
│  └─────────┘  └──────────┘  └───────────┘  └──────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            Model Router / Inference Layer             │  │
│  │  ┌─────────┐  ┌──────────┐  ┌────────────────────┐  │  │
│  │  │ Ollama  │  │ Claude   │  │ OpenAI / Others    │  │  │
│  │  │ (local) │  │ (cloud)  │  │ (cloud)            │  │  │
│  │  └─────────┘  └──────────┘  └────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Core Design Principles

**Event-Driven** — All components communicate via typed Go channels. The audio pipeline streams data between stages without blocking. State transitions (idle → listening → thinking → speaking) propagate to all display layers simultaneously.

**State Machine** — A central finite state machine governs the agent's mode. Every component subscribes to state changes and reacts accordingly. States: `Idle`, `Listening`, `Processing`, `Thinking`, `Speaking`, `Error`, `Warmup`.

**Hot-Swappable** — Voice models, LLM models, display modes, and even the agent's personality/system prompt can be changed at runtime via TUI commands or config reload.

---

## 4. Technology Stack

### 4.1 Core Language: Go 1.23+

### 4.2 Agent Framework: Eino (CloudWeGo)

[github.com/cloudwego/eino](https://github.com/cloudwego/eino) — Go-native LLM application framework with:

- **ChatModelAgent** — ReAct loop with tool calling, perfect for our conversational agent
- **DeepAgent** — Multi-agent orchestration for future expansion (sub-agents for different tasks)
- **Compose module** — Build deterministic pipelines (STT → process → LLM → TTS) as composable graphs
- **Interrupt/Resume** — Human-in-the-loop support for the TUI interaction mode
- **Ollama support** — Official implementation via EinoExt
- **Streaming-first** — Native stream handling throughout, critical for responsive conversation

**Why Eino over alternatives:**

- **vs. LangChainGo** — Eino is more Go-idiomatic, uses generics properly, and has better streaming support. LangChainGo is a translation of Python patterns that don't fit Go well.
- **vs. Custom from scratch** — Eino gives us tool calling, memory management, and agent patterns for free. Building these from scratch would delay getting to a working system by weeks.
- **vs. Agent SDK Go** — Less mature, fewer integrations. Eino has 10k+ stars and active CloudWeGo backing.

### 4.3 LLM Inference: Ollama (Primary) + Cloud Fallback

**Local (Offline):**
- **Ollama** as model server — Go-native, HTTP API, hot-swappable models
- **Recommended starting models:**
  - `gemma3:1b` — Best efficiency/quality ratio on Pi 5, ~15 tok/s
  - `phi-3.5:3b` — Better reasoning, ~8 tok/s, good for existential Burning Man conversations
  - `llama3.2:1b` — Fast, good instruction following
  - `qwen2.5:3b` — Strong multilingual, good reasoning

**Cloud (When Connected):**
- Claude API via Eino's ChatModel abstraction
- OpenAI API as additional option
- Routing logic: try local first, fall back to cloud on timeout or if user explicitly requests it

**Model Router Design:**
```go
type ModelRouter interface {
    Route(ctx context.Context, req ChatRequest) (ChatModel, error)
    SetPreference(pref RoutingPreference)  // local-only, cloud-preferred, auto
    ListAvailable() []ModelInfo
}
```

### 4.4 Speech-to-Text: whisper.cpp with Go Bindings

[github.com/ggml-org/whisper.cpp/bindings/go](https://pkg.go.dev/github.com/ggml-org/whisper.cpp/bindings/go/pkg/whisper)

- Official Go bindings, actively maintained (last published March 2026)
- Recommended model: `whisper-tiny` or `whisper-base` for Pi 5 (good accuracy, fast inference)
- Supports language detection, streaming chunks

### 4.5 Text-to-Speech: Piper TTS via Go

[github.com/piper-tts-go](https://github.com/piper-tts-go) — Go module that embeds Piper directly

- ONNX-based neural TTS, runs efficiently on Pi 5 ARM
- Hot-swappable voice models at runtime
- 35+ languages, dozens of pre-trained voices on HuggingFace
- Custom voice training pipeline via Piper's fine-tuning tools
- **Voice persona system:** Map named personas to ONNX model files + speaking rate + pitch config

### 4.6 Wake Word Detection: OpenWakeWord (ONNX)

- Run as a lightweight goroutine, always listening
- Custom wake word training possible
- ONNX format, runs on CPU efficiently
- Could potentially accelerate with Hailo NPU in future

### 4.7 TUI Framework: Bubble Tea v2

[charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) + Lip Gloss v2 + Bubbles v2

- **v2 released Feb 2026** — 10x performance improvement via new Cursed Renderer (ncurses-based)
- Declarative View fields (no more imperative commands)
- Built-in animation and transition support via `tea.Animation`
- Production-grade: used by NVIDIA, GitHub, AWS, Slack

**TUI Layout Design:**
```
┌─────────────────────────────────────────┐
│  DUSTY v0.1        [LOCAL] gemma3:1b    │  ← Status bar
├─────────────────────────────────────────┤
│                                         │
│  ┌─ Reasoning ────────────────────────┐ │
│  │ > Considering existential themes   │ │  ← Agent thinking stream
│  │ > Drawing from Camus, Watts...     │ │     (scrolling, dimmed)
│  │ > Formulating response...          │ │
│  └────────────────────────────────────┘ │
│                                         │
│  ┌─ Conversation ─────────────────────┐ │
│  │ You: What does it mean to be       │ │  ← Chat history
│  │      present in this moment?       │ │
│  │                                    │ │
│  │ DUSTY: Being present is less       │ │  ← Streaming response
│  │ about emptying the mind and more   │ │     (typing animation)
│  │ about filling awareness with what  │ │
│  │ actually is...█                    │ │
│  └────────────────────────────────────┘ │
│                                         │
│  ┌─ Input ────────────────────────────┐ │
│  │ > _                                │ │  ← Text input with cursor
│  └────────────────────────────────────┘ │
│                                         │
│  [TAB] Visual Mode  [F1] Voice  [F2]   │  ← Mode switching hotkeys
│  Settings                               │
└─────────────────────────────────────────┘
```

### 4.8 Visual Mode: Retro Bitmap Graphics Engine

**Approach:** A custom Go graphics renderer using a framebuffer or lightweight windowing library, outputting to the Pi's display.

**Options (ranked):**

1. **ebiten** ([github.com/hajimehoshi/ebiten](https://github.com/hajimehoshi/ebiten)) — 2D game engine for Go, supports ARM/Linux, GPU-accelerated via OpenGL ES. Perfect for retro pixel art animations. Can render to framebuffer on Pi.

2. **go-sdl2** — SDL2 bindings for Go, lower-level but gives direct framebuffer access and hardware-accelerated 2D rendering on Pi.

3. **Terminal-based** — Use Bubble Tea's half-block/braille character rendering for "graphics" that stay in the terminal. Lower fidelity but zero additional dependencies.

**Visual States & Animations:**

| State | Visual | Description |
|-------|--------|-------------|
| Idle | Breathing orb | Slow pulsing geometric shape, subtle scan lines |
| Listening | Waveform | Audio input visualized as retro oscilloscope waveform |
| Thinking | Matrix rain / neural net | Cascading symbols, branching tree patterns, dithered |
| Speaking | Mouth/wave sync | Amplitude-reactive shapes synced to TTS audio output |
| Error | Glitch | CRT distortion, static, red-shifted palette |
| Warmup | Boot sequence | Retro POST screen, loading bars, system check text |

**Pixel Art Style Guide:** 128x128 or 256x256 base resolution, upscaled with nearest-neighbor. 16-color palette (customizable). CRT shader overlay optional. All sprites/animations stored as asset packs that can be swapped.

---

## 5. Hardware Configuration

### 5.1 Base Build (~$200-250)

| Component | Spec | Price (est.) |
|-----------|------|-------------|
| Raspberry Pi 5 | 8GB RAM | $80 |
| SunFounder Dual NVMe Raft HAT | PCIe Gen 2, dual M.2 slots (NVMe + Hailo-8L) | $30 |
| NVMe SSD | 256GB M.2 2230 (Samsung PM991a or similar) | $25 |
| Display | 4" square IPS LCD (Hyperpixel 4.0 square or similar, 720x720) | $35-50 |
| Audio | USB audio adapter + small speaker + MEMS microphone | $15-20 |
| Power | USB-C PD power bank (65W, 20000mAh+) | $30-40 |
| Case | 3D printed or Pimoroni enclosure | $10-20 |
| MicroSD | 32GB for boot (OS + boot, data on NVMe) | $8 |

**Total: ~$235-290**

### 5.2 AI-Accelerated Upgrade Path (~$350-500)

Add the Hailo-8L AI accelerator ($30-50) into the second M.2 slot on the SunFounder HAT:

- **Hailo-8L**: 13 TOPS (INT8) — can accelerate whisper.cpp and wake word detection, freeing CPU for LLM inference
- Plugs into the same HAT as the NVMe SSD

**Future upgrade:** When budget allows, swap to the **Raspberry Pi AI HAT+ 2** with **Hailo-10H** (40 TOPS INT4). This requires a dedicated PCIe slot, so you'd need a different HAT arrangement (Pineboards Ai Bundle for NPU + NVMe combo, ~$60).

### 5.3 Hardware Accelerator Comparison

| Accelerator | TOPS | Interface | Can Stack with NVMe? | Price | Best For |
|-------------|------|-----------|---------------------|-------|----------|
| None (CPU only) | ~2 TOPS equiv | N/A | N/A | $0 | Starting out, 1-3B models |
| Hailo-8L | 13 TOPS (INT8) | M.2 M-key | Yes (SunFounder dual HAT) | ~$40 | STT/wake word offload |
| Hailo-8 (AI Kit) | 26 TOPS (INT8) | M.2 M-key | Yes (dual HAT) | ~$70 | Heavier vision/audio models |
| Hailo-10H (AI HAT+ 2) | 40 TOPS (INT4) | PCIe HAT | Yes (Pineboards Ai Bundle) | ~$110 | LLM acceleration, future-proof |
| Google Coral Edge TPU | 4 TOPS (INT8) | M.2 E-key | Yes (HatDrive! AI) | ~$35 | TFLite models, low power |

**Recommendation:** Start with CPU-only (Gemma3 1B runs fine at ~15 tok/s). Add Hailo-8L when you want to offload STT to free CPU headroom. The Hailo-10H is the endgame for on-device generative AI acceleration.

### 5.4 Power Budget (Battery Operation)

| Component | Typical Draw | Peak Draw |
|-----------|-------------|-----------|
| Pi 5 (idle) | 3W | — |
| Pi 5 (LLM inference) | 8-12W | 15W |
| NVMe SSD | 1-2W | 3W |
| Display (4" LCD) | 0.5-1W | 1.5W |
| Audio (USB + speaker) | 0.5W | 1W |
| Hailo-8L (if added) | 1-2W | 3W |
| **Total** | **~14-18W** | **~23W** |

With a 20,000mAh / 65W USB-C PD power bank (~72Wh): expect **4-5 hours** of active conversation, longer if idle between interactions. For Burning Man, bring 2-3 power banks and a solar panel for daytime charging.

### 5.5 Burning Man Considerations

- **Dust protection:** Sealed 3D-printed enclosure with filtered air intake, or fully sealed passive cooling (Pi 5 can throttle but survive in sealed case with heatsink)
- **Heat:** Playa temps can hit 40°C+. Use an aluminum heatsink case (Pimoroni or Argon) with thermal pads. Consider a small 5V fan if not sealed.
- **Display:** Anti-glare film on the LCD for daylight readability. High brightness helps.
- **Audio:** Directional MEMS microphone for better voice pickup in noisy camp environments. Consider a bone conduction speaker for personal listening.

---

## 6. Software Architecture Detail

### 6.1 Project Structure

```
dusty/
├── cmd/
│   └── dusty/
│       └── main.go              # Entry point, CLI flags, config loading
├── internal/
│   ├── agent/
│   │   ├── agent.go             # Eino ChatModelAgent setup
│   │   ├── router.go            # Model routing (local/cloud)
│   │   ├── memory.go            # Conversation memory & persistence
│   │   ├── personality.go       # System prompt & persona management
│   │   └── tools.go             # Agent tool definitions
│   ├── audio/
│   │   ├── capture.go           # ALSA microphone capture
│   │   ├── playback.go          # Audio output
│   │   ├── vad.go               # Voice activity detection
│   │   └── wakeword.go          # OpenWakeWord ONNX inference
│   ├── stt/
│   │   ├── engine.go            # STT interface
│   │   └── whisper.go           # whisper.cpp Go bindings
│   ├── tts/
│   │   ├── engine.go            # TTS interface
│   │   ├── piper.go             # Piper TTS Go bindings
│   │   └── personas.go          # Voice persona registry
│   ├── display/
│   │   ├── tui/
│   │   │   ├── app.go           # Bubble Tea application model
│   │   │   ├── views.go         # TUI view components
│   │   │   ├── reasoning.go     # Reasoning chain display
│   │   │   ├── conversation.go  # Chat history view
│   │   │   ├── input.go         # Text input component
│   │   │   └── theme.go         # Retro terminal theme (Lip Gloss)
│   │   └── visual/
│   │       ├── engine.go        # Ebiten-based visual renderer
│   │       ├── states.go        # Visual state animations
│   │       ├── sprites.go       # Sprite/animation asset loading
│   │       ├── shaders.go       # CRT/scanline shader effects
│   │       └── assets/          # Pixel art animation frames
│   ├── state/
│   │   ├── machine.go           # Central state machine
│   │   └── events.go            # Event types and bus
│   └── config/
│       ├── config.go            # Configuration struct
│       └── loader.go            # TOML config file loading
├── configs/
│   ├── default.toml             # Default configuration
│   └── burning-man.toml         # Offline-only preset
├── assets/
│   ├── voices/                  # Piper ONNX voice models
│   ├── visuals/                 # Pixel art animation packs
│   ├── sounds/                  # Sound effects (wav)
│   └── wakewords/               # Wake word ONNX models
├── go.mod
├── go.sum
├── Makefile                     # Build targets including cross-compile for ARM64
└── README.md
```

### 6.2 Event Bus & State Machine

```go
// Core agent states
type AgentState int
const (
    StateIdle AgentState = iota
    StateListening
    StateProcessing   // STT running
    StateThinking     // LLM generating
    StateSpeaking     // TTS playing
    StateError
    StateWarmup
)

// Events flow through typed channels
type Event struct {
    Type      EventType
    Payload   interface{}
    Timestamp time.Time
}

type EventBus struct {
    subscribers map[EventType][]chan Event
    mu          sync.RWMutex
}
```

### 6.3 Configuration (TOML)

```toml
[agent]
name = "DUSTY"
personality = "configs/personas/philosopher.toml"
wake_word = "hey dusty"

[inference]
mode = "local"  # "local", "cloud", "auto"

[inference.local]
provider = "ollama"
model = "gemma3:1b"
endpoint = "http://localhost:11434"

[inference.cloud]
provider = "anthropic"
model = "claude-sonnet-4-20250514"
# api_key loaded from env: ANTHROPIC_API_KEY

[stt]
engine = "whisper"
model = "base"  # tiny, base, small
language = "en"

[tts]
engine = "piper"
default_voice = "philosopher"

[tts.voices.philosopher]
model = "assets/voices/en_US-lessac-medium.onnx"
speaking_rate = 0.95
description = "Warm, contemplative voice for deep conversations"

[tts.voices.robot]
model = "assets/voices/en_US-ryan-medium.onnx"
speaking_rate = 1.0
description = "Crisp, slightly robotic delivery"

[display]
mode = "tui"  # "tui", "visual", "both"
theme = "phosphor-green"  # "phosphor-green", "amber", "blue", "custom"

[display.visual]
resolution = [256, 256]
palette = "16-retro"
crt_shader = true
```

---

## 7. Implementation Phases

### Phase 1: Foundations (Week 1-2)

**Goal:** Basic conversational loop in the terminal.

- Project scaffolding with Go modules
- Eino integration with Ollama ChatModelAgent
- Simple text input → LLM → text output loop
- Configuration system (TOML loading)
- Model router interface (Ollama local, Anthropic cloud)
- Basic conversation memory (in-memory, JSON persistence)

### Phase 2: Voice Pipeline (Week 3-4)

**Goal:** Speak to it, hear it respond.

- whisper.cpp Go bindings integration for STT
- Piper TTS Go bindings integration
- Audio capture via ALSA (PortAudio Go bindings)
- Voice activity detection (simple energy-based)
- Full voice loop: microphone → STT → LLM → TTS → speaker
- Voice persona loading and hot-swapping

### Phase 3: TUI (Week 5-6)

**Goal:** Beautiful terminal interface.

- Bubble Tea v2 application scaffolding
- Split-pane layout: reasoning + conversation + input
- Real-time streaming token display with typing animation
- Reasoning chain visualization (dimmed, scrolling)
- Retro terminal theme via Lip Gloss (phosphor green, scanlines via unicode)
- Mode switching hotkeys
- Settings panel for model/voice/personality selection

### Phase 4: Visual Mode (Week 7-8)

**Goal:** Retro bitmap animations on display.

- Ebiten-based renderer targeting Pi display
- State-driven animation system
- Pixel art asset pack for core states (idle, listen, think, speak)
- CRT shader effects (scanlines, vignette, phosphor bloom)
- Audio-reactive visuals (waveform display, amplitude-driven animations)
- Seamless switching between TUI and visual mode

### Phase 5: Polish & Playa-Ready (Week 9-10)

**Goal:** Reliable offline operation, personality tuning.

- Wake word detection integration
- Burning Man config preset (offline-only, power-saving)
- Custom personality/system prompt authoring
- Conversation export and review
- Error recovery and graceful degradation
- Performance profiling and optimization on Pi 5 hardware
- Dust-proof enclosure design / 3D print files

---

## 8. Key Dependencies

| Dependency | Purpose | Go Package |
|------------|---------|-----------|
| Eino | Agent framework | `github.com/cloudwego/eino` |
| Eino Ext | Ollama + cloud providers | `github.com/cloudwego/eino-ext` |
| Bubble Tea v2 | TUI framework | `charm.land/bubbletea/v2` |
| Lip Gloss v2 | TUI styling | `charm.land/lipgloss/v2` |
| Bubbles v2 | TUI components | `charm.land/bubbles/v2` |
| whisper.cpp | Speech-to-text | `github.com/ggml-org/whisper.cpp/bindings/go` |
| Piper TTS Go | Text-to-speech | `github.com/piper-tts-go/piper` |
| Ebiten | 2D graphics engine | `github.com/hajimehoshi/ebiten/v2` |
| PortAudio Go | Audio I/O | `github.com/gordonklaus/portaudio` |
| TOML | Config parsing | `github.com/BurntSushi/toml` |
| OpenWakeWord | Wake word (ONNX) | Run as sidecar or CGo bindings |

---

## 9. Open Questions & Future Exploration

1. **Hailo NPU integration** — Can we compile whisper.cpp or Piper models to run on Hailo via the HailoRT SDK? This would free the CPU entirely for LLM inference.

2. **Multi-agent patterns** — Eino's DeepAgent could enable a "council of voices" — multiple sub-agents with different personalities debating, with the main agent synthesizing. Very Burning Man.

3. **RAG for personal knowledge** — Add a vector store (Qdrant or Chroma, both have Go clients) for retrieval-augmented generation over personal documents, journals, books.

4. **Camera integration** — Moondream or LLaVA small vision models for "what do you see?" interactions. The Pi Camera Module v3 is $25.

5. **Mesh networking** — At Burning Man, could DUSTY instances discover each other via Bluetooth/WiFi Direct and have agent-to-agent conversations?

6. **Voice cloning** — Train a completely custom voice using Piper's fine-tuning pipeline with your own voice samples or a character voice.

7. **Offline knowledge base** — Pre-load Wikipedia dumps, philosophy texts, poetry collections onto the NVMe for the LLM to reference without internet.

---

## 10. Bill of Materials — Burning Man Build

| # | Item | Notes | Est. Cost |
|---|------|-------|-----------|
| 1 | Raspberry Pi 5 (8GB) | Core compute | $80 |
| 2 | SunFounder Dual NVMe Raft HAT | NVMe + future Hailo slot | $30 |
| 3 | Samsung PM991a 256GB NVMe 2230 | Model storage + data | $25 |
| 4 | Hyperpixel 4.0 Square (or similar 4" IPS) | 720x720 display | $50 |
| 5 | USB audio adapter + MEMS mic | Audio I/O | $15 |
| 6 | Small speaker (3W, 4Ω) | Output audio | $5 |
| 7 | Anker 737 Power Bank (140W, 24000mAh) | ~5hrs runtime | $40 |
| 8 | 32GB MicroSD (boot) | OS boot drive | $8 |
| 9 | Custom 3D printed enclosure | Dust-sealed, heatsink-integrated | $15 |
| 10 | Misc (cables, thermal pads, standoffs) | Assembly | $10 |
| | **Total** | | **~$278** |

---

*This is a living document. Update as hardware is acquired and design decisions are validated.*
