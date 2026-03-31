// Package training provides a non-blocking training data collector for DUSTY.
// Every Chat() call emits a training example containing the user message,
// classifier output, tools actually called, and outcome. These examples are
// written as JSON-lines (one JSON object per line) to a configurable path.
//
// The collector is the most important long-term deliverable of the model-router
// change: Burning Man conversations are irreplaceable domain-specific data for
// fine-tuning a tool-selection model.
package training

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/tristanj/dusty/internal/agent/classifier"
)

const defaultBufferSize = 256

// Example is a single training record. Field names are terse to minimise
// file size on Pi 5 storage. The schema is stable across versions.
type Example struct {
	Timestamp  string              `json:"ts"`
	Message    string              `json:"msg"`
	Classify   classifySnapshot    `json:"classify"`
	ToolsCalled []string           `json:"tools_called"`
	ToolSuccess bool               `json:"tool_success"`
	Model      string              `json:"model"`
}

// classifySnapshot mirrors classifier.ClassifyResult for serialisation.
type classifySnapshot struct {
	ShouldUseTool  bool     `json:"should_use_tool"`
	Confidence     float64  `json:"confidence"`
	SuggestedTools []string `json:"suggested_tools"`
}

// Collector writes training examples asynchronously to a JSONL file.
// It is safe for concurrent use. Record() never blocks the caller.
type Collector struct {
	ch   chan Example
	done chan struct{}
	log  *slog.Logger
}

// New creates a Collector that writes to path. Call Close() when done.
// Returns nil if path is empty (disables collection with no overhead).
func New(path string, log *slog.Logger) (*Collector, error) {
	if path == "" {
		return nil, nil //nolint:nilnil // intentional: nil Collector = disabled
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	c := &Collector{
		ch:   make(chan Example, defaultBufferSize),
		done: make(chan struct{}),
		log:  log,
	}

	go c.run(f)
	return c, nil
}

// Record enqueues a training example. Non-blocking: drops silently when buffer full.
func (c *Collector) Record(
	message string,
	result classifier.ClassifyResult,
	toolsCalled []string,
	toolSuccess bool,
	model string,
) {
	if c == nil {
		return
	}
	ex := Example{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
		Classify: classifySnapshot{
			ShouldUseTool:  result.ShouldUseTool,
			Confidence:     result.Confidence,
			SuggestedTools: result.SuggestedTools,
		},
		ToolsCalled: toolsCalled,
		ToolSuccess: toolSuccess,
		Model:       model,
	}
	if ex.ToolsCalled == nil {
		ex.ToolsCalled = []string{}
	}
	select {
	case c.ch <- ex:
	default:
		// Buffer full — drop rather than block Chat().
	}
}

// Close flushes the write buffer and closes the file. Blocks until done.
func (c *Collector) Close() {
	if c == nil {
		return
	}
	close(c.ch)
	<-c.done
}

func (c *Collector) run(f *os.File) {
	defer func() {
		if err := f.Close(); err != nil {
			c.log.Warn("training collector: file close error", "err", err)
		}
		close(c.done)
	}()

	enc := json.NewEncoder(f)
	for ex := range c.ch {
		if err := enc.Encode(ex); err != nil {
			c.log.Warn("training collector: write error", "err", err)
		}
	}
}
