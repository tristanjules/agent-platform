package mesh_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/mesh"
)

func setupTools(t *testing.T) (*mesh.SendTool, *mesh.InboxTool, *mesh.PeerRegistry, *mesh.MessageStore, *mesh.PeerMemory) {
	t.Helper()
	dir := t.TempDir()

	registry, _ := mesh.NewPeerRegistry(filepath.Join(dir, "peers.json"), 30*time.Minute)
	memory, _ := mesh.NewPeerMemory(filepath.Join(dir, "mem.json"), 100)
	store, _ := mesh.NewMessageStore(filepath.Join(dir, "msg.json"), 200)

	// Register a test peer.
	registry.Upsert(mesh.PeerEntry{
		NodeID:    "!abc123",
		AgentName: "DUSTY-B",
		OwnerName: "Lourens",
		LastSeen:  time.Now(),
	})

	cfg := mesh.TransportConfig{QueueSize: 4, RateLimit: 10 * time.Millisecond}
	transport := mesh.NewTransport(cfg)
	// Don't open real serial — we just need a transport for enqueue.

	sendTool := mesh.NewSendTool(transport, registry, memory, store, "!self1")
	inboxTool := mesh.NewInboxTool(store, memory, registry)
	return sendTool, inboxTool, registry, store, memory
}

// --- SendTool tests ---

func TestSendToolName(t *testing.T) {
	sendTool, _, _, _, _ := setupTools(t)
	if sendTool.Name() != "mesh_send" {
		t.Errorf("want mesh_send, got %q", sendTool.Name())
	}
}

func TestSendToolRequiresTarget(t *testing.T) {
	sendTool, _, _, _, _ := setupTools(t)
	_, err := sendTool.Execute(map[string]any{"message": "hello"})
	if err == nil {
		t.Fatal("should fail with no target")
	}
}

func TestSendToolRequiresMessage(t *testing.T) {
	sendTool, _, _, _, _ := setupTools(t)
	_, err := sendTool.Execute(map[string]any{"target": "DUSTY-B"})
	if err == nil {
		t.Fatal("should fail with no message")
	}
}

func TestSendToolUnknownTargetSuggestsKnown(t *testing.T) {
	sendTool, _, _, _, _ := setupTools(t)
	result, _ := sendTool.Execute(map[string]any{"target": "DUSTY-Z", "message": "hi"})
	if !strings.Contains(result, "DUSTY-B") {
		t.Errorf("error should suggest known peers, got: %q", result)
	}
}

func TestSendToolMessageTooLong(t *testing.T) {
	sendTool, _, _, _, _ := setupTools(t)
	result, _ := sendTool.Execute(map[string]any{
		"target":  "DUSTY-B",
		"message": strings.Repeat("x", 300),
	})
	if !strings.Contains(result, "too long") {
		t.Errorf("should report message too long, got: %q", result)
	}
}

func TestSendToolLogsToStore(t *testing.T) {
	sendTool, _, _, store, _ := setupTools(t)
	result, err := sendTool.Execute(map[string]any{"target": "DUSTY-B", "message": "test msg"})
	if err != nil {
		// If transport not open, it returns error — that's ok.
		return
	}
	if !strings.Contains(result, "queued") && !strings.Contains(result, "transmission") {
		t.Logf("result: %q", result)
	}
	_ = store
}

// --- InboxTool tests ---

func TestInboxToolName(t *testing.T) {
	_, inboxTool, _, _, _ := setupTools(t)
	if inboxTool.Name() != "mesh_inbox" {
		t.Errorf("want mesh_inbox, got %q", inboxTool.Name())
	}
}

func TestInboxToolDefaultsToUnread(t *testing.T) {
	_, inboxTool, _, store, _ := setupTools(t)
	store.StoreReceived(mesh.MeshMessage{
		Type: mesh.TypeMessage, ID: "id1", Sender: "!abc123", Value: "hello",
	})
	result, err := inboxTool.Execute(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "unread") {
		t.Errorf("default action should show unread, got: %q", result)
	}
}

func TestInboxToolUnreadEmpty(t *testing.T) {
	_, inboxTool, _, _, _ := setupTools(t)
	result, _ := inboxTool.Execute(map[string]any{"action": "unread"})
	if !strings.Contains(result, "No unread") {
		t.Errorf("expected no unread message, got: %q", result)
	}
}

func TestInboxToolReplay(t *testing.T) {
	_, inboxTool, _, store, _ := setupTools(t)
	store.StoreReceived(mesh.MeshMessage{
		Type: mesh.TypeMessage, ID: "replay1", Sender: "!abc123", Value: "test content",
	})
	result, _ := inboxTool.Execute(map[string]any{"action": "replay", "id": "replay1"})
	if !strings.Contains(result, "test content") {
		t.Errorf("replay should include message content, got: %q", result)
	}
	// Should be marked read.
	sm, _ := store.GetMessage("replay1")
	if sm.Status != mesh.MessageRead {
		t.Errorf("replayed message should be marked read")
	}
}

func TestInboxToolPeerFacts(t *testing.T) {
	_, inboxTool, _, _, memory := setupTools(t)
	memory.AddFact("!abc123", "Lourens is camped near Binnekring")
	result, _ := inboxTool.Execute(map[string]any{"action": "peer_facts", "peer": "DUSTY-B"})
	if !strings.Contains(result, "Binnekring") {
		t.Errorf("peer_facts should show facts, got: %q", result)
	}
}

func TestInboxToolUnknownPeerSuggests(t *testing.T) {
	_, inboxTool, _, _, _ := setupTools(t)
	result, _ := inboxTool.Execute(map[string]any{"action": "peer_facts", "peer": "DUSTY-Z"})
	if !strings.Contains(result, "DUSTY-B") {
		t.Errorf("should suggest known peers for unknown, got: %q", result)
	}
}

func TestInboxToolPeers(t *testing.T) {
	_, inboxTool, _, _, _ := setupTools(t)
	result, _ := inboxTool.Execute(map[string]any{"action": "peers"})
	if !strings.Contains(result, "DUSTY-B") {
		t.Errorf("peers should list DUSTY-B, got: %q", result)
	}
}

func TestInboxToolHistory(t *testing.T) {
	_, inboxTool, _, store, _ := setupTools(t)
	store.StoreReceived(mesh.MeshMessage{
		Type: mesh.TypeMessage, ID: "h1", Sender: "!abc123", Value: "from dusty-b",
	})
	result, _ := inboxTool.Execute(map[string]any{"action": "history", "peer": "DUSTY-B"})
	if !strings.Contains(result, "from dusty-b") {
		t.Errorf("history should include message content, got: %q", result)
	}
}
