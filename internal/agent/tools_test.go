package agent

import (
	"context"
	"fmt"
	"log/slog"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/tristanj/dusty/internal/agent/classifier"
	"github.com/tristanj/dusty/internal/state"
)

// ---------------------------------------------------------------------------
// ToolInfoAdapter tests (task 4.1)
// ---------------------------------------------------------------------------

func TestToolInfoAdapter_TimeTool(t *testing.T) {
	info := ToolInfoAdapter(TimeTool{})

	if info.Name != "current_time" {
		t.Errorf("expected Name=current_time, got %s", info.Name)
	}
	if info.Desc == "" {
		t.Error("expected non-empty Desc")
	}
	if info.ParamsOneOf != nil {
		t.Error("expected nil ParamsOneOf for TimeTool (no parameters)")
	}
}

func TestToolInfoAdapter_ToolWithParams(t *testing.T) {
	tool := &testTool{
		name: "stub",
		desc: "a stub tool",
		params: map[string]*schema.ParameterInfo{
			"foo": {Type: schema.String, Desc: "foo param", Required: true},
		},
	}
	info := ToolInfoAdapter(tool)

	if info.Name != "stub" {
		t.Errorf("expected Name=stub, got %s", info.Name)
	}
	if info.ParamsOneOf == nil {
		t.Fatal("expected non-nil ParamsOneOf")
	}
	js, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema: %v", err)
	}
	if js == nil {
		t.Fatal("expected non-nil JSON schema")
	}
}

// ---------------------------------------------------------------------------
// executeTool tests (task 4.2)
// ---------------------------------------------------------------------------

func TestExecuteTool_UnknownTool(t *testing.T) {
	a := testAgent(t, &testModel{}, &testTool{name: "known", result: "ok"})
	tc := schema.ToolCall{
		ID:       "tc1",
		Function: schema.FunctionCall{Name: "nonexistent", Arguments: "{}"},
	}
	result, _ := a.executeTool(tc)
	if result != "unknown tool: nonexistent" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestExecuteTool_KnownTool(t *testing.T) {
	tool := &testTool{name: "known", result: "hello"}
	a := testAgent(t, &testModel{}, tool)
	tc := schema.ToolCall{
		ID:       "tc1",
		Function: schema.FunctionCall{Name: "known", Arguments: "{}"},
	}
	result, err := a.executeTool(tc)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExecuteTool_ToolError(t *testing.T) {
	tool := &testTool{name: "broken", execErr: fmt.Errorf("boom")}
	a := testAgent(t, &testModel{}, tool)
	tc := schema.ToolCall{
		ID:       "tc1",
		Function: schema.FunctionCall{Name: "broken", Arguments: "{}"},
	}
	result, err := a.executeTool(tc)
	if result == "" || result == "hello" {
		t.Errorf("expected error string, got: %q", result)
	}
	if err == nil {
		t.Error("expected non-nil error")
	}
}

// ---------------------------------------------------------------------------
// Tool-calling loop tests (tasks 4.3, 4.4)
// ---------------------------------------------------------------------------

func TestToolCallingLoop_TerminatesAfterToolCall(t *testing.T) {
	// Model returns a ToolCall on the first Generate(), then text on the final Stream().
	tool := &testTool{name: "noop", result: "noop result"}
	m := &testModel{
		generateResponses: []*schema.Message{
			// Round 1: ToolCall
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "tc1", Function: schema.FunctionCall{Name: "noop", Arguments: "{}"}},
				},
			},
			// Round 2: no more ToolCalls → triggers final Stream()
			{Role: schema.Assistant, Content: "All done."},
		},
		streamResponse: &schema.Message{Role: schema.Assistant, Content: "Streamed response."},
	}

	a := testAgent(t, m, tool)
	result := chatBlocking(t, a, "do something")
	if result == "" {
		t.Error("expected non-empty final response")
	}
}

func TestToolCallingLoop_MaxRounds(t *testing.T) {
	// Model always returns ToolCalls — loop should stop at maxToolCallRounds.
	tool := &testTool{name: "loop", result: "again"}
	infiniteResponses := make([]*schema.Message, 20)
	for i := range infiniteResponses {
		infiniteResponses[i] = &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{ID: "tc", Function: schema.FunctionCall{Name: "loop", Arguments: "{}"}},
			},
		}
	}

	m := &testModel{
		generateResponses: infiniteResponses,
		streamResponse:    &schema.Message{Role: schema.Assistant, Content: "finally done"},
	}

	a := testAgent(t, m, tool)
	chatBlocking(t, a, "loop forever")

	// Generate() calls should be capped at maxToolCallRounds.
	if m.generateCallCount > maxToolCallRounds {
		t.Errorf("expected ≤ %d Generate() calls, got %d", maxToolCallRounds, m.generateCallCount)
	}
}

