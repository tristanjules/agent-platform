# Eval Configurations

This document explains the three eval configurations, what each measures, how to interpret results, and how to run comparative evaluations.

## Config Matrix

| Config | Default Model | Classifier | Tool Model | Chat Model | What It Measures |
|--------|--------------|------------|------------|------------|------------------|
| `tool-test-gemma.toml` | gemma3:1b | OFF | — | — | Baseline. Gemma can't do tool calling. Expect 1/9 (the no-tool case). |
| `tool-test-qwen.toml` | qwen2.5:1.5b | ON | qwen2.5:1.5b | qwen2.5:1.5b | Qwen solo with classifier gating. Shows qwen's tool-call accuracy when the classifier pre-filters obvious conversation. |
| `tool-test-llama.toml` | llama3.2:1b | ON | qwen2.5:1.5b | llama3.2:1b | Two-model routing. Classifier routes tool intents to qwen, conversation to llama. Tests the full intent-routing pipeline. |

## What the Classifier Does (and Doesn't Do)

The classifier runs before the model sees the message. It decides:
- **ShouldUseTool=true** with high confidence: Route to `tool_model`, inject tool schemas.
- **ShouldUseTool=false** with high confidence: Route to `chat_model`, skip tool schemas entirely.
- **Low confidence** (below `confidence_threshold`): Route to default model with tools enabled. Let the model decide.

The classifier does NOT improve the model's ability to call tools. It only routes messages to the right model and avoids injecting tools when they're clearly not needed. If qwen2.5:1.5b can't handle `current_time` (it historically scores 0%), routing to qwen won't fix that.

## Interpreting Results

### Example: tool-test-llama.toml (two-model routing)

```
[1/9] PASS (1.00) Send DUSTY-B the message hello       ← qwen handles this (100%)
[2/9] FAIL (0.10) Tell dusty-b I'll be at the temple... ← qwen fails on informal phrasing
[8/9] FAIL (0.00) What time is it?                      ← qwen's known current_time failure
[9/9] PASS (1.00) What is the meaning of the burn?      ← llama handles this (no tools)
```

When you see failures, check which model actually ran the case:
- Tool cases (expected_tool non-empty) run on `tool_model` (qwen in this config)
- No-tool cases run on `chat_model` (llama in this config)
- Low-confidence cases run on the default model (llama3.2:1b)

The failures above are qwen's failures, not classifier failures. The classifier correctly routed to qwen — qwen just can't handle those cases.

### Training data check

After any eval run, inspect the training data:

```bash
cat data/training/tool_calls.jsonl | python3 -m json.tool --no-ensure-ascii | head -50
```

Each entry shows the classification decision, the model used, and the actual tools called. This is the raw material for fine-tuning.

## Running Comparative Evals

### Single model eval

```bash
go run ./cmd/eval/ --config configs/tool-test-llama.toml \
    --cases testdata/eval/tool-calling-cases.yaml \
    --runs 5 --out eval-llama.json
```

### All three configs

```bash
for cfg in gemma qwen llama; do
    go run ./cmd/eval/ \
        --config "configs/tool-test-${cfg}.toml" \
        --cases testdata/eval/tool-calling-cases.yaml \
        --runs 10 \
        --out "eval-${cfg}.json" 2>&1
    echo "---"
done
```

### After fine-tuning

Once you have a fine-tuned model deployed to Ollama:

```bash
# Update the tool-test-llama.toml (or create a new config):
# [inference.classifier]
# tool_model = "dusty-tool-caller"

go run ./cmd/eval/ --config configs/tool-test-llama.toml \
    --cases testdata/eval/tool-calling-cases.yaml \
    --runs 10 --out eval-finetuned.json
```

Compare with the baseline to measure improvement.

## Adding New Eval Cases

Edit `testdata/eval/tool-calling-cases.yaml`:

```yaml
- name: send_with_emoji
  prompt: "Send DUSTY-B a message: see you at the burn! 🔥"
  expected_tool: mesh_send
  min_success_rate: 0.70
  runs: 10
  notes: "Tests emoji handling in message body"
```

The `min_success_rate` is the threshold for PASS/FAIL. Set it based on what you consider acceptable for that case. Lower thresholds for harder cases.
