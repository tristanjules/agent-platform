## 1. Prompt Engineering (Phase 1 — immediate reliability improvements)

- [x] 1.1 Update `MeshSystemPromptAddendum` in `internal/agent/tools.go` with few-shot examples for mesh_send, mesh_inbox, and current_time
- [x] 1.2 Add negative examples to the prompt (no tools for philosophy, feelings, general knowledge)
- [x] 1.3 Add case-insensitive node name instruction to the prompt
- [x] 1.4 Add broadcast limitation guidance to the prompt
- [x] 1.5 Run eval suite against llama3.2:1b with updated prompt and compare to baseline

## 2. Classifier Interface & Rule-Based Bootstrap

- [x] 2.1 Create `internal/agent/classifier/classifier.go` with `Classifier` interface, `ClassifyResult` struct (ShouldUseTool, Confidence, SuggestedTools)
- [x] 2.2 Implement `RuleClassifier` in `internal/agent/classifier/rules.go` with keyword/pattern matching for mesh_send, mesh_inbox, current_time
- [x] 2.3 Implement confidence scoring: high (>=0.9) for strong keyword+context matches, medium (0.5-0.8) for partial matches, low for non-matches
- [x] 2.4 Add `ClassifierOpts` for customizable pattern lists with DUSTY defaults
- [x] 2.5 Write unit tests for RuleClassifier covering all 9 golden eval cases plus edge cases (case-insensitive names, philosophical questions, ambiguous coordination requests)

## 3. Training Data Collector

- [x] 3.1 Create `internal/agent/training/collector.go` with `TrainingCollector` struct — buffered channel writer, JSONL format
- [x] 3.2 Implement `Record()` method: accepts message, ClassifyResult, tools called, success, model name — non-blocking with drop-on-full semantics
- [x] 3.3 Implement `Close()` method: flush buffer, close file
- [x] 3.4 Write unit tests: verify JSONL format, non-blocking behavior, flush-on-close

## 4. Config Extension

- [x] 4.1 Add `ClassifierConfig` struct to `InferenceConfig` with `Enabled`, `ToolModel`, `ChatModel`, `ConfidenceThreshold`, `TrainingDataPath` fields
- [x] 4.2 Update `configs/default.toml` with commented-out `[inference.classifier]` section
- [x] 4.3 Add classifier routing config to `configs/tool-test-*.toml` presets for eval testing

## 5. Model Router — Classification-Based Routing

- [x] 5.1 Add `RouteWithClassification(ctx, ClassifyResult)` method to `ModelRouter` interface
- [x] 5.2 Implement classification-based model selection: tool_model for ShouldUseTool+high confidence, chat_model for !ShouldUseTool+high confidence, default model for low confidence
- [x] 5.3 Update `SupportsToolCalling()` to evaluate the tool model when classifier routing is enabled
- [x] 5.4 Ensure backward compatibility: `Route(ctx)` unchanged when classifier routing is disabled
- [x] 5.5 Write unit tests for classification-based routing with enabled/disabled config, confidence threshold edge cases

## 6. Agent Integration

- [x] 6.1 Add optional `Classifier` and `TrainingCollector` fields to `Agent` struct
- [x] 6.2 Wire classifier into `Agent.Chat()` — classify before routing, pass ClassifyResult to router
- [x] 6.3 Gate tool schema injection on ClassifyResult: skip tools when ShouldUseTool=false and Confidence >= threshold, include otherwise
- [x] 6.4 Wire training data collection: after Chat() completes, record message + classification + tools called + outcome
- [x] 6.5 Ensure nil classifier preserves current `SupportsToolCalling()` behavior
- [x] 6.6 Update `NewAgent()` to construct classifier and collector when `inference.classifier.enabled` is true
- [x] 6.7 Update `NewAgentWithRegistry()` to accept optional classifier for eval/test injection
- [x] 6.8 Update `Agent.Close()` to flush and close the TrainingCollector

## 7. Fine-Tuning Pipeline (scripts + docs)

- [x] 7.1 Create `scripts/generate_synthetic_data.py` — generate training examples from golden eval cases with varied phrasings
- [x] 7.2 Create `scripts/export_training_data.py` — convert collector JSONL + synthetic data to Llama 3 / Qwen 2.5 chat template format
- [x] 7.3 Create `docs/eval-configs.md` — document what each config tests, how to interpret results, how to run comparative evals
- [x] 7.4 Create `docs/fine-tuning-guide.md` — end-to-end guide from data collection through LoRA training to Ollama deployment

## 8. Eval Validation (requires live Ollama)

- [x] 8.1 Run full eval suite (9 golden cases) with prompt-only improvements (Phase 1)
- [x] 8.2 Run full eval suite with rule-based classifier + single model (no model routing)
- [x] 8.3 Run full eval suite with classifier + classification-based model routing (tool_model vs chat_model)
- [x] 8.4 Compare results across all three configurations and document accuracy improvements
- [x] 8.5 Verify training data JSONL output contains correct entries for all eval runs
