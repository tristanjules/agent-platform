#!/usr/bin/env python3
"""Generate synthetic training data for DUSTY tool-calling fine-tuning.

Reads the golden eval cases from YAML and generates varied phrasings for each,
producing positive (tool-call) and negative (no-tool) training examples.
Output is JSONL matching the collector schema so it can be fed directly to
export_training_data.py.

Usage:
    python scripts/generate_synthetic_data.py \
        --cases testdata/eval/tool-calling-cases.yaml \
        --out data/training/synthetic.jsonl \
        --variations 10
"""

import argparse
import json
import random
import sys
from datetime import datetime, timezone
from pathlib import Path

# ---------------------------------------------------------------------------
# Variation templates per tool
# ---------------------------------------------------------------------------

MESH_SEND_VARIATIONS = [
    "Send {node} the message {msg}",
    "Tell {node} {msg}",
    "Message {node}: {msg}",
    "Can you send {msg} to {node}?",
    "Reach out to {node} and say {msg}",
    "Ping {node} with: {msg}",
    "Let {node} know: {msg}",
    "Text {node}: {msg}",
    "Send a message to {node} saying {msg}",
    "Please tell {node} that {msg}",
    "msg {node} {msg}",
    "Contact {node} — {msg}",
]

MESH_INBOX_VARIATIONS = [
    "Do I have any unread messages?",
    "What's in my inbox?",
    "Any new transmissions?",
    "Check my messages",
    "Have I heard from anyone?",
    "Show me unread messages",
    "Any word from {node}?",
    "Show me my message history with {node}",
    "What did {node} say?",
    "Read me my messages",
    "Did anyone message me?",
    "Pull up the inbox",
]

CURRENT_TIME_VARIATIONS = [
    "What time is it?",
    "What's the current time?",
    "Can you tell me the time?",
    "Time check",
    "What time do you have?",
    "Current time please",
    "What's the time right now?",
    "How late is it?",
]

NO_TOOL_VARIATIONS = [
    "What is the meaning of the burn?",
    "Tell me about radical self-reliance",
    "How are you feeling today?",
    "What do you think about consciousness?",
    "I feel deeply connected to the playa right now",
    "Explain the ten principles of Burning Man",
    "What's the history of the temple?",
    "Why do we build things just to burn them?",
    "Tell me something beautiful",
    "What is art?",
    "I'm feeling overwhelmed by the dust",
    "What should I do with my life?",
    "Do you believe in free will?",
    "Tell me a story about the desert",
    "What's it like being an AI at Burning Man?",
    "How does gifting work here?",
    "I feel lost",
    "What is love?",
    "Describe the sunrise on the playa",
    "Who are you?",
]

NODE_NAMES = [
    "DUSTY-B", "DUSTY-C", "DUSTY-D", "DUSTY-E",
    "dusty-b", "dusty-c", "Dusty-B", "Dusty-C",  # case variations
]

MESSAGES = [
    "hello",
    "I'll be at the temple at sunset",
    "meet at the effigy burn, 9pm",
    "heading to deep playa",
    "dust storm incoming, seek shelter",
    "anyone have water?",
    "meet at center camp in 30",
    "I'm okay, don't worry",
    "beautiful sunrise right now, come to 10 o'clock",
    "leaving the burn tomorrow morning",
]


def generate_mesh_send_example(variation_idx: int) -> dict:
    """Generate a mesh_send training example."""
    node = random.choice(NODE_NAMES)
    msg = random.choice(MESSAGES)
    template = MESH_SEND_VARIATIONS[variation_idx % len(MESH_SEND_VARIATIONS)]
    prompt = template.format(node=node, msg=msg)
    return {
        "ts": datetime.now(timezone.utc).isoformat(),
        "msg": prompt,
        "classify": {
            "should_use_tool": True,
            "confidence": 1.0,
            "suggested_tools": ["mesh_send"],
        },
        "tools_called": ["mesh_send"],
        "tool_success": True,
        "model": "synthetic",
        "_tool_args": {"target": node.upper(), "message": msg},
    }


def generate_mesh_inbox_example(variation_idx: int) -> dict:
    """Generate a mesh_inbox training example."""
    node = random.choice(NODE_NAMES)
    template = MESH_INBOX_VARIATIONS[variation_idx % len(MESH_INBOX_VARIATIONS)]
    prompt = template.format(node=node)
    return {
        "ts": datetime.now(timezone.utc).isoformat(),
        "msg": prompt,
        "classify": {
            "should_use_tool": True,
            "confidence": 1.0,
            "suggested_tools": ["mesh_inbox"],
        },
        "tools_called": ["mesh_inbox"],
        "tool_success": True,
        "model": "synthetic",
        "_tool_args": {"action": "unread"},
    }


def generate_current_time_example(variation_idx: int) -> dict:
    """Generate a current_time training example."""
    template = CURRENT_TIME_VARIATIONS[variation_idx % len(CURRENT_TIME_VARIATIONS)]
    return {
        "ts": datetime.now(timezone.utc).isoformat(),
        "msg": template,
        "classify": {
            "should_use_tool": True,
            "confidence": 1.0,
            "suggested_tools": ["current_time"],
        },
        "tools_called": ["current_time"],
        "tool_success": True,
        "model": "synthetic",
        "_tool_args": {},
    }


def generate_no_tool_example(variation_idx: int) -> dict:
    """Generate a no-tool (conversation) training example."""
    prompt = NO_TOOL_VARIATIONS[variation_idx % len(NO_TOOL_VARIATIONS)]
    return {
        "ts": datetime.now(timezone.utc).isoformat(),
        "msg": prompt,
        "classify": {
            "should_use_tool": False,
            "confidence": 1.0,
            "suggested_tools": [],
        },
        "tools_called": [],
        "tool_success": True,
        "model": "synthetic",
    }


def main():
    parser = argparse.ArgumentParser(description="Generate synthetic DUSTY training data")
    parser.add_argument("--cases", default="testdata/eval/tool-calling-cases.yaml",
                        help="Golden eval cases YAML (used for reference, not directly)")
    parser.add_argument("--out", default="data/training/synthetic.jsonl",
                        help="Output JSONL path")
    parser.add_argument("--variations", type=int, default=10,
                        help="Number of variations per tool category")
    parser.add_argument("--seed", type=int, default=42,
                        help="Random seed for reproducibility")
    args = parser.parse_args()

    random.seed(args.seed)
    examples = []

    # Generate positive examples for each tool.
    for i in range(args.variations):
        examples.append(generate_mesh_send_example(i))
        examples.append(generate_mesh_inbox_example(i))
        examples.append(generate_current_time_example(i))

    # Generate negative examples (no-tool conversations).
    # Roughly equal count to positive examples total.
    for i in range(args.variations * 2):
        examples.append(generate_no_tool_example(i))

    # Shuffle for training.
    random.shuffle(examples)

    # Write output.
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    with open(out_path, "w") as f:
        for ex in examples:
            f.write(json.dumps(ex) + "\n")

    print(f"Generated {len(examples)} synthetic training examples")
    print(f"  - mesh_send:    {args.variations}")
    print(f"  - mesh_inbox:   {args.variations}")
    print(f"  - current_time: {args.variations}")
    print(f"  - no-tool:      {args.variations * 2}")
    print(f"  → {out_path}")


if __name__ == "__main__":
    main()