// ---------------------------------------------------------------------------
// Memory test (task 4.5)
// ---------------------------------------------------------------------------

func TestChat_MemoryOnlyFinalResponseStored(t *testing.T) {
	// After a tool-assisted chat, memory should contain exactly 2 entries:
	// user message + final assistant response.
	tool := &testTool{name: "noop", result: "noop result"}
	m := &testModel{
		generateResponses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "tc1", Function: schema.FunctionCall{Name: "noop", Arguments: "{}"}},
				},
			},
			{Role: schema.Assistant, Content: "Done."},
		},
		streamResponse: &schema.Message{Role: schema.Assistant, Content: "Final streamed."},
	}

	a := testAgent(t, m, tool)
	initialLen := a.MemoryLen()
	chatBlocking(t, a, "use the tool")

	if a.MemoryLen() != initialLen+2 {
		t.Errorf("expected MemoryLen to increase by 2, got %d → %d", initialLen, a.MemoryLen())
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testTool is a minimal Tool implementation for testing.
type testTool struct {
	name    string
	desc    string
	params  map[string]*schema.ParameterInfo
	result  string
	execErr error
}

func (s *testTool) Name() string                              { return s.name }
func (s *testTool) Description() string                       { return s.desc }
func (s *testTool) Params() map[string]*schema.ParameterInfo { return s.params }
func (s *testTool) Execute(_ map[string]any) (string, error) {
	if s.execErr != nil {
		return "", s.execErr
	}
	if s.result == "" {
		return "stub result", nil
	}
	return s.result, nil
}

// testModel is a stub model.BaseChatModel for testing.
type testModel struct {
	generateResponses []*schema.Message
	generateCallCount int
	streamResponse    *schema.Message
	streamCallCount   int
}

func (m *testModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.generateCallCount++
	idx := m.generateCallCount - 1
	if idx < len(m.generateResponses) {
		return m.generateResponses[idx], nil
	}
	return &schema.Message{Role: schema.Assistant, Content: "default"}, nil
}

func (m *testModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m.streamCallCount++
	resp := m.streamResponse
	if resp == nil {
		resp = &schema.Message{Role: schema.Assistant, Content: "streamed"}
	}
	return schema.StreamReaderFromArray([]*schema.Message{resp}), nil
}

// testRouter wraps a model.BaseChatModel as a ModelRouter.
type testRouter struct {
	m model.BaseChatModel
}

func (r *testRouter) Route(_ context.Context) (model.BaseChatModel, error) { return r.m, nil }
func (r *testRouter) RouteWithClassification(_ context.Context, _ classifier.ClassifyResult) (model.BaseChatModel, error) {
	return r.m, nil
}
func (r *testRouter) SetPreference(_ RoutingPreference)  {}
func (r *testRouter) ListAvailable() []ModelInfo         { return nil }
func (r *testRouter) SupportsToolCalling() bool          { return true }

// testAgent builds a minimal Agent with the given model and extra tools.
func testAgent(t *testing.T, m model.BaseChatModel, extra ...Tool) *Agent {
	t.Helper()
	bus := state.NewEventBus(16)
	sm := state.NewStateMachine(state.StateWarmup, bus)
	if err := sm.Transition(state.StateIdle); err != nil {
		t.Fatal(err)
	}

	registry := map[string]Tool{}
	for _, tool := range extra {
		registry[tool.Name()] = tool
	}

	return &Agent{
		router:       &testRouter{m: m},
		memory:       mustConversationMemory(t),
		persona:      GetPersona("default"),
		sm:           sm,
		bus:          bus,
		log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		toolRegistry: registry,
	}
}

func mustConversationMemory(t *testing.T) *ConversationMemory {
	t.Helper()
	mem, err := NewConversationMemory(100, "")
	if err != nil {
		t.Fatal(err)
	}
	return mem
}

// chatBlocking calls Chat and drains the token channel, returning the full text.
func chatBlocking(t *testing.T, a *Agent, msg string) string {
	t.Helper()
	ch, err := a.Chat(context.Background(), msg)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	var out string
	for tok := range ch {
		out += tok
	}
	return out
}
