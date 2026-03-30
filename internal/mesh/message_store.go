package mesh

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// MessageStatus is the lifecycle state of a stored message.
type MessageStatus string

const (
	MessageUnread    MessageStatus = "unread"
	MessageRead      MessageStatus = "read"
	MessageDismissed MessageStatus = "dismissed"
	MessageSent      MessageStatus = "sent"
)

// StoredMessage is a mesh message persisted in the inbox.
type StoredMessage struct {
	ID         string        `json:"id"`
	SenderID   string        `json:"sender_id"`
	TargetID   string        `json:"target_id,omitempty"` // For sent messages.
	MsgType    string        `json:"type"`
	Content    string        `json:"content"`
	ReceivedAt time.Time     `json:"received_at"`
	SentAt     *time.Time    `json:"sent_at,omitempty"`
	ReadAt     *time.Time    `json:"read_at,omitempty"`
	Status     MessageStatus `json:"status"`
}

// ToMeshMessage reconstructs a MeshMessage from the stored entry.
func (sm StoredMessage) ToMeshMessage() MeshMessage {
	return MeshMessage{
		Type:   sm.MsgType,
		ID:     sm.ID,
		Sender: sm.SenderID,
		Value:  sm.Content,
	}
}

// MessageStore is a persistent inbox for all mesh messages.
// Status lifecycle: unread → read, unread → dismissed, dismissed → read.
type MessageStore struct {
	messages []*StoredMessage
	index    map[string]*StoredMessage // keyed by ID
	path     string
	maxSize  int
	mu       sync.RWMutex
}

// NewMessageStore creates a MessageStore with JSON persistence at path.
// maxSize is the total message retention limit.
func NewMessageStore(path string, maxSize int) (*MessageStore, error) {
	if maxSize <= 0 {
		maxSize = 200
	}
	ms := &MessageStore{
		path:    path,
		maxSize: maxSize,
		index:   make(map[string]*StoredMessage),
	}
	if err := ms.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("message store: load: %w", err)
	}
	return ms, nil
}

// StoreReceived adds an incoming message with status unread.
func (ms *MessageStore) StoreReceived(msg MeshMessage) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	sm := &StoredMessage{
		ID:         msg.ID,
		SenderID:   msg.Sender,
		MsgType:    msg.Type,
		Content:    msg.Value,
		ReceivedAt: now,
		Status:     MessageUnread,
	}
	ms.messages = append(ms.messages, sm)
	ms.index[sm.ID] = sm
	ms.prune()
	_ = ms.save()
}

// StoreSent adds an outgoing message with status sent.
func (ms *MessageStore) StoreSent(nodeID, msgID, msgType, content string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	sm := &StoredMessage{
		ID:         msgID,
		TargetID:   nodeID,
		MsgType:    msgType,
		Content:    content,
		ReceivedAt: now,
		SentAt:     &now,
		Status:     MessageSent,
	}
	ms.messages = append(ms.messages, sm)
	ms.index[sm.ID] = sm
	ms.prune()
	_ = ms.save()
}

// MarkRead transitions a message from unread or dismissed to read.
func (ms *MessageStore) MarkRead(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	sm, ok := ms.index[id]
	if !ok {
		return fmt.Errorf("message %q not found", id)
	}
	if sm.Status != MessageUnread && sm.Status != MessageDismissed {
		return fmt.Errorf("cannot mark %q as read (current status: %s)", id, sm.Status)
	}
	now := time.Now()
	sm.Status = MessageRead
	sm.ReadAt = &now
	_ = ms.save()
	return nil
}

// MarkDismissed transitions a message from unread to dismissed.
func (ms *MessageStore) MarkDismissed(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	sm, ok := ms.index[id]
	if !ok {
		return fmt.Errorf("message %q not found", id)
	}
	if sm.Status != MessageUnread {
		return fmt.Errorf("cannot dismiss %q (current status: %s)", id, sm.Status)
	}
	sm.Status = MessageDismissed
	_ = ms.save()
	return nil
}

// GetMessage returns a copy of the message with the given ID.
func (ms *MessageStore) GetMessage(id string) (StoredMessage, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	sm, ok := ms.index[id]
	if !ok {
		return StoredMessage{}, false
	}
	return *sm, true
}

// ListUnread returns all unread messages, newest first.
func (ms *MessageStore) ListUnread() []StoredMessage {
	return ms.listByStatus(MessageUnread)
}

// ListAll returns up to limit messages, newest first.
func (ms *MessageStore) ListAll(limit int) []StoredMessage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	sorted := ms.sortedNewestFirst()
	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

// ListFromPeer returns up to limit messages from/to a specific peer, newest first.
func (ms *MessageStore) ListFromPeer(nodeID string, limit int) []StoredMessage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var out []StoredMessage
	for _, sm := range ms.messages {
		if sm.SenderID == nodeID || sm.TargetID == nodeID {
			out = append(out, *sm)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReceivedAt.After(out[j].ReceivedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// UnreadCount returns the number of unread messages.
func (ms *MessageStore) UnreadCount() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	count := 0
	for _, sm := range ms.messages {
		if sm.Status == MessageUnread {
			count++
		}
	}
	return count
}

// listByStatus returns messages with the given status, newest first.
func (ms *MessageStore) listByStatus(status MessageStatus) []StoredMessage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var out []StoredMessage
	for _, sm := range ms.messages {
		if sm.Status == status {
			out = append(out, *sm)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReceivedAt.After(out[j].ReceivedAt)
	})
	return out
}

func (ms *MessageStore) sortedNewestFirst() []StoredMessage {
	out := make([]StoredMessage, len(ms.messages))
	for i, sm := range ms.messages {
		out[i] = *sm
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReceivedAt.After(out[j].ReceivedAt)
	})
	return out
}

// prune enforces the retention limit.
// Priority for removal: oldest read, then oldest dismissed. Unread is never removed.
func (ms *MessageStore) prune() {
	if len(ms.messages) <= ms.maxSize {
		return
	}
	// Sort by status priority and age for removal candidates.
	// Build separate age-sorted lists.
	var reads, dismissed []*StoredMessage
	for _, sm := range ms.messages {
		switch sm.Status {
		case MessageRead:
			reads = append(reads, sm)
		case MessageDismissed:
			dismissed = append(dismissed, sm)
		}
	}
	sort.Slice(reads, func(i, j int) bool {
		return reads[i].ReceivedAt.Before(reads[j].ReceivedAt)
	})
	sort.Slice(dismissed, func(i, j int) bool {
		return dismissed[i].ReceivedAt.Before(dismissed[j].ReceivedAt)
	})

	toRemove := make(map[string]bool)
	candidates := append(reads, dismissed...)
	excess := len(ms.messages) - ms.maxSize
	for i := 0; i < excess && i < len(candidates); i++ {
		toRemove[candidates[i].ID] = true
	}

	if len(toRemove) == 0 {
		return
	}

	filtered := ms.messages[:0]
	for _, sm := range ms.messages {
		if !toRemove[sm.ID] {
			filtered = append(filtered, sm)
		} else {
			delete(ms.index, sm.ID)
		}
	}
	ms.messages = filtered
}

func (ms *MessageStore) load() error {
	data, err := os.ReadFile(ms.path)
	if err != nil {
		return err
	}
	var messages []*StoredMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	ms.messages = messages
	ms.index = make(map[string]*StoredMessage, len(messages))
	for _, m := range messages {
		ms.index[m.ID] = m
	}
	return nil
}

func (ms *MessageStore) save() error {
	if ms.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(ms.messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ms.path, data, 0644)
}
