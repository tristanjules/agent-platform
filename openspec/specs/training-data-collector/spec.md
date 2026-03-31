# training-data-collector Specification

## Purpose
Collects classification and tool-call outcome data from every Chat() invocation, writing JSONL training examples for future classifier fine-tuning. Operates asynchronously to avoid impacting inference latency.

## Requirements
### Requirement: TrainingCollector logs classification and outcome for every Chat call
The `TrainingCollector` SHALL record a training example for every `Chat()` invocation. Each example SHALL contain: timestamp, user message, ClassifyResult (ShouldUseTool, Confidence, SuggestedTools), tools actually called (names), whether tool calls succeeded, and the model name used. Examples SHALL be written as JSON-lines to a configurable file path.

#### Scenario: Tool-call interaction logged
- **WHEN** `Chat()` completes with a tool call to mesh_send that succeeds
- **THEN** a JSONL entry is appended with `tools_called: ["mesh_send"]`, `tool_success: true`, and the classification that was made

#### Scenario: Conversation interaction logged
- **WHEN** `Chat()` completes with no tool calls
- **THEN** a JSONL entry is appended with `tools_called: []`, `tool_success: true`, and the classification that was made

#### Scenario: Failed tool call logged
- **WHEN** `Chat()` completes with a tool call that returns an error
- **THEN** a JSONL entry is appended with `tool_success: false`

### Requirement: Training data collection is async and non-blocking
The `TrainingCollector` SHALL write training examples via a buffered channel. The `Record()` method SHALL never block `Chat()`. If the write buffer is full, the example SHALL be dropped silently (matching EventBus drop semantics).

#### Scenario: Slow disk does not block Chat
- **WHEN** the training data file write is slow and the buffer is full
- **THEN** `Record()` returns immediately and the example is dropped

#### Scenario: Normal operation writes all examples
- **WHEN** the buffer has capacity
- **THEN** `Record()` enqueues the example and it is written to disk asynchronously

### Requirement: TrainingCollector is optional and configurable
The `TrainingCollector` SHALL be created only when `inference.classifier.training_data_path` is set in config. When not configured, the agent SHALL skip training data collection with no performance impact. The collector SHALL expose a `Close()` method that flushes the buffer and closes the file.

#### Scenario: Training data path configured
- **WHEN** `inference.classifier.training_data_path = "data/training/tool_calls.jsonl"` is set
- **THEN** a TrainingCollector is created and attached to the Agent

#### Scenario: Training data path not configured
- **WHEN** `inference.classifier.training_data_path` is empty or absent
- **THEN** no TrainingCollector is created and Chat() has no logging overhead

#### Scenario: Clean shutdown flushes buffer
- **WHEN** `collector.Close()` is called
- **THEN** all buffered examples are written to disk before the file is closed

### Requirement: Training data format supports fine-tuning export
Each JSONL entry SHALL follow this schema:
```json
{
  "ts": "RFC3339 timestamp",
  "msg": "user message text",
  "classify": {"should_use_tool": bool, "confidence": float, "suggested_tools": [string]},
  "tools_called": [string],
  "tool_success": bool,
  "model": "model name string"
}
```
Field names SHALL be terse to minimize file size on Pi 5 storage. The format SHALL be stable across versions to ensure training data collected over time remains usable.

#### Scenario: Entry format is valid JSON
- **WHEN** a training example is written
- **THEN** the line is valid JSON matching the schema above

#### Scenario: Entry contains all fields
- **WHEN** a training example is written for a tool-call interaction
- **THEN** all six fields are present and populated
