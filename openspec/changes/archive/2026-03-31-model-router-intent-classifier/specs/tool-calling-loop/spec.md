## MODIFIED Requirements

### Requirement: Agent passes tool schemas to the model on each inference call
The agent SHALL pass all registered tools as `schema.ToolInfo` slices to the model via `model.WithTools()` on Generate/Stream calls based on the classifier result. When `ClassifyResult.ShouldUseTool` is true (at any confidence), tools SHALL be injected. When `ShouldUseTool` is false and `Confidence >= confidence_threshold`, tools SHALL NOT be injected. When `Confidence < confidence_threshold`, tools SHALL be injected as a conservative fallback. When no classifier is configured, the existing `SupportsToolCalling()` gate applies.

#### Scenario: High-confidence tool classification includes tool schemas
- **WHEN** `Chat()` is called with tools registered and classifier returns `ShouldUseTool: true, Confidence: 0.95`
- **THEN** the model's Generate/Stream call includes `WithTools()` containing all registered tool schemas

#### Scenario: High-confidence conversation classification skips tool schemas
- **WHEN** `Chat()` is called with tools registered and classifier returns `ShouldUseTool: false, Confidence: 0.95`
- **THEN** the model's Generate/Stream call does NOT include `WithTools()`

#### Scenario: Low-confidence classification includes tool schemas as fallback
- **WHEN** `Chat()` is called with tools registered and classifier returns `Confidence: 0.4` (below threshold)
- **THEN** the model's Generate/Stream call includes `WithTools()` regardless of `ShouldUseTool` value

#### Scenario: No classifier configured preserves current behavior
- **WHEN** `Chat()` is called with no classifier configured (nil)
- **THEN** tool schema injection uses the existing `SupportsToolCalling()` gate (current behavior)

## ADDED Requirements

### Requirement: Agent accepts an optional Classifier
The `Agent` struct SHALL accept an optional `Classifier` via constructor or setter. When present, `Chat()` SHALL classify the user message before routing. When absent, the agent SHALL behave identically to the current implementation.

#### Scenario: Agent with classifier
- **WHEN** `NewAgent()` is called with a classifier configured
- **THEN** the agent stores the classifier and uses it in `Chat()`

#### Scenario: Agent without classifier
- **WHEN** `NewAgent()` is called without a classifier (nil or not configured)
- **THEN** `Chat()` skips classification and uses `SupportsToolCalling()` as before
