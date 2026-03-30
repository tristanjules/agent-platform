## Context

Phase 2 adds the voice pipeline on top of the Phase 1 text REPL. The target hardware is a Raspberry Pi 5 running Linux ARM64 in a high-noise outdoor environment (Burning Man playa). Key constraints:
- **Offline-first**: no internet once deployed; all inference is local
- **Power-conscious**: Pi runs on battery; unnecessary CPU work is expensive
- **Noisy environment**: VAD threshold and silence timeout tuned higher than defaults in `burning-man.toml`
- **Single binary**: `make build-voice` produces one statically-linked binary (plus external piper binary and whisper model)

## Goals / Non-Goals

**Goals:**
- Complete voice loop: mic → VAD → whisper.cpp STT → Ollama LLM → Piper TTS → speaker
- Sentence-level TTS streaming: first sentence plays while LLM generates the rest (~1 s first-word latency)
- Text REPL remains fully functional during voice mode (concurrent use)
- All tests pass with `make test` (no hardware required) via stub build tags
- CGo boundary isolated to `internal/stt/whisper.go` and `internal/audio/capture.go` / `playback.go`
- `gopls` and IDE analysis tools work without CGo headers via `.vscode/settings.json`

**Non-Goals:**
- Wake word detection (Phase 5)
- Bubble Tea TUI (Phase 3)
- Streaming STT (whisper.cpp batch mode is sufficient for utterances up to 30 s)
- Multiple simultaneous voice sessions
- WebRTC or network audio

## Decisions

### Build-tag isolation for CGo and hardware dependencies

**Decision:** Three paired build tags — `noaudio`/`!noaudio`, `nostt`/`!nostt`, `notts`/`!notts` — control which implementation files are compiled. Each hardware/CGo file has a matching pure-Go stub. The default `make build` applies all three stub tags.

**Rationale:** CI, developers without a microphone, and the text REPL use case must all compile and test cleanly without CGo toolchains or hardware. The `noaudio nostt notts` tags guarantee this. `make build-voice` drops the tags to enable the full pipeline. This pattern is idiomatic Go and avoids build matrix explosion.

---

### malgo (miniaudio) over gordonklaus/portaudio

**Decision:** Audio capture and playback use `github.com/gen2brain/malgo` (miniaudio Go bindings).

**Rationale:** PortAudio has known ARM64 issues and requires system-level installation (`libportaudio-dev`). malgo is a single-header C library bundled with the Go module; it only needs `-ldl` on Linux. It has ARM64-specific optimizations relevant for Pi 5. The miniaudio API is simpler for our single-device use case.

---

### Piper TTS via os/exec, not CGo

**Decision:** TTS synthesis calls the precompiled Piper ARM64 binary via `exec.CommandContext`, passing sentence text to stdin and reading raw PCM int16 from stdout (`--output_raw`).

**Rationale:** `github.com/piper-tts-go/piper` does not exist as a published Go module. Building Piper from source requires a complex CMake+ONNX Runtime toolchain. The rhasspy project provides precompiled ARM64 binaries. `os/exec` is simpler, avoids CGo, allows hot-swapping the binary without recompiling, and streams audio at the sentence boundary.

---

### One Piper process per sentence (not streaming stdin)

**Decision:** `synthesizeSentence` spawns a new `piper` process per sentence and reads all stdout via `cmd.Output()`.

**Rationale:** Piper's `--output_raw` mode reads one utterance from stdin and writes raw PCM then exits. Keeping stdin open for continuous streaming is not supported by the current Piper CLI. Per-sentence process spawning has negligible overhead (~5 ms) compared to synthesis time (~200–500 ms per sentence). It simplifies error handling and enables clean context cancellation between sentences.

---

### whisper.cpp in extern/ not vendor/

**Decision:** whisper.cpp is cloned into `extern/whisper.cpp` (gitignored), not `vendor/whisper.cpp`.

**Rationale:** Go's `vendor/` directory is managed exclusively by `go mod vendor` which deletes any non-Go-module content on every run. Placing the C library source there would cause it to be silently wiped. `extern/` is a conventional location for external C dependencies and is explicitly gitignored via `.gitignore`.

---

### Sentence splitting in the agent goroutine, not in the TTS engine

**Decision:** The voice loop's `agentGoroutine` splits LLM token output on `.?!\n` boundaries and sends individual sentences to `ttsIn`. The TTS engine receives one sentence per `Synthesize` call.

**Rationale:** Splitting at the agent layer enables first-sentence synthesis to begin while the LLM is still generating tokens two sentences ahead. If splitting were done only inside the TTS engine (after accumulating the full response), the first audio chunk would be delayed until the entire LLM response was complete. The ~1 s first-word latency target requires pipeline parallelism at the sentence level.

---

### Six fixed goroutines with typed channels, not a dynamic worker pool

**Decision:** `VoiceLoop.Run` launches exactly six goroutines (capture, VAD, STT, agent, TTS, playback) connected by typed channels. No worker pools, no dynamic scaling.

**Rationale:** The pipeline is strictly sequential — each stage feeds exactly one downstream stage. A worker pool would add complexity without benefit since parallelism within a stage (e.g., two simultaneous whisper transcriptions) is neither safe nor desirable (context continuity). Fixed goroutines with predictable channel buffer sizes make the data flow easy to reason about and test with mock implementations.

---

### Energy-based VAD (RMS threshold), not a neural VAD model

**Decision:** `audio.VAD` uses root mean square energy with a configurable threshold to detect speech boundaries.

**Rationale:** Neural VAD models (e.g., Silero) require ONNX Runtime or a custom CGo binding, adding a third CGo dependency. Energy-based VAD is a pure Go implementation, fully testable with synthetic PCM without hardware. For close-range microphone use (the primary DUSTY use case) RMS energy detection is reliable. The `burning-man.toml` preset raises the threshold to handle playa background noise.

---

### Channel buffer sizes from architecture plan

| Channel | Buffer | Rationale |
|---------|--------|-----------|
| `audioRaw` | 32 | Absorbs malgo callback jitter |
| `vadOut` | 4 | Max queued utterances before backpressure |
| `sttOut` | 4 | whisper.cpp is slow; queue prevents dropped speech |
| `ttsIn` | 8 | Absorbs burst sentences at response start |
| `playIn` | 64 | Allows piper to run ahead of playback |
