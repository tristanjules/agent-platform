# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

DUSTY (Desert Utility System for Thinking & Yearning) — a modular AI agent platform in Go targeting Raspberry Pi 5. Local LLM inference via Ollama with optional Claude cloud fallback, conversation memory, LoRa mesh communications, and a Bubble Tea v2 TUI. Ships as a single binary.

## Build & Run

```bash
make build                # Text REPL binary (no CGo, no hardware deps)
make run                  # Build + run with configs/default.toml
make build-voice          # Full voice build (requires whisper.cpp + piper)
make build-pi             # Cross-compile for Pi 5 (ARM64 Linux)
```

## Testing

```bash
make test                 # Stub build tags, runs everywhere
make test-all             # Includes hardware-dependent tests
# Single package:
go test -tags "noaudio nostt notts" ./internal/state/ -run TestStateMachine -timeout 60s
```

Always use `-tags "noaudio nostt notts"` when running tests locally without audio hardware.

## Lint

```bash
make lint                 # runs fmt + vet with stub tags
make fmt
make vet
```

## Build Tags

Three build tags control conditional compilation for hardware-dependent features:

- **`noaudio`** — stubs audio capture/playback (`internal/audio/*_stub.go`)
- **`nostt`** — stubs Whisper STT (`internal/stt/whisper_stub.go`)
- **`notts`** — stubs Piper TTS (`internal/tts/piper_stub.go`)

Default `make build` and `make test` include all three stubs. The `build-voice` target omits them (requires CGo + native libs).

## Architecture

### Core Flow (Text Mode)

```
stdin → REPL/TUI → agent.Chat(ctx, msg)
  → memory.Add("user", msg)
  → router.Route() → Eino BaseChatModel (Ollama or Claude)
  → tool-calling loop (up to 5 rounds via Generate(), non-streaming)
  → StreamGenerate() for final response → token channel → display
  → memory.Add("assistant", response)
  → EventBus publishes EventAgentResponse
```

### Key Packages

- **`internal/agent`** — Agent orchestration, model router (Ollama/Claude via Eino), persona system prompts, sliding-window conversation memory (JSON persistence), tool interface
- **`internal/state`** — `StateMachine` with enforced valid transitions (`Warmup→Idle→Thinking→Idle` for text; full voice loop adds `Listening→Processing→Speaking`). `EventBus` is pub/sub over buffered Go channels with non-blocking publish (drops on full channel)
- **`internal/config`** — TOML config with defaults-first pattern: `Defaults()` → `Load(path)` unmarshals over defaults → `resolveEnv()` for secrets. All 5 phases defined in structs now to avoid future schema migrations
- **`internal/mesh`** — LoRa mesh comms: 200-byte JSON envelope protocol, Meshtastic-compatible framing (0x94 0xC3 header), rate-limited transport, peer registry with TTL, message store, deduplicator
- **`internal/display/tui`** — Bubble Tea v2 TUI with conversation/reasoning panes, settings overlay, mesh transmission overlay, theme system (phosphor-green/amber/blue)
- **`internal/voice`** — 6-stage concurrent pipeline: AudioCapture → VAD → STT → Agent → TTS → Playback (all goroutines connected by typed channels)

### Entry Points

- **`cmd/dusty/main.go`** — CLI flags, mesh subsystem init, runs TUI or headless REPL based on TTY detection
- **`cmd/mesh-sim/main.go`** — Virtual LoRa hub (TCP relay) for local multi-node development without hardware

### LLM Framework

Uses [Eino](https://github.com/cloudwego/eino) (CloudWeGo) with `BaseChatModel` interface. `Generate()` for tool-calling rounds (structured, non-streaming), `StreamGenerate()` for final user-facing output. Provider implementations from `eino-ext` for Ollama and Claude.

### Mesh Protocol

- Terse JSON field names (`t`, `id`, `s`, `to`, `a`, `v`, `ag`, `pe`) to fit 200-byte LoRa envelope
- Message types: `cmd`, `ack`, `hb` (heartbeat), `loc` (GPS), `msg` (text), `syn` (sync), `dis` (discovery)
- Mesh tools (`mesh_send`, `mesh_inbox`) are injected into the Eino tool set at startup when mesh is enabled

### Config Files

- `configs/default.toml` — Desktop/dev (mesh disabled, local LLM)
- `configs/burning-man.toml` — Playa-optimized (mesh enabled, smaller history, fixed GPS)
- `configs/sim-node-{a,b}.toml` — Simulator mode (`serial_port = "sim://localhost:9090"`)

## Non-Obvious Patterns

- **Lazy model instantiation**: Router creates model instances on each `Route()` call, checking connectivity at inference time rather than startup
- **API key isolation**: `ANTHROPIC_API_KEY` stored in module-level variable, not in config struct, to prevent accidental logging/serialization
- **Event bus drop semantics**: Slow subscribers lose events rather than blocking the system — prevents one laggy component from stalling everything
- **Mesh idle suppression**: Notifier tracks user activity; incoming transmissions queue silently during idle, show unread count on return
- **Frame sync recovery**: Mesh simulator scans byte-by-byte for Meshtastic frame delimiters to re-sync after corruption

## OpenSpec

The `openspec/` directory contains the spec-driven development workflow. Specs live in `openspec/specs/`, changes in `openspec/changes/`. Use `/openspec-*` or `/opsx:*` slash commands to manage the artifact workflow.
