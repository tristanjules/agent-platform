package training

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/tristanj/dusty/internal/agent/classifier"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCollector_WritesJSONL(t *testing.T) {
	f, err := os.CreateTemp("", "training-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	c, err := New(path, testLogger())
	if err != nil {
		t.Fatal(err)
	}

	c.Record("Send DUSTY-B hello",
		classifier.ClassifyResult{ShouldUseTool: true, Confidence: 0.95, SuggestedTools: []string{"mesh_send"}},
		[]string{"mesh_send"}, true, "llama3.2:1b",
	)
	c.Record("What is the meaning of life?",
		classifier.ClassifyResult{ShouldUseTool: false, Confidence: 0.92},
		[]string{}, true, "llama3.2:1b",
	)
	c.Close()

	// Read back and verify.
	f2, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	var examples []Example
	scanner := bufio.NewScanner(f2)
	for scanner.Scan() {
		var ex Example
		if err := json.Unmarshal(scanner.Bytes(), &ex); err != nil {
			t.Fatalf("invalid JSON line: %s: %v", scanner.Text(), err)
		}
		examples = append(examples, ex)
	}

	if len(examples) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(examples))
	}

	// First example.
	ex := examples[0]
	if ex.Message != "Send DUSTY-B hello" {
		t.Errorf("msg = %q", ex.Message)
	}
	if !ex.Classify.ShouldUseTool {
		t.Error("expected should_use_tool=true")
	}
	if ex.Classify.Confidence != 0.95 {
		t.Errorf("confidence = %.2f", ex.Classify.Confidence)
	}
	if len(ex.ToolsCalled) != 1 || ex.ToolsCalled[0] != "mesh_send" {
		t.Errorf("tools_called = %v", ex.ToolsCalled)
	}
	if !ex.ToolSuccess {
		t.Error("expected tool_success=true")
	}
	if ex.Model != "llama3.2:1b" {
		t.Errorf("model = %q", ex.Model)
	}
	if ex.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}

	// Second example — no tools.
	ex2 := examples[1]
	if len(ex2.ToolsCalled) != 0 {
		t.Errorf("expected empty tools_called, got %v", ex2.ToolsCalled)
	}
}

func TestCollector_NonBlocking(t *testing.T) {
	// Use a tiny buffer and no drain goroutine — the channel fills immediately.
	// Record() must never block; if it did, the test would deadlock and timeout.
	c := &Collector{
		ch:   make(chan Example, 1),
		done: make(chan struct{}),
		log:  testLogger(),
	}
	for i := 0; i < 100; i++ {
		c.Record("msg", classifier.ClassifyResult{}, nil, true, "test")
	}
	// If we reach here, Record() never blocked. Success.
}

func TestCollector_NilIsNoop(t *testing.T) {
	var c *Collector
	// None of these should panic.
	c.Record("msg", classifier.ClassifyResult{}, nil, true, "test")
	c.Close()
}

func TestCollector_DisabledOnEmptyPath(t *testing.T) {
	c, err := New("", testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Error("expected nil Collector for empty path")
	}
}
