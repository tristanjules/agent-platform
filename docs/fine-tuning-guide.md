# Fine-Tuning a Tool-Calling Model for DUSTY

End-to-end guide: collecting data on the Pi, generating synthetic examples, training a LoRA adapter, and deploying the fine-tuned model back to Ollama.

## Overview

```
Pi 5 (data collection)          Workstation (training)         Pi 5 (deployment)
─────────────────────           ──────────────────────         ─────────────────
DUSTY runs with                 Unsloth + LoRA fine-tunes      Ollama serves the
training collector ON    →      on collected + synthetic    →   fine-tuned GGUF
                                data                           model
data/training/                  training_data_llama3.jsonl     dusty-tool-caller
  tool_calls.jsonl              dusty-tool-caller.gguf
  synthetic.jsonl
```

## Prerequisites

On your training workstation (Mac M-series or Linux with GPU):

```bash
pip install unsloth transformers datasets pyyaml trl
```

On the Pi 5:

- Ollama installed and running
- DUSTY built with classifier enabled

## Step 1: Collect Real Training Data

Enable the classifier and training data collector in your config:

```toml
[inference.classifier]
enabled = true
tool_model = "qwen2.5:1.5b"
chat_model = "llama3.2:1b"
confidence_threshold = 0.7
training_data_path = "data/training/tool_calls.jsonl"
```

Run DUSTY normally. Every conversation writes a JSONL entry:

```json
{"ts":"2026-08-25T22:15:00Z","msg":"tell dusty-b meet at the temple","classify":{"should_use_tool":true,"confidence":0.95,"suggested_tools":["mesh_send"]},"tools_called":["mesh_send"],"tool_success":true,"model":"qwen2.5:1.5b"}
```

After a session (or after Burning Man), copy `data/training/tool_calls.jsonl` to your workstation.

**How much data do you need?**

- Minimum: 50 examples (30 tool-call + 20 no-tool). Expect modest improvement.
- Good: 150+ examples. Clear improvement on failure modes.
- Excellent: 500+. Approaches the model's ceiling.

The synthetic generator helps bootstrap before you have real data.

## Step 2: Generate Synthetic Training Data

On your workstation:

```bash
python3 scripts/generate_synthetic_data.py \
    --variations 20 \
    --out data/training/synthetic.jsonl
```

This generates ~100 examples (20 per tool + 40 no-tool) with varied phrasings. The synthetic data covers:

- `mesh_send`: all node name capitalizations, varied verbs (send/tell/message/ping/contact)
- `mesh_inbox`: unread checks, history queries, peer lookups
- `current_time`: direct and informal phrasings
- No-tool: philosophy, feelings, burn culture, general knowledge

Inspect the output:

```bash
head -5 data/training/synthetic.jsonl | python3 -m json.tool
```

## Step 3: Export to Model Format

Convert to the target model's function-calling chat template:

```bash
# For Llama 3.2 fine-tuning:
python3 scripts/export_training_data.py \
    --input data/training/synthetic.jsonl data/training/tool_calls.jsonl \
    --format llama3 \
    --out data/training/training_data_llama3.jsonl

# For Qwen 2.5 fine-tuning:
python3 scripts/export_training_data.py \
    --input data/training/synthetic.jsonl \
    --format qwen25 \
    --out data/training/training_data_qwen25.jsonl
```

Note: real collector data (`tool_calls.jsonl`) that involved tool calls is skipped by default because the exact arguments the model used aren't recorded. Use `--include-real` to include them with inferred defaults. Synthetic data has exact arguments and is always included.

## Step 4: Fine-Tune with Unsloth

Create `scripts/train_lora.py` (or run interactively in a notebook):

