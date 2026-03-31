## 1. Dependencies

- [x] 1.1 Add `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`, `golang.org/x/term` to go.mod via `go get`
- [x] 1.2 Resolve transitive dependency `github.com/atotto/clipboard` and run `go mod vendor`

## 2. Theme System

- [x] 2.1 Create `internal/display/tui/theme.go` with `Theme` struct and `ThemeByName()` function
- [x] 2.2 Implement phosphor-green, amber, and blue theme palettes using Lip Gloss v2 styles
- [x] 2.3 Define all required style fields: Primary, Dimmed, Accent, UserLabel, AgentLabel, Border, StatusBar, HotkeyBar, HotkeyKey, Input, Error, Cursor

## 3. EventBus Bridge

- [x] 3.1 Create `internal/display/tui/bridge.go` with `EventMsg` struct and `StartBridge()` function
- [x] 3.2 Subscribe to all 8 display-relevant event types, one goroutine per type
- [x] 3.3 Verify goroutines exit cleanly when EventBus is closed

## 4. Input Component

- [x] 4.1 Create `internal/display/tui/input.go` wrapping `bubbles/v2/textinput`
- [x] 4.2 Implement Enter-to-submit emitting `SubmitMsg{Text}` and clearing the field
- [x] 4.3 Expose `SetWidth()` method for terminal resize handling

## 5. Reasoning Pane

- [x] 5.1 Create `internal/display/tui/reasoning.go` wrapping `bubbles/v2/viewport`
- [x] 5.2 Handle `EventStateChanged` → append `[From → To]` line in Dimmed style
- [x] 5.3 Handle `EventAgentTokens` → echo token in Dimmed style
- [x] 5.4 Handle `EventSTTResult`, `EventTTSStarted`, `EventTTSDone` → annotated lines
- [x] 5.5 Cap line history at 200 entries; auto-scroll to bottom on append

## 6. Conversation Pane

- [x] 6.1 Create `internal/display/tui/conversation.go` wrapping `bubbles/v2/viewport`
- [x] 6.2 Implement `AddUserMessage()`, `AppendToken()`, `FinalizeResponse()`, `AddSystemMessage()`
- [x] 6.3 Show streaming `▌` cursor during response; remove on `FinalizeResponse()`
- [x] 6.4 Reconcile with full response text from `EventAgentResponse`
- [x] 6.5 Auto-scroll to bottom on all content updates

## 7. Views and Layout Helpers

- [x] 7.1 Create `internal/display/tui/views.go` with `RenderStatusBar()` showing agent name, version, mode, model, and state
- [x] 7.2 Implement `RenderHotkeyBar()` showing F1/F2/Ctrl+L/Ctrl+C hints with voice state indicator
- [x] 7.3 Implement `RenderPane()` with box-drawing titled border

## 8. Command Handler

- [x] 8.1 Create `internal/display/tui/commands.go` with `CommandResult` struct and `CommandHandler`
- [x] 8.2 Implement all commands: /clear, /persona, /model, /mode, /status, /voice, /help, /quit
- [x] 8.3 Return structured `CommandResult` instead of printing to stdout
- [x] 8.4 Define `VoiceController` interface with `Active()` and `ToggleVoice()` methods

## 9. Settings Overlay

- [x] 9.1 Create `internal/display/tui/settings.go` with `SettingsOverlay` struct
- [x] 9.2 Implement Persona and Mode sections with list navigation (↑↓ select, Tab next section)
- [x] 9.3 Apply selection on Enter, calling agent methods and updating cfg
- [x] 9.4 Toggle visibility on F2/Esc

## 10. Root App Model

- [x] 10.1 Create `internal/display/tui/app.go` implementing Bubble Tea `Model` interface
- [x] 10.2 Implement `relayout()` splitting height into reasoning (25%) and conversation (75%)
- [x] 10.3 Wire `EventMsg` handling to update reasoning, conversation, and state display
- [x] 10.4 Route `SubmitMsg` to command handler (slash prefix) or `agent.Chat()` (plain text)
- [x] 10.5 Enable alt-screen by setting `v.AltScreen = true` on returned `tea.View`
- [x] 10.6 Drain agent token channel in background goroutine (tokens consumed via EventBus)

## 11. Main Entry Point

- [x] 11.1 Add `--headless` flag and `term.IsTerminal()` detection to `cmd/dusty/main.go`
- [x] 11.2 Implement `runTUI()` creating Bubble Tea program, starting bridge, running program
- [x] 11.3 Extract existing REPL into `runREPL()` preserving all existing behavior
- [x] 11.4 Make `voiceState` implement `tui.VoiceController` interface with exported `Active()` and `ToggleVoice()` methods

## 12. Verification

- [x] 12.1 `go build ./...` succeeds with no errors
- [x] 12.2 `go build -tags "noaudio nostt notts" ./...` succeeds (Pi stub builds)
- [x] 12.3 `go test ./...` passes all 18 existing tests with no regressions
