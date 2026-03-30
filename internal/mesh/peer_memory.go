package mesh

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// InteractionDirection indicates whether a message was sent or received.
type InteractionDirection string

const (
	DirectionSent     InteractionDirection = "sent"
	DirectionReceived InteractionDirection = "received"
)

// PeerFact is a timestamped free-text fact about a peer.
type PeerFact struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// PeerInteraction is a log entry for a message sent to or received from a peer.
type PeerInteraction struct {
	Time      time.Time            `json:"time"`
	Direction InteractionDirection `json:"direction"`
	MsgType   string               `json:"type"`
	Summary   string               `json:"summary"`
}

// PeerMemoryEntry holds all memory for a single peer.
type PeerMemoryEntry struct {
	NodeID       string            `json:"node_id"`
	AgentName    string            `json:"agent_name"`
	OwnerName    string            `json:"owner_name"`
	Facts        []PeerFact        `json:"facts"`
	Interactions []PeerInteraction `json:"interactions"`
}

// PeerMemorySummary is a brief summary of a peer's memory state.
type PeerMemorySummary struct {
	NodeID        string
	AgentName     string
	OwnerName     string
	FactCount     int
	LastInteraction time.Time
}

// PeerMemory is a separate store for rich per-peer facts and interaction history.
// It is distinct from PeerRegistry (connectivity metadata only).
type PeerMemory struct {
	entries  map[string]*PeerMemoryEntry // keyed by NodeID
	path     string
	maxInteractions int
	mu       sync.RWMutex
}

// NewPeerMemory creates a PeerMemory store with JSON persistence at path.
// maxInteractions is the per-peer interaction history retention limit.
func NewPeerMemory(path string, maxInteractions int) (*PeerMemory, error) {
	if maxInteractions <= 0 {
		maxInteractions = 100
	}
	pm := &PeerMemory{
		entries:         make(map[string]*PeerMemoryEntry),
		path:            path,
		maxInteractions: maxInteractions,
	}
	if err := pm.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("peer memory: load: %w", err)
	}
	return pm, nil
}

// SeedFromDiscovery creates or updates a peer memory entry from discovery metadata,
// seeding initial facts (owner, agent name, persona).
func (pm *PeerMemory) SeedFromDiscovery(nodeID, agentName, ownerName, persona string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry, exists := pm.entries[nodeID]
	if !exists {
		entry = &PeerMemoryEntry{NodeID: nodeID, AgentName: agentName, OwnerName: ownerName}
		pm.entries[nodeID] = entry
	}
	entry.AgentName = agentName
	entry.OwnerName = ownerName

	// Seed facts from discovery metadata.
	now := time.Now()
	if ownerName != "" {
		entry.Facts = appendFactIfAbsent(entry.Facts, "Owner: "+ownerName, now)
	}
	if agentName != "" {
		entry.Facts = appendFactIfAbsent(entry.Facts, "Agent name: "+agentName, now)
	}
	if persona != "" {
		entry.Facts = appendFactIfAbsent(entry.Facts, "Persona: "+persona, now)
	}
	_ = pm.save()
}

// AddFact appends a free-text fact for a peer.
func (pm *PeerMemory) AddFact(nodeID, text string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry := pm.ensureEntry(nodeID)
	entry.Facts = append(entry.Facts, PeerFact{Text: text, CreatedAt: time.Now()})
	_ = pm.save()
}

// GetFacts returns all facts for a peer, newest first.
func (pm *PeerMemory) GetFacts(nodeID string) []PeerFact {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	entry, ok := pm.entries[nodeID]
	if !ok {
		return nil
	}
	out := make([]PeerFact, len(entry.Facts))
	copy(out, entry.Facts)
	// Reverse for newest-first order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// LogInteraction records a sent/received message in the peer's interaction log.
func (pm *PeerMemory) LogInteraction(nodeID string, dir InteractionDirection, msgType, summary string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry := pm.ensureEntry(nodeID)
	entry.Interactions = append(entry.Interactions, PeerInteraction{
		Time:      time.Now(),
		Direction: dir,
		MsgType:   msgType,
		Summary:   summary,
	})
	// Prune to retention limit.
	if len(entry.Interactions) > pm.maxInteractions {
		entry.Interactions = entry.Interactions[len(entry.Interactions)-pm.maxInteractions:]
	}
	_ = pm.save()
}

// GetInteractions returns the most recent interactions for a peer (newest first).
func (pm *PeerMemory) GetInteractions(nodeID string, limit int) []PeerInteraction {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	entry, ok := pm.entries[nodeID]
	if !ok {
		return nil
	}
	all := entry.Interactions
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	// Return copy in newest-first order.
	out := make([]PeerInteraction, len(all))
	copy(out, all)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// GetAllPeerSummaries returns a brief summary for each known peer.
func (pm *PeerMemory) GetAllPeerSummaries() []PeerMemorySummary {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	summaries := make([]PeerMemorySummary, 0, len(pm.entries))
	for _, e := range pm.entries {
		s := PeerMemorySummary{
			NodeID:    e.NodeID,
			AgentName: e.AgentName,
			OwnerName: e.OwnerName,
			FactCount: len(e.Facts),
		}
		if len(e.Interactions) > 0 {
			s.LastInteraction = e.Interactions[len(e.Interactions)-1].Time
		}
		summaries = append(summaries, s)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].AgentName < summaries[j].AgentName
	})
	return summaries
}

func (pm *PeerMemory) ensureEntry(nodeID string) *PeerMemoryEntry {
	entry, ok := pm.entries[nodeID]
	if !ok {
		entry = &PeerMemoryEntry{NodeID: nodeID}
		pm.entries[nodeID] = entry
	}
	return entry
}

func (pm *PeerMemory) load() error {
	data, err := os.ReadFile(pm.path)
	if err != nil {
		return err
	}
	var entries []*PeerMemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	pm.entries = make(map[string]*PeerMemoryEntry, len(entries))
	for _, e := range entries {
		pm.entries[e.NodeID] = e
	}
	return nil
}

func (pm *PeerMemory) save() error {
	if pm.path == "" {
		return nil
	}
	entries := make([]*PeerMemoryEntry, 0, len(pm.entries))
	for _, e := range pm.entries {
		entries = append(entries, e)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pm.path, data, 0644)
}

// appendFactIfAbsent adds a fact only if no existing fact has the same text.
func appendFactIfAbsent(facts []PeerFact, text string, t time.Time) []PeerFact {
	for _, f := range facts {
		if f.Text == text {
			return facts
		}
	}
	return append(facts, PeerFact{Text: text, CreatedAt: t})
}
