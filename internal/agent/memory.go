package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Message represents a single conversation turn.
type Message struct {
	Role      string    `json:"role"`      // "system", "user", "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ConversationMemory stores conversation history in memory with
// optional JSON persistence to disk.
type ConversationMemory struct {
	messages   []Message
	maxHistory int
	persistTo  string // File path for JSON persistence; empty = no persistence.
	mu         sync.RWMutex
}

// NewConversationMemory creates a memory store. If persistPath is non-empty,
// it will attempt to load existing history from that file.
func NewConversationMemory(maxHistory int, persistPath string) (*ConversationMemory, error) {
	m := &ConversationMemory{
		maxHistory: maxHistory,
		persistTo:  persistPath,
	}

	// Attempt to load existing history.
	if persistPath != "" {
		if err := m.load(); err != nil {
			// Not an error if file doesn't exist yet.
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading memory from %s: %w", persistPath, err)
			}
		}
	}

	return m, nil
}

// Add appends a message to the conversation history and trims to maxHistory.
func (m *ConversationMemory) Add(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Sliding window: keep only the last maxHistory messages.
	if m.maxHistory > 0 && len(m.messages) > m.maxHistory {
		m.messages = m.messages[len(m.messages)-m.maxHistory:]
	}
}

// Messages returns a copy of the current conversation history.
func (m *ConversationMemory) Messages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Message, len(m.messages))
	copy(out, m.messages)
	return out
}

// Clear removes all messages from memory.
func (m *ConversationMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

// Len returns the number of messages in memory.
func (m *ConversationMemory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// Save persists the conversation history to the configured JSON file.
func (m *ConversationMemory) Save() error {
	if m.persistTo == "" {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.messages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling memory: %w", err)
	}

	if err := os.WriteFile(m.persistTo, data, 0644); err != nil {
		return fmt.Errorf("writing memory to %s: %w", m.persistTo, err)
	}

	return nil
}

// load reads conversation history from the persistence file.
func (m *ConversationMemory) load() error {
	data, err := os.ReadFile(m.persistTo)
	if err != nil {
		return err
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("unmarshaling memory from %s: %w", m.persistTo, err)
	}

	// Apply maxHistory limit on load.
	if m.maxHistory > 0 && len(messages) > m.maxHistory {
		messages = messages[len(messages)-m.maxHistory:]
	}

	m.messages = messages
	return nil
}
