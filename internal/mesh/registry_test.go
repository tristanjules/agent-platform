package mesh_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/mesh"
)

func TestRegistryUpsertAndGet(t *testing.T) {
	r, cleanup := tempRegistry(t)
	defer cleanup()

	entry := mesh.PeerEntry{
		NodeID:    "!abc123",
		AgentName: "DUSTY-B",
		OwnerName: "Lourens",
		LastSeen:  time.Now(),
	}
	isNew := r.Upsert(entry)
	if !isNew {
		t.Error("first upsert should be new")
	}

	got, ok := r.GetPeer("!abc123")
	if !ok {
		t.Fatal("peer should be retrievable after upsert")
	}
	if got.OwnerName != "Lourens" {
		t.Errorf("want OwnerName=Lourens, got %q", got.OwnerName)
	}
	if got.Status != mesh.PeerActive {
		t.Errorf("want status Active, got %q", got.Status)
	}
}

func TestRegistryUpsertExistingIsNotNew(t *testing.T) {
	r, cleanup := tempRegistry(t)
	defer cleanup()

	entry := mesh.PeerEntry{NodeID: "!abc123", AgentName: "DUSTY-B", LastSeen: time.Now()}
	r.Upsert(entry)
	isNew := r.Upsert(entry)
	if isNew {
		t.Error("second upsert of same node should not be new")
	}
}

func TestRegistryListActivePeers(t *testing.T) {
	r, cleanup := tempRegistry(t)
	defer cleanup()

	r.Upsert(mesh.PeerEntry{NodeID: "!n1", AgentName: "A", LastSeen: time.Now()})
	r.Upsert(mesh.PeerEntry{NodeID: "!n2", AgentName: "B", LastSeen: time.Now()})
	// Manually mark n2 as lost.
	r.MarkStalePeers() // Won't mark yet since just added.

	active := r.ListActivePeers()
	if len(active) != 2 {
		t.Errorf("expected 2 active peers, got %d", len(active))
	}
}

func TestRegistryMarkStalePeers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers.json")
	r, err := mesh.NewPeerRegistry(path, 100*time.Millisecond) // Very short TTL.
	if err != nil {
		t.Fatal(err)
	}
	r.Upsert(mesh.PeerEntry{NodeID: "!n1", AgentName: "A", LastSeen: time.Now().Add(-200 * time.Millisecond)})

	lost := r.MarkStalePeers()
	if len(lost) != 1 || lost[0] != "!n1" {
		t.Errorf("expected !n1 to be marked lost, got %v", lost)
	}
	peer, _ := r.GetPeer("!n1")
	if peer.Status != mesh.PeerLost {
		t.Errorf("expected status Lost, got %q", peer.Status)
	}
}

func TestRegistryPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers.json")

	r1, err := mesh.NewPeerRegistry(path, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	r1.Upsert(mesh.PeerEntry{NodeID: "!abc", AgentName: "DUSTY-B", OwnerName: "Lourens", LastSeen: time.Now()})

	// Create new registry from same path.
	r2, err := mesh.NewPeerRegistry(path, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	peer, ok := r2.GetPeer("!abc")
	if !ok {
		t.Fatal("peer should persist across registry instances")
	}
	if peer.OwnerName != "Lourens" {
		t.Errorf("want OwnerName=Lourens, got %q", peer.OwnerName)
	}
}

func TestRegistryGetPeerByName(t *testing.T) {
	r, cleanup := tempRegistry(t)
	defer cleanup()

	r.Upsert(mesh.PeerEntry{NodeID: "!abc", AgentName: "DUSTY-B", LastSeen: time.Now()})
	peer, ok := r.GetPeerByName("DUSTY-B")
	if !ok {
		t.Fatal("should find peer by agent name")
	}
	if peer.NodeID != "!abc" {
		t.Errorf("wrong peer returned: %+v", peer)
	}
}

func tempRegistry(t *testing.T) (*mesh.PeerRegistry, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")
	r, err := mesh.NewPeerRegistry(path, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return r, func() { os.Remove(path) }
}
