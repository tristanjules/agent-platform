## ADDED Requirements

### Requirement: F2 toggles settings overlay visibility
Pressing F2 SHALL show the settings overlay if hidden, or hide it if shown.

#### Scenario: F2 opens overlay
- **WHEN** settings overlay is hidden and user presses F2
- **THEN** the overlay becomes visible in the next render

#### Scenario: F2 closes overlay
- **WHEN** settings overlay is visible and user presses F2
- **THEN** the overlay is hidden

#### Scenario: Esc closes overlay
- **WHEN** settings overlay is visible and user presses Esc
- **THEN** the overlay is hidden

### Requirement: Arrow keys navigate options within a section
While the settings overlay is visible, up/down arrows SHALL move selection within the active section; left/right arrows or Tab SHALL switch between sections.

#### Scenario: Down arrow moves selection
- **WHEN** overlay is visible, active section has 3 options, and user presses down
- **THEN** the selection index increments (wraps at bottom)

#### Scenario: Tab switches section
- **WHEN** overlay is visible and user presses Tab
- **THEN** the active section cycles to the next section

### Requirement: Enter applies the highlighted selection immediately
Pressing Enter on a highlighted option SHALL apply it immediately (calling the appropriate agent method and updating cfg), and return a CommandResult with SettingsChanged=true.

#### Scenario: Persona selection applied
- **WHEN** "companion" is highlighted in the Persona section and user presses Enter
- **THEN** agent.SetPersona("companion") is called and cfg.Agent.Personality = "companion"

#### Scenario: Mode selection applied
- **WHEN** "cloud" is highlighted in the Mode section and user presses Enter
- **THEN** agent.SetRoutingPreference(PreferCloud) is called and cfg.Inference.Mode = "cloud"

### Requirement: Sections cover Persona and Routing Mode
The settings overlay SHALL expose sections for Persona (listing all available personas) and Mode (local, cloud, auto).

#### Scenario: All personas listed
- **WHEN** settings overlay is opened
- **THEN** the Persona section lists all values returned by agent.ListPersonas()
