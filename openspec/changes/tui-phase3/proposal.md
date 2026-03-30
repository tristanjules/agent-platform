## Why

DUSTY needed a rich terminal UI to replace the plain stdin REPL — making conversations visually engaging with real-time streaming, state visibility, and keyboard-driven controls. Phase 3 delivers the Bubble Tea v2 TUI described in the spec, giving the system a retro-terminal aesthetic suitable for Burning Man and everyday use.

## What Changes

- **New**: `internal/display/tui/` package with 9 files implementing the full TUI
- **New**: `--headless` flag on the `dusty` binary; REPL auto-activates when stdin is not a TTY
- **New**: `VoiceController` interface decoupling voice lifecycle from main.go
- **Modified**: `cmd/dusty/main.go` split into `runTUI()` / `runREPL()` paths
- **New dependencies**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`, `golang.org/x/term`

## Capabilities

### New Capabilities

- `tui-app`: Root Bubble Tea v2 application model — composes all sub-components, handles layout, keyboard shortcuts, and event routing
- `tui-theme`: Retro terminal theme system (phosphor-green, amber, blue) built with Lip Gloss v2
- `tui-event-bridge`: Bridge layer that forwards EventBus channel events into the Bubble Tea message loop via `tea.Program.Send()`
- `tui-conversation-pane`: Scrollable chat history viewport with real-time streaming token display and blinking cursor
- `tui-reasoning-pane`: Scrollable pane showing state transitions, dimmed token echo, and voice pipeline events
- `tui-input`: Text input component with Enter-to-submit and slash-command detection
- `tui-commands`: Slash command handler returning structured `CommandResult` (replaces stdout-printing handleCommand)
- `tui-settings`: F2 overlay panel for runtime switching of persona, model, and routing mode

### Modified Capabilities

- `conversational-agent`: No requirement changes — the agent's `Chat()` interface is unchanged. The TUI consumes tokens via EventBus rather than the returned channel directly, but agent behavior is identical.
- `event-bus`: No requirement changes — new subscribers added (TUI bridge), but the EventBus contract is unchanged.

## Impact

- `cmd/dusty/main.go` — substantially refactored; old REPL preserved as `runREPL()`
- `go.mod` / `vendor/` — 19 new packages added (Bubble Tea v2 ecosystem)
- No changes to agent, state machine, voice pipeline, or config packages
- All 18 existing tests continue to pass
