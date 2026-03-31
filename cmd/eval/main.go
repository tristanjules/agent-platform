// cmd/eval is a standalone tool calling evaluation harness.
// It loads golden test cases from YAML, runs them against a real Ollama model,
// measures per-case tool call success rates, and writes a JSON report.
//
// Usage:
//
//	go run ./cmd/eval/ --config configs/tool-test-qwen.toml \
//	                   --cases testdata/eval/tool-calling-cases.yaml \
//	                   --runs 10 --out eval-report.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tristanj/dusty/internal/agent"
	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/state"
)

// --- YAML case schema ---

type EvalCases struct {
	Cases []EvalCase `yaml:"cases"`
}

type EvalCase struct {
	Name           string  `yaml:"name"`
	Prompt         string  `yaml:"prompt"`
	ExpectedTool   string  `yaml:"expected_tool"`
	MinSuccessRate float64 `yaml:"min_success_rate"`
	Runs           int     `yaml:"runs"`
	Notes          string  `yaml:"notes"`
}

// --- Report schema ---

type EvalReport struct {
	Model       string            `json:"model"`
	ConfigPath  string            `json:"config_path"`
	CasesPath   string            `json:"cases_path"`
	RunsPerCase int               `json:"runs_per_case_default"`
	Timestamp   time.Time         `json:"timestamp"`
	Cases       []CaseResult      `json:"cases"`
	Summary     ReportSummary     `json:"summary"`
}

type CaseResult struct {
	Name           string  `json:"name"`
	Prompt         string  `json:"prompt"`
	ExpectedTool   string  `json:"expected_tool"`
	MinSuccessRate float64 `json:"min_success_rate"`
	Runs           int     `json:"runs"`
	Successes      int     `json:"successes"`
	Rate           float64 `json:"rate"`
	Pass           bool    `json:"pass"`
}

type ReportSummary struct {
	TotalCases  int     `json:"total_cases"`
	PassedCases int     `json:"passed_cases"`
	FailedCases int     `json:"failed_cases"`
	OverallRate float64 `json:"overall_rate"`
}

func main() {
	cfgPath := flag.String("config", "configs/tool-test-gemma.toml", "TOML config file selecting the model")
	casesPath := flag.String("cases", "testdata/eval/tool-calling-cases.yaml", "YAML golden test cases file")
	runsFlag := flag.Int("runs", 5, "default number of runs per case (overridden by case-level 'runs' field)")
	outPath := flag.String("out", "eval-report.json", "output JSON report path (use '-' for stdout)")
	flag.Parse()

	// Load config.
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load golden cases.
	cases, err := loadCases(*casesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading cases: %v\n", err)
		os.Exit(1)
	}

	modelName := cfg.Inference.Local.Model
	fmt.Printf("=== DUSTY Tool Calling Eval ===\n")
	fmt.Printf("Model:  %s\n", modelName)
	fmt.Printf("Config: %s\n", *cfgPath)
	fmt.Printf("Cases:  %s (%d cases)\n", *casesPath, len(cases))
	fmt.Printf("Runs:   %d (default)\n\n", *runsFlag)

	// Build recording tools for all well-known tool names seen in the cases.
	toolSet := buildRecordingToolSet(cases)

	// Build eval agent (real Ollama, recording tools).
	evalAgent := buildEvalAgent(cfg, toolSet)

	// Run cases.
	report := EvalReport{
		Model:       modelName,
		ConfigPath:  *cfgPath,
		CasesPath:   *casesPath,
		RunsPerCase: *runsFlag,
		Timestamp:   time.Now(),
	}

	anyFail := false
	for i, c := range cases {
		runs := c.Runs
		if runs <= 0 {
			runs = *runsFlag
		}
		result := runCase(evalAgent, toolSet, c, runs)
		report.Cases = append(report.Cases, result)

		status := "PASS"
		if !result.Pass {
			status = "FAIL"
			anyFail = true
		}
		fmt.Printf("[%d/%d] %s (%.2f) %s\n", i+1, len(cases), status, result.Rate, truncate(c.Prompt, 60))
	}

	// Compute summary.
	passed := 0
	for _, r := range report.Cases {
		if r.Pass {
			passed++
		}
	}
	report.Summary = ReportSummary{
		TotalCases:  len(report.Cases),
		PassedCases: passed,
		FailedCases: len(report.Cases) - passed,
		OverallRate: float64(passed) / float64(len(report.Cases)),
	}

	fmt.Printf("\n--- Summary: %d/%d passed (%.0f%%) ---\n",
		passed, len(report.Cases), report.Summary.OverallRate*100)

	// Write report.
	if err := writeReport(*outPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "error writing report: %v\n", err)
		os.Exit(1)
	}
	if *outPath != "-" {
		fmt.Printf("Report written to %s\n", *outPath)
	}

	if anyFail {
		os.Exit(1)
	}
}

