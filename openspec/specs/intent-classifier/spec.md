# intent-classifier Specification

## Purpose
Classifies user messages to determine whether they require tool invocation, enabling the model router to select the appropriate model and the tool-calling loop to decide whether to inject tool schemas. Provides a pluggable interface with a rule-based default implementation.

## Requirements
### Requirement: Classifier returns a ClassifyResult struct
The `Classifier` SHALL be defined as a Go interface with a single method: `Classify(ctx context.Context, message string) ClassifyResult`. The `ClassifyResult` struct SHALL contain:
- `ShouldUseTool bool` — binary gate indicating whether tools should be injected for this message
- `Confidence float64` — value between 0.0 and 1.0 indicating classification certainty
- `SuggestedTools []string` — optional list of tool names the classifier believes are relevant (empty means let the model decide)

Implementations (rule-based, ML-based, fine-tuned model) SHALL be interchangeable without changing calling code.

#### Scenario: Interface contract with rule-based implementation
- **WHEN** a `RuleClassifier` is created
- **THEN** it satisfies the `Classifier` interface by implementing `Classify(ctx, string) ClassifyResult`

#### Scenario: High-confidence tool classification
- **WHEN** Classify is called with "Send DUSTY-B the message hello"
- **THEN** the returned ClassifyResult has `ShouldUseTool: true`, `Confidence >= 0.9`, and `SuggestedTools` containing "mesh_send"

#### Scenario: High-confidence conversation classification
- **WHEN** Classify is called with "What is the meaning of the burn?"
- **THEN** the returned ClassifyResult has `ShouldUseTool: false`, `Confidence >= 0.9`, and empty `SuggestedTools`

#### Scenario: Low-confidence classification
- **WHEN** Classify is called with "Let the other agents know I'm heading to the temple"
- **THEN** the returned ClassifyResult has `Confidence < 0.7` (below the default confidence threshold)

### Requirement: Rule-based classifier matches tool-related patterns
The `RuleClassifier` implementation SHALL use keyword and pattern matching to identify tool intents. It SHALL recognize:
- Mesh send patterns: "send", "tell", "message" combined with node name patterns (DUSTY-X, case-insensitive)
- Inbox patterns: "inbox", "unread", "messages from", "message history"
- Time patterns: "what time", "current time"

High-confidence matches (strong keyword + node name or clear tool keyword) SHALL set `Confidence >= 0.9`. Partial matches SHALL set `Confidence` between 0.5 and 0.8. Non-matches SHALL set `ShouldUseTool: false`.

#### Scenario: Case-insensitive node name matching
- **WHEN** Classify is called with "tell dusty-b I'll be late"
- **THEN** `ShouldUseTool` is true, `Confidence >= 0.9`, and `SuggestedTools` contains "mesh_send"

#### Scenario: Inbox query detection
- **WHEN** Classify is called with "Do I have any unread messages?"
- **THEN** `ShouldUseTool` is true, `Confidence >= 0.9`, and `SuggestedTools` contains "mesh_inbox"

#### Scenario: Time query detection
- **WHEN** Classify is called with "What time is it?"
- **THEN** `ShouldUseTool` is true, `Confidence >= 0.9`, and `SuggestedTools` contains "current_time"

#### Scenario: Philosophical question avoids tool classification
- **WHEN** Classify is called with "What is the meaning of life?"
- **THEN** `ShouldUseTool` is false and `Confidence >= 0.9`

#### Scenario: Ambiguous input returns low confidence
- **WHEN** Classify is called with "Can you help coordinate with the others?"
- **THEN** `Confidence < 0.7`

### Requirement: Classifier is configurable via constructor
`NewRuleClassifier()` SHALL accept an options struct that allows customizing pattern lists. Default patterns SHALL cover the DUSTY mesh tool set (mesh_send, mesh_inbox, current_time). A nil options argument SHALL use defaults.

#### Scenario: Default construction
- **WHEN** `NewRuleClassifier(nil)` is called
- **THEN** a classifier with default DUSTY tool patterns is returned

#### Scenario: Custom patterns
- **WHEN** `NewRuleClassifier(&ClassifierOpts{ToolPatterns: custom})` is called
- **THEN** the classifier uses the provided patterns instead of defaults
