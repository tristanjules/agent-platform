## Context

DUSTY previously ran a blocking `bufio.Scanner` REPL in `main()`. Streaming LLM tokens were written directly to stdout with `fmt.Print`. All slash commands printed results to stdout. This worked but offered no visual structure, no real-time state visibility, and no keyboard shortcuts.

Bubble Tea v2 (released Feb 2026) provides a mature, production-grade TUI framework used by NVIDIA, GitHub, AWS, and Slack. Its Cursed Renderer (ncurses-based) offers 10x performance over v1 and native alt-screen support. The existing EventBus and StateMachine provide a clean integration surface.

## Goals / Non-Goals

**Goals:**
- Replace the REPL with a full-screen TUI when stdin is a TTY
- Show streaming tokens in real-time without blocking the event loop
- Display state transitions and agent activity in a dedicated reasoning pane
- Provide F1/F2/Ctrl+L keyboard shortcuts and settings overlay
- Preserve the REPL 100% for headless/piped use cases

**Non-Goals:**
- Visual/graphical mode (Phase 4, Ebiten)
- Mouse support
- Persistent TUI log / export
- Multiple concurrent conversations

## Decisions

### EventBus → Bubble Tea bridge via goroutines

**Decision:** Subscribe to EventBus channels in dedicated goroutines that call `tea.Program.Send(EventMsg{...})` to inject events into the Bubble Tea update loop.

**Rationale:** Bubble Tea is single-threaded by design. All model mutations must happen inside `Update()`. The EventBus uses Go channels, which live outside Bubble Tea. The bridge pattern is idiomatic — it's the same pattern Bubble Tea uses internally for its own I/O (e.g., `listenForActivity`). Each event type gets its own goroutine to avoid head-of-line blocking on slow consumers.

**Alternative considered:** Polling the EventBus inside a `tea.Tick` command. Rejected because it introduces latency proportional to tick interval and wastes CPU when idle.

### Token channel drained separately; EventBus carries display tokens

**Decision:** `agent.Chat()` returns a `<-chan string` that must be drained to avoid goroutine leaks. The TUI drains it silently in a goroutine; actual token display comes from `EventAgentTokens` events on the bus.

**Rationale:** The token channel and the EventBus carry the same data. Displaying from both would duplicate content. The EventBus path is already wired; the channel is a safety contract from the agent's perspective. `EventAgentResponse` (full text) serves as a reconciliation point if tokens are dropped.

### Headless detection via `term.IsTerminal()` + `--headless` flag

**Decision:** Auto-detect non-TTY stdin and fall back to REPL. `--headless` flag forces REPL regardless of terminal state.

**Rationale:** Preserves CI/script compatibility with zero user friction. Users piping input (`echo "hello" | dusty`) or running over SSH without a PTY get the plain REPL automatically.

### Settings overlay rendered inline (not a true overlay layer)

**Decision:** The settings panel appends below the main content in the `View()` string rather than using a true floating overlay.

**Rationale:** Bubble Tea v2's `View` struct supports layers but implementing a true positioned overlay adds significant complexity (z-ordering, coordinate transforms). The inline approach is simpler, ships in Phase 3, and can be upgraded to a proper overlay in a later pass. The settings panel is rarely open and the UX degradation is minor.

### `VoiceController` interface in `tui` package

**Decision:** Define `tui.VoiceController` as an interface with `Active() bool` and `ToggleVoice() string`. `main.voiceState` implements it.

**Rationale:** Decouples the TUI from `main.go`'s concrete voice struct. Enables testing the TUI with a mock voice controller. Avoids the `tui` package importing `main` (circular).

## Risks / Trade-offs

- **EventBus drop policy**: The bus drops events non-blocking when subscriber buffer is full. Under pathological LLM output (very fast tokens), the reasoning pane echo may drop tokens. The conversation pane reconciles on `EventAgentResponse`. → Acceptable: visible content (conversation) is guaranteed; reasoning echo is best-effort.
- **Alt-screen flicker on slow terminals**: Alt-screen mode clears and redraws the full terminal on each update. On very slow connections (serial, slow SSH), this may flicker. → Mitigation: Bubble Tea's Cursed Renderer uses differential rendering to minimize writes.
- **Width calculation with ANSI codes**: `RenderPane` uses `lipgloss.Width()` which correctly handles ANSI escape codes. If a third-party renderer produces non-standard ANSI, line padding could be off. → Low risk: all content goes through Lip Gloss styles.

## Migration Plan

The binary is backwards-compatible. Existing users see no change when running in a non-TTY context or with `--headless`. TUI activates automatically when a TTY is detected. No config changes required; `display.theme` already existed in `default.toml`.

## Open Questions

- Should the settings overlay support live model hot-swap (Ollama model list fetched at runtime)?  Currently it only shows the configured model; runtime Ollama model listing requires an async fetch.
- Should the reasoning pane be hidden by default (height=0) and toggled with a key? The current 25% split may feel cramped on small terminals.
