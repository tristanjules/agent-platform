package mesh_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/mesh"
)

func TestPeerMemoryAddAndGetFacts(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	pm.AddFact("!n1", "Lourens is camped near Binnekring")
	pm.AddFact("!n1", "Has a solar charger")

	facts := pm.GetFacts("!n1")
	if len(facts) != 2 {
		t.Fatalf("want 2 facts, got %d", len(facts))
	}
	// Newest first.
	if facts[0].Text != "Has a solar charger" {
		t.Errorf("newest fact should be first, got %q", facts[0].Text)
	}
}

func TestPeerMemoryFactsUnknownPeer(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	facts := pm.GetFacts("!unknown")
	if facts != nil {
		t.Error("unknown peer should return nil facts")
	}
}

func TestPeerMemoryLogInteraction(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	pm.LogInteraction("!n1", mesh.DirectionReceived, mesh.TypeMessage, "heading to Binnekring")
	pm.LogInteraction("!n1", mesh.DirectionSent, mesh.TypeMessage, "meet at grid 4-7")

	interactions := pm.GetInteractions("!n1", 10)
	if len(interactions) != 2 {
		t.Fatalf("want 2 interactions, got %d", len(interactions))
	}
	// Newest first.
	if interactions[0].Summary != "meet at grid 4-7" {
		t.Errorf("newest interaction should be first")
	}
}

func TestPeerMemoryInteractionLimitPrune(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mem.json")
	pm, err := mesh.NewPeerMemory(path, 3) // Small limit.
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		pm.LogInteraction("!n1", mesh.DirectionReceived, mesh.TypeMessage, "msg")
	}
	interactions := pm.GetInteractions("!n1", 100)
	if len(interactions) > 3 {
		t.Errorf("want at most 3 interactions, got %d", len(interactions))
	}
}

func TestPeerMemorySeedFromDiscovery(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	pm.SeedFromDiscovery("!n1", "DUSTY-B", "Lourens", "Philosopher")
	facts := pm.GetFacts("!n1")

	texts := make(map[string]bool)
	for _, f := range facts {
		texts[f.Text] = true
	}
	if !texts["Owner: Lourens"] {
		t.Error("expected 'Owner: Lourens' fact after seeding")
	}
	if !texts["Agent name: DUSTY-B"] {
		t.Error("expected 'Agent name: DUSTY-B' fact after seeding")
	}
	if !texts["Persona: Philosopher"] {
		t.Error("expected 'Persona: Philosopher' fact after seeding")
	}
}

func TestPeerMemorySeedIsIdempotent(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	pm.SeedFromDiscovery("!n1", "DUSTY-B", "Lourens", "")
	pm.SeedFromDiscovery("!n1", "DUSTY-B", "Lourens", "")

	facts := pm.GetFacts("!n1")
	count := 0
	for _, f := range facts {
		if f.Text == "Owner: Lourens" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("seed should be idempotent: expected 1 'Owner' fact, got %d", count)
	}
}

func TestPeerMemoryPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mem.json")

	pm1, err := mesh.NewPeerMemory(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	pm1.AddFact("!n1", "camped at Binnekring")
	pm1.LogInteraction("!n1", mesh.DirectionReceived, mesh.TypeMessage, "hello")

	pm2, err := mesh.NewPeerMemory(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	facts := pm2.GetFacts("!n1")
	if len(facts) != 1 {
		t.Errorf("want 1 persisted fact, got %d", len(facts))
	}
	interactions := pm2.GetInteractions("!n1", 10)
	if len(interactions) != 1 {
		t.Errorf("want 1 persisted interaction, got %d", len(interactions))
	}
}

func TestPeerMemoryGetAllSummaries(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	pm.SeedFromDiscovery("!n1", "DUSTY-B", "Lourens", "")
	pm.SeedFromDiscovery("!n2", "DUSTY-C", "Alex", "")
	pm.LogInteraction("!n1", mesh.DirectionReceived, mesh.TypeMessage, "hello")

	summaries := pm.GetAllPeerSummaries()
	if len(summaries) != 2 {
		t.Fatalf("want 2 summaries, got %d", len(summaries))
	}
	// Sorted by AgentName.
	if summaries[0].AgentName != "DUSTY-B" {
		t.Errorf("expected first summary to be DUSTY-B, got %q", summaries[0].AgentName)
	}
	if summaries[0].LastInteraction.IsZero() {
		t.Error("DUSTY-B should have a non-zero last interaction time")
	}
}

func TestPeerMemoryGetInteractionsLimit(t *testing.T) {
	pm, cleanup := tempPeerMemory(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		pm.LogInteraction("!n1", mesh.DirectionReceived, mesh.TypeMessage, "msg")
		time.Sleep(time.Microsecond)
	}
	interactions := pm.GetInteractions("!n1", 5)
	if len(interactions) != 5 {
		t.Errorf("want 5 interactions with limit, got %d", len(interactions))
	}
}

func tempPeerMemory(t *testing.T) (*mesh.PeerMemory, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mem.json")
	pm, err := mesh.NewPeerMemory(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	return pm, func() {}
}
