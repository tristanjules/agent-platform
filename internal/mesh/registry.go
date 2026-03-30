package mesh

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// PeerStatus represents whether a peer is considered reachable.
type PeerStatus string

const (
	PeerActive PeerStatus = "active"
	PeerLost   PeerStatus = "lost"
)

// PeerEntry stores lean connectivity metadata for a known mesh node.
// Rich facts and interaction history are in PeerMemory.
type PeerEntry struct {
	NodeID    string     `json:"node_id"`
	AgentName string     `json:"agent_name"`
	OwnerName string     `json:"owner_name"`
	Persona   string     `json:"persona,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastSeen  time.Time  `json:"last_seen"`
	Location  *PeerLocation `json:"location,omitempty"`
	Status    PeerStatus `json:"status"`
}

// PeerLocation holds the last-known GPS coordinates of a peer.
type PeerLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// PeerRegistry manages known mesh peers with JSON persistence.
// It stays lean — only connectivity metadata. See PeerMemory for facts.
type PeerRegistry struct {
	peers    map[string]*PeerEntry // keyed by NodeID
	path     string
	peerTTL  time.Duration
	mu       sync.RWMutex
	bus      EventPublisher
}

// EventPublisher is a minimal interface for publishing events.
type EventPublisher interface {
	Publish(event interface{})
}

// NewPeerRegistry creates a registry with JSON persistence at path.
// peerTTL is how long a peer can be unseen before being marked lost.
func NewPeerRegistry(path string, peerTTL time.Duration) (*PeerRegistry, error) {
	r := &PeerRegistry{
		peers:   make(map[string]*PeerEntry),
		path:    path,
		peerTTL: peerTTL,
	}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("peer registry: load: %w", err)
	}
	return r, nil
}

// Upsert adds a new peer or updates an existing one. Returns true if new.
func (r *PeerRegistry) Upsert(entry PeerEntry) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.peers[entry.NodeID]
	entry.Status = PeerActive
	r.peers[entry.NodeID] = &entry
	_ = r.save()
	return !exists
}

// UpdateLastSeen refreshes the last-seen timestamp for a peer.
func (r *PeerRegistry) UpdateLastSeen(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.peers[nodeID]; ok {
		p.LastSeen = time.Now()
		p.Status = PeerActive
		_ = r.save()
	}
}

// UpdateLocation stores the last-known location for a peer.
func (r *PeerRegistry) UpdateLocation(nodeID string, lat, lon float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.peers[nodeID]; ok {
		p.Location = &PeerLocation{Lat: lat, Lon: lon}
		_ = r.save()
	}
}

// GetPeer returns a copy of the peer entry for the given node ID.
func (r *PeerRegistry) GetPeer(nodeID string) (PeerEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.peers[nodeID]
	if !ok {
		return PeerEntry{}, false
	}
	return *p, true
}

// GetPeerByName returns the first peer whose AgentName matches (case-insensitive prefix not required).
func (r *PeerRegistry) GetPeerByName(agentName string) (PeerEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.peers {
		if p.AgentName == agentName {
			return *p, true
		}
	}
	return PeerEntry{}, false
}

// ListPeers returns copies of all peer entries.
func (r *PeerRegistry) ListPeers() []PeerEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]PeerEntry, 0, len(r.peers))
	for _, p := range r.peers {
		out = append(out, *p)
	}
	return out
}

// ListActivePeers returns only peers with status PeerActive.
func (r *PeerRegistry) ListActivePeers() []PeerEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []PeerEntry
	for _, p := range r.peers {
		if p.Status == PeerActive {
			out = append(out, *p)
		}
	}
	return out
}

// MarkStalePeers checks all peers against the TTL and marks overdue ones as lost.
// Returns the node IDs of any peers newly marked lost.
func (r *PeerRegistry) MarkStalePeers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lost []string
	now := time.Now()
	changed := false
	for _, p := range r.peers {
		if p.Status == PeerActive && now.Sub(p.LastSeen) > r.peerTTL {
			p.Status = PeerLost
			lost = append(lost, p.NodeID)
			changed = true
		}
	}
	if changed {
		_ = r.save()
	}
	return lost
}

// KnownAgentNames returns the agent names of all peers (for display/suggestion).
func (r *PeerRegistry) KnownAgentNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.peers))
	for _, p := range r.peers {
		if p.AgentName != "" {
			names = append(names, p.AgentName)
		}
	}
	return names
}

func (r *PeerRegistry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	var peers []*PeerEntry
	if err := json.Unmarshal(data, &peers); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	r.peers = make(map[string]*PeerEntry, len(peers))
	for _, p := range peers {
		r.peers[p.NodeID] = p
	}
	return nil
}

func (r *PeerRegistry) save() error {
	if r.path == "" {
		return nil
	}
	peers := make([]*PeerEntry, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p)
	}
	data, err := json.MarshalIndent(peers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}
