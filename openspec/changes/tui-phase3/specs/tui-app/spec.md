## ADDED Requirements

### Requirement: Full-screen TUI launches when stdin is a terminal
The system SHALL launch the Bubble Tea v2 full-screen TUI when the binary is invoked with a TTY stdin and `--headless` is not set. When stdin is not a TTY or `--headless` is specified, the system SHALL fall back to the plain REPL.

#### Scenario: TTY stdin launches TUI
- **WHEN** the binary is invoked with a TTY stdin and no `--headless` flag
- **THEN** the full-screen TUI renders with status bar, reasoning pane, conversation pane, input pane, and hotkey bar

#### Scenario: Non-TTY stdin uses REPL
- **WHEN** the binary is invoked with piped stdin (e.g. `echo "hello" | dusty`)
- **THEN** the plain REPL runs without attempting to render a TUI

#### Scenario: --headless flag forces REPL
- **WHEN** the binary is invoked with `--headless` even on a TTY
- **THEN** the plain REPL runs

### Requirement: Layout divides terminal into fixed zones
The TUI SHALL render a layout composed of: status bar (1 line, top), reasoning pane (≈25% of available height, bordered), conversation pane (≈75% of available height, bordered), input pane (3 lines, bordered), hotkey bar (1 line, bottom).

#### Scenario: Layout fills terminal dimensions
- **WHEN** a `WindowSizeMsg` is received
- **THEN** all panes are resized to fill the terminal width and the heights sum to the terminal height

#### Scenario: Minimum height graceful degradation
- **WHEN** terminal height is too small for all panes
- **THEN** conversation pane is allocated at least 5 lines and panes do not overlap

### Requirement: Global keyboard shortcuts
The TUI SHALL handle: `Ctrl+C` (quit + save memory), `Ctrl+L` (clear memory), `F1` (toggle voice), `F2` (toggle settings), `Tab` (reserved for visual mode, no-op with informational message).

#### Scenario: Ctrl+C saves memory and exits
- **WHEN** user presses `Ctrl+C`
- **THEN** `agent.Close()` is called, memory is persisted, and the program exits cleanly

#### Scenario: Ctrl+L clears memory
- **WHEN** user presses `Ctrl+L`
- **THEN** `agent.ClearMemory()` is called and a confirmation message appears in conversation pane

### Requirement: Settings overlay absorbs keys when visible
When the settings overlay is open, the TUI SHALL route all key events to the overlay and suppress global hotkeys.

#### Scenario: Global hotkeys suppressed while settings open
- **WHEN** the settings overlay is visible and user presses `Ctrl+L`
- **THEN** the key is handled by the settings overlay (or ignored), not the global handler
