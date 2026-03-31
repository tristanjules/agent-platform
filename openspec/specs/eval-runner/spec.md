## ADDED Requirements

### Requirement: Eval runner accepts golden test cases from a YAML file
The `cmd/eval/` binary SHALL load test cases from a YAML file specified by `--cases`. Each case SHALL define: `prompt` (string), `expected_tool` (string, the expected tool name), `min_success_rate` (float, 0–1), and `runs` (int, number of repetitions).

#### Scenario: Valid YAML cases loaded
- **WHEN** `dusty-eval --cases testdata/eval/tool-calling-cases.yaml` is executed with a valid YAML file
- **THEN** all cases are parsed and the run count, expected tools, and success thresholds are applied

#### Scenario: Invalid or missing YAML file
- **WHEN** `dusty-eval --cases nonexistent.yaml` is executed
- **THEN** the binary exits with a non-zero code and prints a descriptive error

### Requirement: Eval runner accepts a config file to select the model under test
The `cmd/eval/` binary SHALL accept a `--config` flag pointing to a TOML config file. The model used for inference SHALL be taken from `cfg.Inference.Local.Model`. This allows the same golden cases to be run against different model configs.

#### Scenario: Different model configs produce different reports
- **WHEN** the eval runner is invoked twice with `--config configs/tool-test-gemma.toml` and `--config configs/tool-test-qwen.toml`
- **THEN** each run produces a separate JSON report identifying the model and its per-case success rates

### Requirement: Eval runner measures per-case success rate
For each test case, the eval runner SHALL run the agent `runs` times and record whether a `ToolCall` was emitted with `Function.Name == expected_tool`. The per-case success rate SHALL be `successful_runs / total_runs`.

#### Scenario: All runs succeed
- **WHEN** a case runs 10 times and all 10 produce the expected ToolCall
- **THEN** the case success rate is 1.0 and the case is marked PASS

#### Scenario: Success rate below threshold
- **WHEN** a case runs 10 times and only 6 produce the expected ToolCall (rate = 0.6) against a threshold of 0.75
- **THEN** the case is marked FAIL in the report

### Requirement: Eval runner writes a structured JSON report
On completion, the eval runner SHALL write a JSON report to the path specified by `--out` (defaulting to `eval-report.json`). The report SHALL include: model name, config path, timestamp, per-case results (prompt, expected tool, runs, successes, rate, pass/fail), and an overall pass rate.

#### Scenario: Report written after successful run
- **WHEN** the eval runner completes all cases
- **THEN** a valid JSON file exists at the `--out` path containing per-case results and metadata

#### Scenario: Report printed to stdout without --out
- **WHEN** `--out -` is passed
- **THEN** the JSON report is written to stdout

### Requirement: Eval runner prints a human-readable summary to stdout during execution
The eval runner SHALL print live progress as cases run, including case index, prompt (truncated), current success count, and a final summary table.

#### Scenario: Live progress output
- **WHEN** the eval runner is executing a 10-case suite
- **THEN** each completed case prints a one-line result: case index, PASS/FAIL, success rate, and prompt summary

### Requirement: Eval runner exits with non-zero code if any case fails its threshold
If any test case's measured success rate is below its `min_success_rate`, the eval runner SHALL exit with code 1.

#### Scenario: All cases pass
- **WHEN** every case meets or exceeds its success threshold
- **THEN** the binary exits with code 0

#### Scenario: One or more cases fail
- **WHEN** at least one case is below its threshold
- **THEN** the binary exits with code 1 (enabling use in CI scripts)
