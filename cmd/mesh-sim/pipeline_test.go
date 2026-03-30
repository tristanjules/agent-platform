package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/haptic"
	"github.com/tristanj/dusty/internal/mesh"
	"github.com/tristanj/dusty/internal/notify"
	"github.com/tristanj/dusty/internal/state"
)

// TestFullPipeline tests the complete chain:
//   send → hub → receive → Discovery → EventBus → Notifier → MessageStore
//
// Requires: go run ./cmd/mesh-sim  (in another terminal on port 9090)
func TestFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pipeline test in short mode")
	}

	hubAddr := "sim://localhost:9090"

	// --- Node A: sender ---
	trA := mesh.NewTransport(mesh.TransportConfig{
		PortPath: hubAddr, BaudRate: 115200,
		QueueSize: 8, RateLimit: 10 * time.Millisecond,
	})
	if err := trA.Open(); err != nil {
		t.Skipf("mesh-sim hub not running: %v", err)
	}
	defer trA.Close()

	// --- Node B: receiver with full subsystem stack ---
	trB := mesh.NewTransport(mesh.TransportConfig{
		PortPath: hubAddr, BaudRate: 115200,
		QueueSize: 8, RateLimit: 10 * time.Millisecond,
	})
	if err := trB.Open(); err != nil {
		t.Fatalf("node B connect: %v", err)
	}
	defer trB.Close()

	busB := state.NewEventBus(16)
	defer busB.Close()

	regB, err := mesh.NewPeerRegistry(t.TempDir()+"/peers.json", 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := mesh.NewMessageStore(t.TempDir()+"/messages.json", 50)
	if err != nil {
		t.Fatal(err)
	}

	// Discovery for B — routes non-discovery messages to EventBus.
	discB := mesh.NewDiscovery(mesh.DiscoveryConfig{
		NodeID:            "!bbbbbbbb",
		AgentName:         "DUSTY-B",
		HeartbeatInterval: 1 * time.Hour,
		DiscoveryTimeout:  1 * time.Second,
	}, regB, trB, busB)
	discB.Start()
	defer discB.Stop()

	// Notifier for B — stores messages and fires EventNotificationTriggered.
	notifB := notify.NewNotifier(notify.NotifierConfig{
		IdleTimeout: 5 * time.Minute,
	}, busB, nil, haptic.NewStub(), storeB)
	stopCh := make(chan struct{})
	defer close(stopCh)
	notifB.Start(stopCh)
	notifB.RecordActivity()

	notifCh := busB.Subscribe(state.EventNotificationTriggered)

	time.Sleep(100 * time.Millisecond) // let loops start

	// --- A sends a message ---
	msg := mesh.MeshMessage{
		Type:   mesh.TypeMessage,
		ID:     mesh.NewMessageID(),
		Sender: "!aaaaaaaa",
		Target: "!bbbbbbbb",
		Value:  "full pipeline test",
	}
	payload, _ := mesh.Encode(msg)
	t.Logf("A sends msg ID=%s: %q", msg.ID, msg.Value)
	if err := trA.Send("!bbbbbbbb", payload); err != nil {
		t.Fatal(err)
	}

	// --- Verify EventNotificationTriggered fires on B ---
	select {
	case evt := <-notifCh:
		p, ok := evt.Payload.(notify.NotificationPayload)
		if !ok {
			t.Fatalf("unexpected payload type: %T", evt.Payload)
		}
		t.Logf("✓ NotificationTriggered: ID=%s Value=%q IsIdle=%v", p.Message.ID, p.Message.Value, p.IsIdle)
		if p.Message.ID != msg.ID {
			t.Errorf("ID mismatch: want %q got %q", msg.ID, p.Message.ID)
		}
		if p.Message.Value != msg.Value {
			t.Errorf("value mismatch: want %q got %q", msg.Value, p.Message.Value)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("TIMEOUT: EventNotificationTriggered did not fire within 3s")
	}

	// --- Verify message was stored in B's inbox ---
	time.Sleep(50 * time.Millisecond)
	if storeB.UnreadCount() != 1 {
		t.Errorf("want 1 unread in store, got %d", storeB.UnreadCount())
	}
	stored, ok := storeB.GetMessage(msg.ID)
	if !ok {
		t.Fatal("message not found in store")
	}
	if stored.Content != msg.Value {
		t.Errorf("stored content mismatch: want %q got %q", msg.Value, stored.Content)
	}
	fmt.Printf("✓ Full pipeline: msg stored (status=%s, content=%q)\n", stored.Status, stored.Content)
}
