## Context

DUSTY's current architecture uses a single model for all inference — conversation and tool calling. Eval results across three small models (gemma3:1b, qwen2.5:1.5b, llama3.2:1b) show that no single off-the-shelf model reliably handles both. The best performer (llama3.2:1b) achieves only 67% on 9 golden tool-calling cases and has a 40% false-positive rate on conversational prompts.

The Pi 5 has 8GB RAM, which cannot run two LLMs simultaneously. Ollama model swapping takes 1-3 seconds. The current `ModelRouter` is stateless per-call — it creates a new `BaseChatModel` on each `Route()` invocation and selects provider (Ollama vs Claude) based on a fixed `RoutingPreference`. Tool calling is gated by `SupportsToolCalling()`, a static prefix allowlist.

The end goal is a fine-tuned small model that handles tool selection reliably without needing a separate pre-classifier. Getting there requires training data that doesn't exist yet. This design builds the data pipeline and the abstraction layer that makes the transition seamless.

## Goals / Non-Goals

**Goals:**
- Build a `Classifier` interface whose contract supports rule-based, ML, and fine-tuned model implementations without changes to calling code
- Collect training data from every interaction to build the fine-tuning dataset
- Ship prompt engineering improvements immediately (measurable via eval)
- Ship rule-based classifier as bootstrap — good enough for 3 tools, self-aware that it won't scale
- Enable config-driven model-per-intent routing for A/B testing
- Design for the classifier to eventually be replaced by a fine-tuned model that doesn't need a pre-classifier at all

**Non-Goals:**
- Building the fine-tuning pipeline itself (external tooling, not a Go dependency)
- Supporting simultaneous multi-model inference (hardware constraint)
- Building a general-purpose NLU pipeline — the interface is narrow and purpose-built
- Achieving >95% accuracy in this change — that's the fine-tuned model's job

## Decisions

### Decision 1: Rich ClassifyResult instead of enum taxonomy

**Choice**: The Classifier returns a `ClassifyResult` struct, not a 3-value enum:

```go
type Classifier interface {
    Classify(ctx context.Context, message string) ClassifyResult
}

type ClassifyResult struct {
    ShouldUseTool  bool      // the binary gate — only thing the agent must act on
    Confidence     float64   // 0.0–1.0; enables confidence-based routing thresholds
    SuggestedTools []string  // optional hint: which tools seem relevant (empty = let model decide)
}
```

**Alternatives considered**:
- *Three-intent enum (tool_call/conversation/ambiguous)*: Too coarse. Bakes in a taxonomy that doesn't scale past 3 tools. "Ambiguous" becomes a dumping ground as tool surface grows. The enum also can't express "I think this needs mesh_send specifically" — just "this needs some tool."
- *Full tool-call prediction (return tool name + args)*: Too ambitious for the bootstrap phase, and that's actually what the fine-tuned model will do. The interface should accommodate it (via SuggestedTools) without requiring it.

**Rationale**: `ShouldUseTool` is the only decision the agent truly needs from the classifier. `Confidence` enables future behaviors (low-confidence → ask the user, high-confidence → skip model deliberation). `SuggestedTools` lets a fine-tuned classifier eventually say "this is a mesh_send" without changing the interface. A rule-based classifier can populate all three fields. An ML classifier adds better confidence. A fine-tuned model adds accurate SuggestedTools. Same interface throughout.

### Decision 2: Rule-based classifier as bootstrap, not destination

**Choice**: Ship a `RuleClassifier` that handles the known 3-tool schema with keyword/pattern matching. Treat it explicitly as temporary scaffolding.

**What makes it scaffolding, not architecture**:
- The `Classifier` interface doesn't mention rules, patterns, or keywords
- No code outside the classifier package knows or cares that it's rule-based
- The rule-based implementation lives in its own file, easy to delete
- Training data collection starts from day one, building toward its replacement

