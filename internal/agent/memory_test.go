package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanj/dusty/internal/agent"
)

func TestMemoryAddAndRetrieve(t *testing.T) {
	m, err := agent.NewConversationMemory(10, "")
	if err != nil {
		t.Fatal(err)
	}

	m.Add("user", "hello")
	m.Add("assistant", "hi there")

	msgs := m.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi there" {
		t.Errorf("unexpected second message: %+v", msgs[1])
	}
}

func TestMemorySlidingWindow(t *testing.T) {
	m, err := agent.NewConversationMemory(3, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		m.Add("user", string(rune('a'+i)))
	}

	msgs := m.Messages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (max_history), got %d", len(msgs))
	}
	// Should have the last 3: c, d, e
	if msgs[0].Content != "c" || msgs[1].Content != "d" || msgs[2].Content != "e" {
		t.Errorf("unexpected sliding window contents: %v", msgs)
	}
}

func TestMemoryClear(t *testing.T) {
	m, err := agent.NewConversationMemory(10, "")
	if err != nil {
		t.Fatal(err)
	}

	m.Add("user", "hello")
	m.Clear()

	if m.Len() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", m.Len())
	}
}

func TestMemoryJSONPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")

	// Write
	m1, err := agent.NewConversationMemory(10, path)
	if err != nil {
		t.Fatal(err)
	}
	m1.Add("user", "first message")
	m1.Add("assistant", "first response")
	if err := m1.Save(); err != nil {
		t.Fatal(err)
	}

	// Verify file was written
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("memory file was not created")
	}

	// Read back
	m2, err := agent.NewConversationMemory(10, path)
	if err != nil {
		t.Fatal(err)
	}

	msgs := m2.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages after reload, got %d", len(msgs))
	}
	if msgs[0].Content != "first message" {
		t.Errorf("unexpected message[0]: %q", msgs[0].Content)
	}
	if msgs[1].Content != "first response" {
		t.Errorf("unexpected message[1]: %q", msgs[1].Content)
	}
}

func TestMemoryLoadFromNonexistentFile(t *testing.T) {
	// Should succeed — file doesn't exist yet is not an error.
	m, err := agent.NewConversationMemory(10, "/tmp/dusty-nonexistent-99999.json")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got %v", err)
	}
	if m.Len() != 0 {
		t.Errorf("expected empty memory, got %d messages", m.Len())
	}
}