```python
from unsloth import FastLanguageModel
from trl import SFTTrainer
from transformers import TrainingArguments
from datasets import load_dataset

# 1. Load base model (4-bit quantized for memory efficiency)
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name="unsloth/Llama-3.2-1B-Instruct",
    max_seq_length=2048,
    load_in_4bit=True,
)

# 2. Apply LoRA adapters
model = FastLanguageModel.get_peft_model(
    model,
    r=16,                # LoRA rank — 16 is good for small models
    target_modules=[
        "q_proj", "k_proj", "v_proj", "o_proj",
        "gate_proj", "up_proj", "down_proj",
    ],
    lora_alpha=16,
    lora_dropout=0,
    use_gradient_checkpointing="unsloth",
)

# 3. Load training data
dataset = load_dataset(
    "json",
    data_files="data/training/training_data_llama3.jsonl",
    split="train",
)

# 4. Format for SFTTrainer
# The training data is already in messages format.
# Unsloth handles the chat template application.
def format_example(example):
    return tokenizer.apply_chat_template(
        example["messages"],
        tools=example.get("tools"),
        tokenize=False,
    )

dataset = dataset.map(lambda x: {"text": format_example(x)})

# 5. Train
trainer = SFTTrainer(
    model=model,
    tokenizer=tokenizer,
    train_dataset=dataset,
    dataset_text_field="text",
    max_seq_length=2048,
    args=TrainingArguments(
        output_dir="./lora-output",
        per_device_train_batch_size=2,
        gradient_accumulation_steps=4,
        num_train_epochs=3,
        learning_rate=2e-4,
        warmup_steps=5,
        logging_steps=1,
        fp16=True,
        save_strategy="epoch",
    ),
)

trainer.train()

# 6. Export as GGUF for Ollama
model.save_pretrained_gguf(
    "dusty-tool-caller",
    tokenizer,
    quantization_method="q4_k_m",  # Good balance of quality and size
)
print("Exported to dusty-tool-caller/")
```

**Training time estimates (M-series Mac):**

- 50 examples, 3 epochs: ~10 minutes
- 150 examples, 3 epochs: ~30 minutes
- 500 examples, 3 epochs: ~90 minutes

## Step 5: Deploy to Ollama

Copy the GGUF file to the Pi 5 and create an Ollama model:

```bash
# On the Pi 5 (or wherever Ollama runs):
cat > Modelfile << 'EOF'
FROM ./dusty-tool-caller-Q4_K_M.gguf
PARAMETER temperature 0.1
PARAMETER num_ctx 2048
EOF

ollama create dusty-tool-caller -f Modelfile
```

Verify it loads:

```bash
ollama run dusty-tool-caller "What time is it?"
```

## Step 6: Update Config and Eval

Update your config to use the fine-tuned model:

```toml
[inference.local]
model = "llama3.2:1b"  # default for general use

[inference.classifier]
enabled = true
tool_model = "dusty-tool-caller"  # your fine-tuned model
chat_model = "llama3.2:1b"
confidence_threshold = 0.7
training_data_path = "data/training/tool_calls.jsonl"
```

Run the eval:

```bash
go run ./cmd/eval/ --config configs/tool-test-llama.toml \
    --cases testdata/eval/tool-calling-cases.yaml \
    --runs 10 --out eval-finetuned.json
```

Compare with baseline:

```bash
# Before fine-tuning (qwen2.5:1.5b as tool model):
# 3/9 passed (33%)

# After fine-tuning (dusty-tool-caller as tool model):
# Target: 8/9 passed (89%+)
```

## Step 7: Iterate

The fine-tuned model's failures become your next training data:

1. Run the eval, note which cases fail
2. Add more synthetic variations targeting those failures
3. Combine with real playa data from the collector
4. Re-export, re-train, re-deploy
5. Repeat until you hit your accuracy target

Each iteration should improve because you're training on exactly the failure modes the model struggles with.

## Long-Term: Replacing the Classifier

Once the fine-tuned model reliably handles all 9 golden cases at >90%:

1. The rule-based classifier becomes unnecessary for tool/no-tool routing
2. You can simplify to a single model for everything:
  ```toml
   [inference.local]
   model = "dusty-tool-caller"

   [inference.classifier]
   enabled = false  # model is good enough on its own
  ```
3. Keep the training data collector running — continued data collection improves future fine-tuning rounds
4. The `Classifier` interface stays in the codebase as an option for future tool-surface expansion

## Troubleshooting

**"CUDA out of memory" during training**: Use `load_in_4bit=True` (already in the script). If still failing, reduce `per_device_train_batch_size` to 1.

**Model generates garbled tool calls after fine-tuning**: Check that the chat template in `export_training_data.py` matches the base model's expected format. Llama 3 and Qwen 2.5 have slightly different tool-call token sequences.

**Fine-tuned model is worse than base**: Usually means too few training examples or too many epochs (overfitting). Try reducing to 1-2 epochs, or add more diverse examples.

**Ollama rejects the GGUF**: Ensure the quantization method matches what Ollama supports. `q4_k_m` is the safest choice. Check `ollama --version` for compatibility.