// loadCases parses the YAML golden cases file.
func loadCases(path string) ([]EvalCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var ec EvalCases
	if err := yaml.Unmarshal(data, &ec); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(ec.Cases) == 0 {
		return nil, fmt.Errorf("no cases found in %s", path)
	}
	return ec.Cases, nil
}

// buildRecordingToolSet creates a RecordingTool for every distinct expected_tool in the cases,
// plus the standard tool set (mesh_send, mesh_inbox, current_time).
func buildRecordingToolSet(cases []EvalCase) map[string]*agent.RecordingTool {
	names := map[string]bool{
		"mesh_send":    true,
		"mesh_inbox":   true,
		"current_time": true,
	}
	for _, c := range cases {
		if c.ExpectedTool != "" {
			names[c.ExpectedTool] = true
		}
	}

	tools := make(map[string]*agent.RecordingTool, len(names))
	for name := range names {
		tools[name] = agent.NewRecordingTool(agent.EvalToolStub(name))
	}
	return tools
}

// buildEvalAgent creates an Agent with real Ollama routing and recording tools.
func buildEvalAgent(cfg *config.Config, toolSet map[string]*agent.RecordingTool) *agent.Agent {
	bus := state.NewEventBus(32)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	registry := make(map[string]agent.Tool, len(toolSet))
	for name, rt := range toolSet {
		registry[name] = rt
	}

	return agent.NewAgentWithRegistry(cfg, bus, log, registry)
}

// runCase executes one golden case `runs` times and returns the result.
func runCase(a *agent.Agent, toolSet map[string]*agent.RecordingTool, c EvalCase, runs int) CaseResult {
	successes := 0
	for i := 0; i < runs; i++ {
		// Reset all recording tools before each run.
		for _, rt := range toolSet {
			rt.Reset()
		}
		// Also reset agent memory so each run is independent.
		a.ClearMemory()

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		ch, err := a.Chat(ctx, c.Prompt)
		if err == nil {
			// Drain the channel.
			for range ch {
			}
		}
		cancel()

		// Check success.
		if c.ExpectedTool == "" {
			// No-tool case: success = no tool was called.
			anyCall := false
			for _, rt := range toolSet {
				if rt.WasCalled() {
					anyCall = true
					break
				}
			}
			if !anyCall {
				successes++
			}
		} else {
			if rt, ok := toolSet[c.ExpectedTool]; ok && rt.WasCalled() {
				successes++
			}
		}
	}

	rate := float64(successes) / float64(runs)
	pass := rate >= c.MinSuccessRate
	// No-tool cases use an inverted check with 0.0 threshold: pass if rate ≥ 0 (always),
	// but the real intent is rate < 0.3. Treat empty expected_tool + min_success_rate == 0.0
	// as "tool call rate must be below 0.3".
	if c.ExpectedTool == "" && c.MinSuccessRate == 0.0 {
		toolCallRate := 1.0 - rate // rate is "no-tool success rate"
		pass = toolCallRate < 0.30
	}

	return CaseResult{
		Name:           c.Name,
		Prompt:         c.Prompt,
		ExpectedTool:   c.ExpectedTool,
		MinSuccessRate: c.MinSuccessRate,
		Runs:           runs,
		Successes:      successes,
		Rate:           rate,
		Pass:           pass,
	}
}

// writeReport writes the JSON report to outPath (or stdout if "-").
func writeReport(outPath string, report EvalReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if outPath == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// buildCompareTable prints a summary comparing multiple eval reports (used by make eval-compare).
func buildCompareTable(reports []EvalReport) string {
	var sb strings.Builder
	sb.WriteString("Model                  | Total | Passed | Rate\n")
	sb.WriteString("-----------------------|-------|--------|------\n")
	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("%-22s | %5d | %6d | %.0f%%\n",
			truncate(r.Model, 22),
			r.Summary.TotalCases,
			r.Summary.PassedCases,
			r.Summary.OverallRate*100))
	}
	return sb.String()
}
