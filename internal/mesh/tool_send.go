package mesh

import (
	"fmt"
	"strings"
)

// SendTool is the agent-callable mesh_send tool.
// It resolves targets, validates payload size, enqueues messages,
// and logs them to the message store and peer memory.
type SendTool struct {
	transport *Transport
	registry  *PeerRegistry
	memory    *PeerMemory
	store     *MessageStore
	selfID    string
}

// NewSendTool creates a mesh_send tool.
func NewSendTool(transport *Transport, registry *PeerRegistry, memory *PeerMemory, store *MessageStore, selfID string) *SendTool {
	return &SendTool{
		transport: transport,
		registry:  registry,
		memory:    memory,
		store:     store,
		selfID:    selfID,
	}
}

func (t *SendTool) Name() string { return "mesh_send" }

func (t *SendTool) Description() string {
	return "Send a message to another DUSTY agent over the LoRa mesh radio. " +
		"Args: target (agent name or node ID), message (text, keep under 100 chars), " +
		"type (optional: msg/cmd, default: msg)."
}

// Execute sends a mesh message to the specified target.
func (t *SendTool) Execute(args map[string]any) (string, error) {
	target, _ := args["target"].(string)
	message, _ := args["message"].(string)
	msgType, _ := args["type"].(string)

	if target == "" {
		return "", fmt.Errorf("mesh_send: 'target' argument is required")
	}
	if message == "" {
		return "", fmt.Errorf("mesh_send: 'message' argument is required")
	}
	if msgType == "" {
		msgType = TypeMessage
	}

	// Resolve target to node ID.
	nodeID, err := t.resolveTarget(target)
	if err != nil {
		return err.Error(), nil // Return as text so agent can relay to user.
	}

	msgID := NewMessageID()
	msg := MeshMessage{
		Type:   msgType,
		ID:     msgID,
		Sender: t.selfID,
		Target: nodeID,
		Value:  message,
	}

	payload, err := Encode(msg)
	if err != nil {
		// Message too large.
		budget := MaxPayloadBytes - 60 // rough fixed-field overhead
		return fmt.Sprintf("Message too long for mesh transmission. Please summarise to under %d characters.", budget), nil
	}

	if err := t.transport.Send(nodeID, payload); err != nil {
		return "", fmt.Errorf("mesh_send: send failed: %w", err)
	}

	// Log to store and peer memory.
	if t.store != nil {
		t.store.StoreSent(nodeID, msgID, msgType, message)
	}
	if t.memory != nil {
		t.memory.LogInteraction(nodeID, DirectionSent, msgType, truncateSummary(message, 80))
	}

	return fmt.Sprintf("Message queued for transmission to %s.", target), nil
}

// resolveTarget converts an agent name or raw node ID to a node ID.
func (t *SendTool) resolveTarget(target string) (string, error) {
	// Raw node IDs start with '!'.
	if strings.HasPrefix(target, "!") {
		return target, nil
	}
	peer, ok := t.registry.GetPeerByName(target)
	if !ok {
		known := t.registry.KnownAgentNames()
		if len(known) == 0 {
			return "", fmt.Errorf("Unknown peer '%s'. No peers currently in mesh.", target)
		}
		return "", fmt.Errorf("Unknown peer '%s'. Known peers: %s", target, strings.Join(known, ", "))
	}
	return peer.NodeID, nil
}

// truncateSummary caps a string at maxLen for interaction log summaries.
func truncateSummary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}
