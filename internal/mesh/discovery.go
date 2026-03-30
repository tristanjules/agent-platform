package mesh

import (
	"log"
	"time"

	"github.com/tristanj/dusty/internal/state"
)

// DiscoveryConfig holds configuration for the discovery handshake and heartbeats.
type DiscoveryConfig struct {
	NodeID           string
	AgentName        string
	OwnerName        string
	Persona          string
	DiscoveryTimeout time.Duration // How long to listen for responses (default 30s).
	HeartbeatInterval time.Duration // How often to send heartbeats (default 5min).
}

// Discovery manages the startup handshake, periodic heartbeats, and
// heartbeat reception — updating the peer registry on each contact.
// Non-discovery messages (msg, cmd, etc.) are forwarded to the EventBus
// as EventMeshMessageReceived so the notifier and message store can handle them.
type Discovery struct {
	cfg       DiscoveryConfig
	registry  *PeerRegistry
	transport *Transport
	bus       *state.EventBus
	stopCh    chan struct{}
}

// NewDiscovery creates a Discovery controller. The bus is used to publish
// EventMeshMessageReceived for non-discovery message types.
func NewDiscovery(cfg DiscoveryConfig, registry *PeerRegistry, transport *Transport, bus *state.EventBus) *Discovery {
	if cfg.DiscoveryTimeout == 0 {
		cfg.DiscoveryTimeout = 30 * time.Second
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 5 * time.Minute
	}
	return &Discovery{
		cfg:       cfg,
		registry:  registry,
		transport: transport,
		bus:       bus,
		stopCh:    make(chan struct{}),
	}
}

// Start broadcasts a discovery message and begins the heartbeat loop.
// It also starts listening for incoming discovery and heartbeat messages.
func (d *Discovery) Start() {
	d.sendDiscovery()
	go d.heartbeatLoop()
	go d.receiveLoop()
	go d.stalePeerLoop()
}

// Stop shuts down all discovery loops.
func (d *Discovery) Stop() {
	close(d.stopCh)
}

func (d *Discovery) sendDiscovery() {
	msg := MeshMessage{
		Type:      TypeDiscover,
		ID:        NewMessageID(),
		Sender:    d.cfg.NodeID,
		AgentName: d.cfg.AgentName,
		Owner:     d.cfg.OwnerName,
		Persona:   d.cfg.Persona,
	}
	payload, err := Encode(msg)
	if err != nil {
		log.Printf("[mesh/discovery] encode discovery: %v", err)
		return
	}
	if err := d.transport.Send("", payload); err != nil {
		log.Printf("[mesh/discovery] send discovery: %v", err)
	}
}

func (d *Discovery) sendHeartbeat() {
	msg := MeshMessage{
		Type:   TypeHeartbeat,
		ID:     NewMessageID(),
		Sender: d.cfg.NodeID,
	}
	payload, err := Encode(msg)
	if err != nil {
		log.Printf("[mesh/discovery] encode heartbeat: %v", err)
		return
	}
	if err := d.transport.Send("", payload); err != nil {
		log.Printf("[mesh/discovery] send heartbeat: %v", err)
	}
}

func (d *Discovery) heartbeatLoop() {
	ticker := time.NewTicker(d.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.sendHeartbeat()
		}
	}
}

// receiveLoop processes incoming messages. Discovery/heartbeat/location
// messages are handled internally; all other types are published to the
// EventBus so the notifier, message store, and TUI can react.
func (d *Discovery) receiveLoop() {
	for {
		select {
		case <-d.stopCh:
			return
		case msg, ok := <-d.transport.Incoming():
			if !ok {
				return
			}
			switch msg.Type {
			case TypeDiscover:
				d.handleDiscovery(msg)
			case TypeHeartbeat:
				d.handleHeartbeat(msg)
			case TypeLocation:
				d.handleLocation(msg)
			default:
				// Forward non-discovery messages (msg, cmd, ack, syn, etc.)
				// to the rest of the system via the event bus.
				if d.bus != nil {
					d.bus.Publish(state.NewEvent(state.EventMeshMessageReceived, msg))
				}
			}
		}
	}
}

func (d *Discovery) handleDiscovery(msg MeshMessage) {
	if msg.Sender == d.cfg.NodeID {
		return // Ignore our own discovery reflections.
	}
	entry := PeerEntry{
		NodeID:    msg.Sender,
		AgentName: msg.AgentName,
		OwnerName: msg.Owner,
		Persona:   msg.Persona,
		LastSeen:  time.Now(),
	}
	isNew := d.registry.Upsert(entry)
	if isNew {
		log.Printf("[mesh/discovery] new peer: %s (%s / %s)", msg.Sender, msg.AgentName, msg.Owner)
	}
	// Send a discovery reply so they know we exist too.
	d.sendDiscovery()
}

func (d *Discovery) handleHeartbeat(msg MeshMessage) {
	if msg.Sender == d.cfg.NodeID {
		return
	}
	d.registry.UpdateLastSeen(msg.Sender)
}

func (d *Discovery) handleLocation(msg MeshMessage) {
	if msg.Sender == d.cfg.NodeID || msg.Lat == 0 && msg.Lon == 0 {
		return
	}
	lat, lon := DecodeLocation(msg.Lat, msg.Lon)
	d.registry.UpdateLocation(msg.Sender, lat, lon)
}

// stalePeerLoop periodically checks for and marks stale peers.
func (d *Discovery) stalePeerLoop() {
	ticker := time.NewTicker(d.cfg.HeartbeatInterval / 2)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			lost := d.registry.MarkStalePeers()
			for _, nodeID := range lost {
				log.Printf("[mesh/discovery] peer lost: %s", nodeID)
			}
		}
	}
}
