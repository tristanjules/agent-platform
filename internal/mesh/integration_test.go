package mesh

import (
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/state"
)

// ─── 14.1 Full send/receive cycle ───────────────────────────────────────────

// TestIntegration_ReceiveCycle verifies that an incoming framed packet is
// decoded and delivered to the incoming channel.
func TestIntegration_ReceiveCycle(t *testing.T) {
	mock := &mockPort{}
	tr := newTestTransport(mock)

	// Write a framed message into the mock read buffer BEFORE starting the loop.
	incoming := MeshMessage{
		Type:   TypeMessage,
		ID:     "abc12345",
		Sender: "!node01",
		Value:  "hello from the playa",
	}
	payload, _ := Encode(incoming)
	mock.readBuf.Write(framePacket(payload))

	go tr.receiveLoop()
	go tr.sendLoop() // required so Close() doesn't hang waiting for doneCh
	defer tr.Close()

	var received MeshMessage
	select {
	case received = <-tr.Incoming():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for incoming message")
	}
	if received.ID != incoming.ID {
		t.Errorf("want ID %q, got %q", incoming.ID, received.ID)
	}
	if received.Value != incoming.Value {
		t.Errorf("want value %q, got %q", incoming.Value, received.Value)
	}
}

// TestIntegration_SendCycle verifies that a queued outbound message is encoded,
// framed, and written to the serial port.
func TestIntegration_SendCycle(t *testing.T) {
	mock := &mockPort{}
	tr := newTestTransport(mock)

	// Only start the send loop (no receive loop to trigger reconnect).
	go tr.sendLoop()
	defer tr.Close()

	outbound := MeshMessage{
		Type:   TypeMessage,
		ID:     "xyz98765",
		Sender: "!self",
		Target: "!node01",
		Value:  "acknowledged",
	}
	outPayload, _ := Encode(outbound)
	if err := tr.Send("!node01", outPayload); err != nil {
		t.Fatal(err)
	}

	// Wait for the send loop's rate-limit ticker (10ms) to fire.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		mock.mu.Lock()
		n := mock.writeBuf.Len()
		mock.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mock.mu.Lock()
	written := mock.writeBuf.Bytes()
	mock.mu.Unlock()

	if len(written) < 4 {
		t.Fatalf("expected framed data written, got %d bytes", len(written))
	}
	if written[0] != 0x94 || written[1] != 0xC3 {
		t.Errorf("unexpected frame header: %02x %02x", written[0], written[1])
	}
}

// ─── 14.2 Discovery handshake with peer memory auto-seeding ─────────────────