**Rationale**: A rule-based classifier at 3 tools is fine. At 10+ tools the combinatorial pattern overlap makes it unmaintainable. The interface is designed so the rule-based implementation can be swapped for a fine-tuned model with zero changes to agent, router, or config.

### Decision 3: Training data collection from day one

**Choice**: Every `Chat()` call logs a training example:

```json
{"timestamp": "...", "message": "Send DUSTY-B hello", "classification": {"should_use_tool": true, "confidence": 0.95, "suggested_tools": ["mesh_send"]}, "tools_called": ["mesh_send"], "tool_success": true, "model": "llama3.2:1b"}
```

Stored as JSON-lines in a configurable path (default `data/training/tool_calls.jsonl`). Async write via a buffered channel — zero latency impact on Chat().

**Alternatives considered**:
- *Log only when tools are called*: Misses negative examples (conversation correctly classified as no-tool). You need both positive and negative examples for fine-tuning.
- *Log to structured DB (SQLite)*: Overkill for append-only training data. JSONL is simpler, portable, and directly consumable by fine-tuning scripts.
- *Don't collect until fine-tuning is imminent*: The most expensive part of fine-tuning is data collection. Starting late means waiting longer for a usable dataset. Every conversation on the playa is a training example.

**Rationale**: The training data collector is the most important piece of this change for the long-term goal. A fine-tuned model is only as good as its training set. Burning Man conversations are irreplaceable domain-specific data — you can't recreate "tell dusty-b I'll be at the temple of the inner flame at sunset" in a lab.

### Decision 4: Prompt engineering as Phase 1 (unchanged)

**Choice**: Before building the classifier, improve `MeshSystemPromptAddendum` with few-shot tool-call examples targeting known failure modes.

**What changes in the prompt**:
- Few-shot examples for each tool (mesh_send, mesh_inbox, current_time)
- Explicit instruction to match node names case-insensitively
- Negative examples: "Do NOT call any tool for questions about philosophy, feelings, or general knowledge"
- Explicit guidance for broadcast-like intents (no broadcast tool exists — respond with text)

**Rationale**: Prompt improvements help every phase — they make the model better at tool calling regardless of whether a classifier is present. They also establish baseline improvement metrics before adding classifier complexity. Even a fine-tuned model benefits from good system prompts.

### Decision 5: Config-driven model-per-classification mapping

**Choice**: Add optional `[inference.classifier]` config section:

```toml
[inference.classifier]
enabled = true
tool_model = "qwen2.5:1.5b"      # used when ShouldUseTool=true
chat_model = "llama3.2:1b"       # used when ShouldUseTool=false
confidence_threshold = 0.7        # below this, treat as ambiguous (use default model with tools)
training_data_path = "data/training/tool_calls.jsonl"
```

When `enabled = false` or absent, the router uses the single `inference.local.model` for everything (current behavior). The `confidence_threshold` enables a soft boundary: high-confidence classifications route to specialized models, low-confidence falls back to the default model with tools enabled.

**Rationale**: The confidence threshold is critical for the transition path. A rule-based classifier can set confidence to 1.0 or 0.0 (it's sure or it's not). An ML classifier produces real probabilities. A fine-tuned model might always return high confidence once trained. The threshold lets you tune the aggression of routing without changing code.

### Decision 6: Classifier in Agent, routing result passed to Router

**Choice**: The `Classifier` is called in `Agent.Chat()` before routing. The `ClassifyResult` is passed to a new `RouteWithClassification(ctx, result)` method so the router can select the appropriate model.

**Rationale**: Agent already decides whether to pass tools to the model (line 140 of agent.go). Classification is a refinement of that same decision. The router doesn't need to understand messages — it just needs to know which model to load based on the classification.

### Decision 7: Package structure

```
internal/agent/classifier/
├── classifier.go       # Classifier interface, ClassifyResult, constructors
├── rules.go            # RuleClassifier implementation (bootstrap)
└── rules_test.go       # Tests against golden eval cases

internal/agent/training/
├── collector.go        # TrainingCollector — async JSONL logger
└── collector_test.go
```

