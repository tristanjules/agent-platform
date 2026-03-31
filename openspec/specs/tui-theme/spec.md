## ADDED Requirements

### Requirement: Named theme lookup
The system SHALL provide a `ThemeByName(name string) Theme` function returning the named theme. Unknown names SHALL default to `phosphor-green`.

#### Scenario: Known theme returned
- **WHEN** `ThemeByName("amber")` is called
- **THEN** a Theme with amber color palette is returned

#### Scenario: Unknown name defaults to phosphor-green
- **WHEN** `ThemeByName("unknown")` is called
- **THEN** the phosphor-green theme is returned

### Requirement: Three built-in themes
The system SHALL provide phosphor-green (#33ff33 on #0a0a0a), amber (#ffb000 on #0a0800), and blue (#00aaff on #00080a) themes.

#### Scenario: Phosphor-green theme colors
- **WHEN** the phosphor-green theme is loaded
- **THEN** `Primary` foreground is #33ff33, `Accent` foreground is #88ff88, background is #0a0a0a

#### Scenario: Theme applied from config
- **WHEN** `display.theme = "amber"` is set in the config
- **THEN** the TUI renders with the amber color palette

### Requirement: Theme struct covers all UI zones
Each Theme SHALL expose styles for: Primary, Dimmed, Accent, UserLabel, AgentLabel, Border, StatusBar, HotkeyBar, HotkeyKey, Input, Error, Cursor.

#### Scenario: All style fields populated
- **WHEN** any theme is loaded via ThemeByName
- **THEN** all Theme fields contain non-zero Lip Gloss styles
