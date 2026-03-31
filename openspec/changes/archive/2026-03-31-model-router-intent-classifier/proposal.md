## Why

Tool calling on Pi 5 is unreliable with every tested small model. The best performer (llama3.2:1b) only passes 6/9 golden cases at 67% accuracy, with 40% false-positive tool triggers on conversational prompts. No single off-the-shelf model can handle both conversation and tool calling well within the Pi 5's 8GB RAM constraint.

The long-term solution is a fine-tuned tool-selection model trained on DUSTY's exact tool schema — but that requires training data that doesn't exist yet. This change builds the foundations: a flexible classifier interface that starts rule-based, collects training data from every interaction, and is designed to be replaced by a fine-tuned model once sufficient examples exist. The architecture treats the rule-based classifier as bootstrap scaffolding, not a destination.

## What Changes

- Define a `Classifier` interface returning a rich `ClassifyResult` (bool gate + confidence + suggested tools) — designed to serve rule-based, ML, and fine-tuned model implementations without contract changes.
- Implement a rule-based `RuleClassifier` as the bootstrap implementation, targeting known failure modes from eval data.
- Add training data collection: log every `(user_message, classification, tools_called, success)` tuple to build the dataset for fine-tuning.
- Extend `ModelRouter` to support classifier-driven model selection — route tool intents to a tool-optimized model when configured.
- Add prompt engineering improvements to `MeshSystemPromptAddendum` with few-shot examples as an immediate reliability boost.
- Add a fine-tuning data pipeline: export collected training data in a format suitable for LoRA/QLoRA fine-tuning of small models.

## Capabilities

### New Capabilities
- `intent-classifier`: Classifier interface with `ClassifyResult` contract (ShouldUseTool, Confidence, SuggestedTools). Rule-based bootstrap implementation. Designed for seamless replacement by fine-tuned model.
- `prompt-engineering-tools`: Few-shot examples and improved system prompt addendum for mesh tool calling, targeting known failure modes (lowercase node names, broadcast intent, false positives on philosophical prompts).
- `training-data-collector`: Logs classification decisions and tool-call outcomes from every Chat() call. Exports training datasets for fine-tuning.

### Modified Capabilities
- `model-router`: Extend to support classifier-driven model selection — route based on ClassifyResult, not a fixed preference.
- `tool-calling-loop`: Gate tool schema injection on ClassifyResult.ShouldUseTool instead of the static SupportsToolCalling() allowlist.

## Impact

- **Code**: `internal/agent/router.go` (dynamic routing), `internal/agent/agent.go` (classifier integration + training data hooks), new `internal/agent/classifier/` package, new `internal/agent/training/` package
- **Config**: New `[inference.classifier]` section in TOML for classifier settings, model-per-intent mappings
- **Data**: New `data/training/` directory for collected training examples (JSON-lines format)
- **Dependencies**: None for Phase 1-2 (pure Go). Phase 3 fine-tuning uses external tooling (not a Go dependency).
- **Performance**: Adds <1ms classification step (rule-based) or <50ms (future ML). Training data logging is async, no latency impact.
- **Eval**: Existing `cmd/eval/` and `testdata/eval/tool-calling-cases.yaml` validate improvements. Training data collector also feeds back into eval coverage.
