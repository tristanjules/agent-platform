package agent

import (
	"sync"

	"github.com/cloudwego/eino/schema"
)

// RecordingTool wraps a Tool and records whether Execute() was called.
// Used in integration tests and the eval runner to observe tool invocations
// without requiring real infrastructure (mesh radio, etc.).
type RecordingTool struct {
	inner   Tool
	mu      sync.Mutex
	called  bool
	callLog []map[string]any
}

// NewRecordingTool wraps an existing Tool with call recording.
func NewRecordingTool(inner Tool) *RecordingTool {
	return &RecordingTool{inner: inner}
}

func (r *RecordingTool) Name() string                              { return r.inner.Name() }
func (r *RecordingTool) Description() string                       { return r.inner.Description() }
func (r *RecordingTool) Params() map[string]*schema.ParameterInfo { return r.inner.Params() }

func (r *RecordingTool) Execute(args map[string]any) (string, error) {
	r.mu.Lock()
	r.called = true
	r.callLog = append(r.callLog, args)
	r.mu.Unlock()
	return r.inner.Execute(args)
}

// WasCalled reports whether Execute() was invoked at least once.
func (r *RecordingTool) WasCalled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.called
}

// CallCount returns the number of times Execute() was called.
func (r *RecordingTool) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.callLog)
}

// Reset clears the call log and called flag.
func (r *RecordingTool) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = false
	r.callLog = nil
}
