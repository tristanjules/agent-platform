## Context

DUSTY is a Go-based AI agent platform targeting offline operation on a Raspberry Pi 5 for Burning Man. Phase 1 establishes the core conversational loop and all the architectural plumbing that phases 2–5 will build on. The system is designed for a single-developer hobby context with a strong preference for simplicity, low operational overhead, and a single deployable binary.

The primary constraint is that all Go module dependencies must work on `GOOS=linux GOARCH=arm64` for Pi 5 deployment. CGo dependencies (whisper.cpp, Piper TTS) are deferred to Phase 2 where they are required.

## Goals / Non-Goals

**Goals:**
- Working text-in / streaming-text-out conversational loop with Ollama on localhost
- Cloud fallback to Anthropic Claude when `ANTHROPIC_API_KEY` is set
- Conversation history persisted as JSON across restarts with configurable sliding-window
- Full config schema defined for all 5 phases (even if unused in Phase 1) to avoid future migrations
- State machine and event bus scaffolded and in use even if lightly — so Phase 2+ components can subscribe without refactoring
- Single `go build` produces a binary that works locally and cross-compiles to ARM64 without CGo

**Non-Goals:**
- Voice I/O (Phase 2)
- Bubble Tea TUI (Phase 3)
- Ebiten visual renderer (Phase 4)
- Wake word detection (Phase 5)
- Tool calling / agentic loops (future)
- Multi-turn reasoning chains / chain-of-thought display
- Semantic memory / vector search

## Decisions

### Use Eino `BaseChatModel` as the provider abstraction

**Decision:** The `ModelRouter` returns `model.BaseChatModel` from `github.com/cloudwego/eino/components/model`, not the concrete `*ollama.ChatModel` or `*claude.ChatModel`.

**Rationale:** This gives us a single `Stream(ctx, []*schema.Message) → *schema.StreamReader` interface regardless of provider. The `agent.go` Chat method is completely provider-agnostic — swapping Ollama for Gemini or a future local provider requires only a new `router.go` branch. Alternatives considered: direct `github.com/ollama/ollama/api` client (loses abstraction) or LangChainGo (too Python-centric, weaker streaming).

---

### Lazy ChatModel creation (on each `Route()` call)

**Decision:** `router.Route(ctx)` creates a new `ChatModel` instance on every call rather than caching one at startup.

**Rationale:** Ollama may not be running when `dusty` starts (especially on the Pi). Lazy creation means the binary starts instantly and gives a clear error at the moment you try to chat, rather than failing at init. It also makes model switching (e.g. `/model phi3.5:3b`) take effect immediately without restart. The overhead of constructing a lightweight HTTP client wrapper per-request is negligible.

---

### Conversation history in `[]*schema.Message` form, built fresh each request

**Decision:** `agent.buildMessages()` constructs the full message slice (system prompt + history) on every `Chat()` call rather than maintaining a running `[]*schema.Message` slice.

**Rationale:** The `ConversationMemory` stores `[]agent.Message` (our own type with timestamps) which is more useful for persistence and inspection. Converting to Eino's `schema.Message` at call time keeps the memory layer decoupled from the Eino schema. This also makes it easy to inject a different system prompt or truncate history without touching the stored messages.

---

### Sliding-window memory truncation (not summarization)

**Decision:** Memory truncation uses a simple sliding window (`messages[len-maxHistory:]`) rather than LLM-based summarization.

**Rationale:** Summarization requires an additional LLM call which adds latency and complexity. On small models (1B–3B) running on a Pi, summarization quality is poor anyway. A 50-message window (default) is plenty for meaningful multi-turn conversations. Summarization can be layered on in a later phase if needed.

---

### EventBus uses non-blocking publish (drop on full channel)

**Decision:** `EventBus.Publish()` uses a `select { case ch <- event: default: }` pattern — if a subscriber's buffer is full, the event is silently dropped for that subscriber.

**Rationale:** In Phase 1 the bus is lightly used (state transitions, token events). In Phase 2+ the audio pipeline will be publishing at high frequency (audio chunks). Blocking publish would allow a slow TUI subscriber to stall audio capture — unacceptable for real-time voice. Dropping events for slow subscribers is always preferable to blocking producers. Subscribers that need lossless delivery should use larger buffers.

---

### Config struct fully defined for all 5 phases

**Decision:** `internal/config/config.go` defines `STTConfig`, `TTSConfig`, `DisplayConfig`, and `VisualConfig` even though they're empty in Phase 1.

**Rationale:** TOML files written during Phase 1 (`configs/default.toml`) should remain valid at Phase 5. If we add fields later without the struct scaffold, old config files will fail TOML unmarshaling. Defining the structs now with zero cost makes the schema forward-compatible.

---

### API key stored in package-level var, not in Config struct

**Decision:** `config.cloudAPIKeyStore` is a package-level `string` populated during `Load()`. The `Config` struct does not have an `APIKey` field.

**Rationale:** Prevents the API key from being accidentally serialized (e.g., JSON marshal of Config for debugging), logged via `%+v`, or written to memory files. The key is accessed via `config.CloudAPIKey()` only when the cloud provider is instantiated.

## Risks / Trade-offs

**Eino version lock** → Eino is pre-1.0. API surface may change in ways that require updates to `router.go` and `agent.go`. Mitigation: The `BaseChatModel` interface is minimal (`Generate` + `Stream`); any breakage is localized to the router.

**Ollama not installed on dev machine** → Build and tests pass without Ollama, but integration testing requires it. Mitigation: Unit tests mock at the memory/state level; a future integration test suite can be gated behind a build tag.

**`eino-ext/claude` pulls in heavy AWS SDK dependencies** → The Claude provider transitively requires `github.com/aws/aws-sdk-go-v2` for Bedrock support even when unused. This bloats the binary by ~5MB. Mitigation: Acceptable for now; if binary size becomes a constraint on the Pi, we can write a thin direct adapter using `anthropics/anthropic-sdk-go` that satisfies `BaseChatModel`.

**context.Background() in router** → `localModel()` and `cloudModel()` call Eino constructors with `context.Background()` rather than the request context. Mitigation: Eino constructors don't actually do I/O on construction (Ollama and Claude both connect lazily), so this is safe. When constructors become async, pass the request context through.

## Open Questions

- Should `ConversationMemory` eventually support multiple named sessions (e.g., per-persona)? Current design assumes a single conversation file per binary invocation.
- Should the `ModelRouter` cache the last-used `ChatModel` instance across calls (for connection reuse)? Profiling needed on Pi.
- Phase 3 will need the `EventBus` to deliver `EventAgentTokens` to the TUI at ~50 tok/s. Current buffer size of 32 on the token channel may need tuning.
