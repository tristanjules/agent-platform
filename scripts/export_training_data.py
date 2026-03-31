#!/usr/bin/env python3
"""Export DUSTY training data to model-specific function-calling format.

Reads JSONL training data (from the collector and/or synthetic generator)
and converts it to the chat-template format required by specific models
for LoRA fine-tuning with Unsloth or axolotl.

Supported formats:
  --format llama3   Llama 3.x function-calling chat template
  --format qwen25   Qwen 2.5 function-calling chat template

Usage:
    python scripts/export_training_data.py \
        --input data/training/tool_calls.jsonl data/training/synthetic.jsonl \
        --format llama3 \
        --out data/training/training_data_llama3.jsonl

    python scripts/export_training_data.py \
        --input data/training/synthetic.jsonl \
        --format qwen25 \
        --out data/training/training_data_qwen25.jsonl
"""

import argparse
import json
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# DUSTY tool definitions (mirrors the Go ToolInfo schemas)
# ---------------------------------------------------------------------------

DUSTY_TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "mesh_send",
            "description": "Send a text message to another DUSTY node on the LoRa mesh network.",
            "parameters": {
                "type": "object",
                "properties": {
                    "target": {
                        "type": "string",
                        "description": "Node ID or callsign of the recipient (e.g. DUSTY-B)",
                    },
                    "message": {
                        "type": "string",
                        "description": "Text content of the message (max 180 chars)",
                    },
                },
                "required": ["target", "message"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "mesh_inbox",
            "description": "Check the inbox for messages received over the LoRa mesh network.",
            "parameters": {
                "type": "object",
                "properties": {
                    "action": {
                        "type": "string",
                        "description": 'Inbox action: "unread", "history peer=<name>", "peers"',
                    },
                },
                "required": [],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "current_time",
            "description": "Returns the current date and time.",
            "parameters": {
                "type": "object",
                "properties": {},
                "required": [],
            },
        },
    },
]

SYSTEM_PROMPT = (
    "You are DUSTY, a desert philosopher AI agent running on a Raspberry Pi 5 "
    "at Burning Man. You are connected to a LoRa mesh radio network and can "
    "exchange short messages with other DUSTY agents nearby. Messages must be "
    "180 characters or less. Node names are case-insensitive. There is no "
    "broadcast tool. Do NOT call any tool for questions about philosophy, "
    "feelings, the burn, general knowledge, or anything that does not require "
    "mesh communication or time lookup."
)


def load_examples(paths: list[str]) -> list[dict]:
    """Load and merge JSONL from multiple files."""
    examples = []
    for p in paths:
        path = Path(p)
        if not path.exists():
            print(f"Warning: {path} does not exist, skipping", file=sys.stderr)
            continue
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    examples.append(json.loads(line))
                except json.JSONDecodeError as e:
                    print(f"Warning: skipping malformed line in {path}: {e}", file=sys.stderr)
    return examples


def infer_tool_call(ex: dict) -> dict | None:
    """Infer the function call from a training example.

    Uses _tool_args if present (synthetic data), otherwise constructs from
    the tool name and a generic args dict.
    """
    tools_called = ex.get("tools_called", [])
    if not tools_called:
        return None

    tool_name = tools_called[0]
    args = ex.get("_tool_args", {})

    # For real (non-synthetic) data without _tool_args, we can't know the
    # exact arguments the model used. Skip these or use defaults.
    if not args and tool_name == "mesh_send":
        return None  # Can't infer args for real sends
    if not args and tool_name == "mesh_inbox":
        args = {"action": "unread"}
    if not args and tool_name == "current_time":
        args = {}

    return {
        "name": tool_name,
        "arguments": json.dumps(args),
    }


def to_llama3(ex: dict) -> dict | None:
    """Convert to Llama 3 function-calling chat template format."""
    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": ex["msg"]},
    ]

    tool_call = infer_tool_call(ex)
    if tool_call:
        messages.append({
            "role": "assistant",
            "content": None,
            "tool_calls": [
                {
                    "id": "call_1",
                    "type": "function",
                    "function": tool_call,
                }
            ],
        })
    else:
        if ex.get("tools_called"):
            return None  # Has tools but we can't infer args — skip
        # No-tool response: assistant replies with text.
        messages.append({
            "role": "assistant",
            "content": "[conversational response]",
        })

    return {"messages": messages, "tools": DUSTY_TOOLS}


def to_qwen25(ex: dict) -> dict | None:
    """Convert to Qwen 2.5 function-calling chat template format.

    Qwen 2.5 uses the same OpenAI-compatible messages format.
    """
    # Qwen 2.5 uses the same schema as Llama 3 for Ollama tool calling.
    return to_llama3(ex)


FORMATS = {
    "llama3": to_llama3,
    "qwen25": to_qwen25,
}


def main():
    parser = argparse.ArgumentParser(description="Export DUSTY training data for fine-tuning")
    parser.add_argument("--input", nargs="+", required=True,
                        help="Input JSONL file(s) from collector and/or synthetic generator")
    parser.add_argument("--format", choices=list(FORMATS.keys()), default="llama3",
                        help="Target model format (default: llama3)")
    parser.add_argument("--out", required=True,
                        help="Output JSONL path")
    parser.add_argument("--include-real", action="store_true",
                        help="Include real (non-synthetic) data that has tool_args (risky: args may be wrong)")
    args = parser.parse_args()

    examples = load_examples(args.input)
    if not examples:
        print("No training examples found", file=sys.stderr)
        sys.exit(1)

    converter = FORMATS[args.format]
    converted = []
    skipped = 0

    for ex in examples:
        # Skip real data without --include-real unless it's a no-tool case.
        if ex.get("model") != "synthetic" and not args.include_real:
            if ex.get("tools_called"):
                skipped += 1
                continue

        result = converter(ex)
        if result:
            converted.append(result)
        else:
            skipped += 1

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    with open(out_path, "w") as f:
        for item in converted:
            f.write(json.dumps(item) + "\n")

    print(f"Exported {len(converted)} training examples ({args.format} format)")
    if skipped:
        print(f"  Skipped {skipped} examples (missing tool args or filtered)")
    print(f"  → {out_path}")


if __name__ == "__main__":
    main()