**Rationale**: Classifier and training are separate concerns. The classifier doesn't know about training data. The training collector observes classifications and outcomes from the agent level. Both packages have no dependencies on agent internals — they operate on plain strings and structs.

## Risks / Trade-offs

**[Risk] Rule-based classifier hits ceiling at 3 tools** → This is expected and by design. The rule-based classifier is scaffolding. The `Classifier` interface and training data pipeline are the real deliverables. When accuracy plateaus, the fine-tuned model takes over.

**[Risk] Training data quality depends on classifier accuracy** → The collector logs what actually happened (tools called, success), not just what the classifier predicted. Even wrong classifications produce useful training data because the ground truth (did a tool get called? did it succeed?) is captured independently.

**[Risk] Ollama model swap latency (1-3s) on classification transitions** → The classifier reduces unnecessary swaps by routing pure conversation away from the tool model entirely. The confidence threshold prevents flip-flopping on borderline cases. Future: a fine-tuned single model eliminates swapping entirely.

**[Risk] JSONL training data grows unbounded** → Mitigation: configurable max file size with rotation. Playa deployment is ~1 week, roughly 500-2000 interactions — well under any storage concern. Post-event, data is exported and the local file is cleared.

**[Risk] Fine-tuned model never materializes** → Even without fine-tuning, the rule-based classifier + prompt engineering is a strict improvement over the current static allowlist. The training data is also valuable for expanding the eval suite with real-world examples. Nothing is wasted.

### Decision 8: Fine-tuning pipeline (scripts, not Go)

**Choice**: The fine-tuning workflow lives in `scripts/` as Python, not in the Go codebase. Two scripts:

1. `scripts/generate_synthetic_data.py` — generates training examples from the golden eval cases and the MeshSystemPromptAddendum, producing both tool-call and no-tool examples with varied phrasings. Outputs raw JSONL.

2. `scripts/export_training_data.py` — reads the collector JSONL (`data/training/tool_calls.jsonl`) and/or synthetic JSONL and converts to the target model's function-calling chat template format (Llama 3, Qwen 2.5). Outputs a dataset ready for Unsloth/axolotl.

**Training data flow**:
```
Golden eval cases (YAML)       Real conversations (JSONL)
         │                              │
         ▼                              ▼
generate_synthetic_data.py     data/training/tool_calls.jsonl
         │                              │
         └──────────┬───────────────────┘
                    ▼
         export_training_data.py
                    │
                    ▼
         training_data_llama.jsonl  (Llama 3 chat template)
         training_data_qwen.jsonl   (Qwen 2.5 chat template)
                    │
                    ▼
         Unsloth LoRA fine-tuning  (external, on Mac/GPU)
                    │
                    ▼
         dusty-tool-caller.gguf
                    │
                    ▼
         ollama create dusty-tool-caller -f Modelfile
                    │
                    ▼
         [inference.classifier] tool_model = "dusty-tool-caller"
```

**Alternatives considered**:
- *Go-based training pipeline*: Python has the best tooling for ML (Unsloth, transformers, datasets). Adding Python ML deps to a Go project is wrong.
- *Single script*: Separating synthetic generation from export keeps concerns clean. You can regenerate synthetic data without re-exporting real data and vice versa.

**Rationale**: The scripts are the bridge between "collecting data" (Go, running on Pi) and "training a model" (Python, running on a workstation). They don't need to be fast or deployable — they're development tooling.

### Decision 9: Documentation as operational guides

**Choice**: Create `docs/` with two operational guides:

1. `docs/eval-configs.md` — explains what each config tests, how to interpret eval results, and how to run comparative evals.
2. `docs/fine-tuning-guide.md` — end-to-end guide from data collection through LoRA training to Ollama deployment.

**Rationale**: The fine-tuning pipeline has many steps across multiple tools. Without documentation, the knowledge lives in one person's head. The docs are the difference between "we could fine-tune" and "anyone on the team can fine-tune."
