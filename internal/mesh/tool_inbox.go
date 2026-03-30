package mesh

import (
	"fmt"
	"strings"
	"time"
)

// InboxTool is the agent-callable mesh_inbox tool.
// Supports actions: unread, history, replay, peer_facts, peers.
type InboxTool struct {
	store    *MessageStore
	memory   *PeerMemory
	registry *PeerRegistry
}

// NewInboxTool creates a mesh_inbox tool.
func NewInboxTool(store *MessageStore, memory *PeerMemory, registry *PeerRegistry) *InboxTool {
	return &InboxTool{store: store, memory: memory, registry: registry}
}

func (t *InboxTool) Name() string { return "mesh_inbox" }

func (t *InboxTool) Description() string {
	return "Query the mesh message inbox and peer information. " +
		"Args: action (unread/history/replay/peer_facts/peers, default: unread), " +
		"peer (agent name for history/peer_facts), id (message ID for replay)."
}

// Execute dispatches to the appropriate action handler.
func (t *InboxTool) Execute(args map[string]any) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		action = "unread"
	}

	switch action {
	case "unread":
		return t.actionUnread()
	case "history":
		peer, _ := args["peer"].(string)
		return t.actionHistory(peer)
	case "replay":
		id, _ := args["id"].(string)
		return t.actionReplay(id)
	case "peer_facts":
		peer, _ := args["peer"].(string)
		return t.actionPeerFacts(peer)
	case "peers":
		return t.actionPeers()
	default:
		return fmt.Sprintf("Unknown action '%s'. Valid actions: unread, history, replay, peer_facts, peers.", action), nil
	}
}

func (t *InboxTool) actionUnread() (string, error) {
	messages := t.store.ListUnread()
	if len(messages) == 0 {
		return "No unread transmissions.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "⚡ %d unread transmission(s):\n\n", len(messages))
	for _, m := range messages {
		senderName := t.resolveSenderName(m.SenderID)
		fmt.Fprintf(&sb, "[%s] %s: %s\n", relativeTime(m.ReceivedAt), senderName, m.Content)
	}
	return sb.String(), nil
}

func (t *InboxTool) actionHistory(peerArg string) (string, error) {
	if peerArg == "" {
		return "Please specify a peer: mesh_inbox(action=\"history\", peer=\"DUSTY-B\")", nil
	}
	nodeID, err := t.resolvePeerName(peerArg)
	if err != nil {
		return err.Error(), nil
	}
	messages := t.store.ListFromPeer(nodeID, 20)
	if len(messages) == 0 {
		return fmt.Sprintf("No message history with %s.", peerArg), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Message history with %s (%d messages):\n\n", peerArg, len(messages))
	// Display chronologically (reverse the newest-first list).
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		dir := "→"
		label := peerArg
		if m.SenderID == nodeID {
			dir = "←"
		}
		fmt.Fprintf(&sb, "[%s] %s %s: %s\n", relativeTime(m.ReceivedAt), dir, label, m.Content)
	}
	return sb.String(), nil
}

func (t *InboxTool) actionReplay(id string) (string, error) {
	if id == "" {
		return "Please specify a message ID: mesh_inbox(action=\"replay\", id=\"<id>\")", nil
	}
	sm, ok := t.store.GetMessage(id)
	if !ok {
		return fmt.Sprintf("Message '%s' not found.", id), nil
	}
	_ = t.store.MarkRead(id)
	senderName := t.resolveSenderName(sm.SenderID)
	return fmt.Sprintf("Transmission from %s (%s):\n\n%s", senderName, relativeTime(sm.ReceivedAt), sm.Content), nil
}

func (t *InboxTool) actionPeerFacts(peerArg string) (string, error) {
	if peerArg == "" {
		return "Please specify a peer: mesh_inbox(action=\"peer_facts\", peer=\"DUSTY-B\")", nil
	}
	nodeID, err := t.resolvePeerName(peerArg)
	if err != nil {
		return err.Error(), nil
	}
	facts := t.memory.GetFacts(nodeID)
	if len(facts) == 0 {
		return fmt.Sprintf("No stored facts about %s.", peerArg), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Known facts about %s:\n\n", peerArg)
	for _, f := range facts {
		fmt.Fprintf(&sb, "• %s  (%s)\n", f.Text, relativeTime(f.CreatedAt))
	}
	return sb.String(), nil
}

func (t *InboxTool) actionPeers() (string, error) {
	peers := t.registry.ListPeers()
	if len(peers) == 0 {
		return "No peers known. Run a discovery scan to find nearby DUSTY agents.", nil
	}
	summaries := t.memory.GetAllPeerSummaries()
	factsByNode := make(map[string]int)
	for _, s := range summaries {
		factsByNode[s.NodeID] = s.FactCount
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Known mesh peers (%d):\n\n", len(peers))
	for _, p := range peers {
		status := "●"
		if p.Status == PeerLost {
			status = "○"
		}
		owner := p.OwnerName
		if owner == "" {
			owner = "unknown"
		}
		facts := factsByNode[p.NodeID]
		fmt.Fprintf(&sb, "%s %s [%s] — last seen %s, %d fact(s)\n",
			status, p.AgentName, owner, relativeTime(p.LastSeen), facts)
	}
	return sb.String(), nil
}

// resolvePeerName converts an agent name to a node ID, with helpful error on unknown.
func (t *InboxTool) resolvePeerName(name string) (string, error) {
	peer, ok := t.registry.GetPeerByName(name)
	if !ok {
		known := t.registry.KnownAgentNames()
		if len(known) == 0 {
			return "", fmt.Errorf("Unknown peer '%s'. No peers in mesh.", name)
		}
		return "", fmt.Errorf("Unknown peer '%s'. Known peers: %s", name, strings.Join(known, ", "))
	}
	return peer.NodeID, nil
}

// resolveSenderName returns the agent name for a node ID, falling back to the raw ID.
func (t *InboxTool) resolveSenderName(nodeID string) string {
	peer, ok := t.registry.GetPeer(nodeID)
	if ok && peer.AgentName != "" {
		return peer.AgentName
	}
	return nodeID
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hrs := int(d.Hours())
		if hrs == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hrs)
	default:
		return t.Format("2006-01-02 15:04")
	}
}