func TestIntegration_DiscoveryHandshakeAndPeerMemory(t *testing.T) {
	registry, err := NewPeerRegistry(tmpFile(t, "peers*.json"), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	peerMemory, err := NewPeerMemory(tmpFile(t, "memory*.json"), 10)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate receiving a discovery message from a peer and processing it.
	discoverMsg := MeshMessage{
		Type:      TypeDiscover,
		ID:        NewMessageID(),
		Sender:    "!peer01",
		AgentName: "DUSTY-B",
		Owner:     "Lourens",
		Persona:   "philosopher",
	}

	// Registry upsert (as discovery handler would do).
	isNew := registry.Upsert(PeerEntry{
		NodeID:    discoverMsg.Sender,
		AgentName: discoverMsg.AgentName,
		OwnerName: discoverMsg.Owner,
		Persona:   discoverMsg.Persona,
		LastSeen:  time.Now(),
		Status:    PeerActive,
	})
	if !isNew {
		t.Error("expected new peer on first discovery")
	}

	// Auto-seed peer memory from discovery metadata.
	peerMemory.SeedFromDiscovery(discoverMsg.Sender, discoverMsg.AgentName, discoverMsg.Owner, discoverMsg.Persona)

	// Verify registry has the peer.
	peer, ok := registry.GetPeer("!peer01")
	if !ok {
		t.Fatal("peer not found in registry")
	}
	if peer.AgentName != "DUSTY-B" {
		t.Errorf("want AgentName=DUSTY-B, got %s", peer.AgentName)
	}
	if peer.OwnerName != "Lourens" {
		t.Errorf("want OwnerName=Lourens, got %s", peer.OwnerName)
	}

	// Peer memory should have seeded facts mentioning the owner and agent name.
	facts := peerMemory.GetFacts("!peer01")
	if len(facts) == 0 {
		t.Fatal("expected seeded facts in peer memory")
	}
	var foundOwner bool
	for _, f := range facts {
		if stringContains(f.Text, "Lourens") {
			foundOwner = true
		}
	}
	if !foundOwner {
		t.Error("expected owner 'Lourens' in seeded facts")
	}

	// Log an interaction and verify it appears.
	peerMemory.LogInteraction("!peer01", DirectionReceived, TypeDiscover, "discovery handshake")
	interactions := peerMemory.GetInteractions("!peer01", 10)
	if len(interactions) == 0 {
		t.Error("expected at least one interaction")
	}
	if interactions[0].Summary != "discovery handshake" {
		t.Errorf("want summary 'discovery handshake', got %q", interactions[0].Summary)
	}
}

// ─── 14.3 Message lifecycle: receive → unread → dismiss → replay → read ─────

func TestIntegration_MessageLifecycle(t *testing.T) {
	msgStore, err := NewMessageStore(tmpFile(t, "msgs*.json"), 50)
	if err != nil {
		t.Fatal(err)
	}

	incoming := MeshMessage{
		Type:   TypeMessage,
		ID:     "life001",
		Sender: "!peer01",
		Value:  "meet us at deep playa",
	}
	msgStore.StoreReceived(incoming)

	if msgStore.UnreadCount() != 1 {
		t.Fatalf("want 1 unread, got %d", msgStore.UnreadCount())
	}

	// Dismiss it — should move from unread → dismissed.
	if err := msgStore.MarkDismissed("life001"); err != nil {
		t.Fatal(err)
	}
	if msgStore.UnreadCount() != 0 {
		t.Error("dismissed message should not count as unread")
	}
	sm, ok := msgStore.GetMessage("life001")
	if !ok {
		t.Fatal("message not found after dismiss")
	}
	if sm.Status != MessageDismissed {
		t.Errorf("want dismissed, got %v", sm.Status)
	}

	// Replay — dismissed → read (allowed transition).
	if err := msgStore.MarkRead("life001"); err != nil {
		t.Fatalf("replay (dismissed→read): %v", err)
	}
	sm, _ = msgStore.GetMessage("life001")
	if sm.Status != MessageRead {
		t.Errorf("want read, got %v", sm.Status)
	}
	if sm.ReadAt == nil {
		t.Error("ReadAt should be set after MarkRead")
	}
}

// ─── 14.4 Idle period → messages queued → unread on resume ──────────────────

func TestIntegration_IdleQueueAndUnreadResume(t *testing.T) {
	bus := state.NewEventBus(16)
	defer bus.Close()

	msgStore, err := NewMessageStore(tmpFile(t, "idle*.json"), 50)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate 3 messages arriving while user is idle.
	for i := 0; i < 3; i++ {
		msgStore.StoreReceived(MeshMessage{
			Type:   TypeMessage,
			ID:     NewMessageID(),
			Sender: "!peer01",
			Value:  "idle message",
		})
	}

	// Unread count should reflect all messages.
	if msgStore.UnreadCount() != 3 {
		t.Fatalf("want 3 unread after idle, got %d", msgStore.UnreadCount())
	}

	// ListUnread returns them all.
	unread := msgStore.ListUnread()
	if len(unread) != 3 {
		t.Errorf("want 3 unread messages, got %d", len(unread))
	}

	// "Resume" — mark all read (as if user reviewed them).
	for _, msg := range unread {
		_ = msgStore.MarkRead(msg.ID)
	}
	if msgStore.UnreadCount() != 0 {
		t.Errorf("want 0 unread after marking all read, got %d", msgStore.UnreadCount())
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

// tmpFile returns a path in a temp dir that does NOT yet exist,
// so stores can create it fresh (empty file would fail JSON unmarshal).
func tmpFile(t *testing.T, name string) string {
	t.Helper()
	return t.TempDir() + "/" + name
}

func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
