package tui

import (
	"fmt"
	"strings"

	"github.com/tristanj/dusty/internal/mesh"
	"github.com/tristanj/dusty/internal/notify"
	"github.com/tristanj/dusty/internal/state"
)

// TransmissionOverlay manages the retro sci-fi "INCOMING TRANSMISSION" interrupt.
// It appears on top of the normal TUI when a mesh message arrives.
type TransmissionOverlay struct {
	visible  bool
	message  mesh.MeshMessage
	sender   string // Human-readable sender (agent name or node ID).
	owner    string // Owner name.
	theme    Theme
	width    int
	unreadCount int // Count of messages waiting in queue (for banner mode).
}

// NewTransmissionOverlay creates a TransmissionOverlay.
func NewTransmissionOverlay(theme Theme) TransmissionOverlay {
	return TransmissionOverlay{theme: theme}
}

// Show displays the overlay for the given message.
func (t *TransmissionOverlay) Show(msg mesh.MeshMessage, senderName, ownerName string) {
	t.message = msg
	t.sender = senderName
	t.owner = ownerName
	t.visible = true
}

// Dismiss hides the overlay without reading the message.
func (t *TransmissionOverlay) Dismiss() {
	t.visible = false
}

// Accept hides the overlay (message is marked read by the caller).
func (t *TransmissionOverlay) Accept() mesh.MeshMessage {
	msg := t.message
	t.visible = false
	return msg
}

// Visible returns true if the overlay is currently shown.
func (t *TransmissionOverlay) Visible() bool { return t.visible }

// SetWidth sets the overlay width for rendering.
func (t *TransmissionOverlay) SetWidth(w int) { t.width = w }

// SetUnreadCount sets the unread queue count for the banner.
func (t *TransmissionOverlay) SetUnreadCount(n int) { t.unreadCount = n }

// View renders the overlay as a string.
func (t *TransmissionOverlay) View() string {
	if !t.visible {
		return ""
	}
	w := t.width
	if w < 40 {
		w = 80
	}
	line := strings.Repeat("═", w-2)
	sender := t.sender
	if t.owner != "" {
		sender = fmt.Sprintf("%s [%s]", t.sender, t.owner)
	}

	var sb strings.Builder
	sb.WriteString(t.theme.Primary.Render("╔" + line + "╗"))
	sb.WriteByte('\n')

	title := center(">>> INCOMING TRANSMISSION <<<", w-2)
	sb.WriteString(t.theme.Primary.Render("║" + title + "║"))
	sb.WriteByte('\n')

	from := center("From: "+sender, w-2)
	sb.WriteString(t.theme.Primary.Render("║" + from + "║"))
	sb.WriteByte('\n')

	empty := strings.Repeat(" ", w-2)
	sb.WriteString(t.theme.Primary.Render("║" + empty + "║"))
	sb.WriteByte('\n')

	prompt := center("[Enter] Accept    [Esc] Dismiss", w-2)
	sb.WriteString(t.theme.Primary.Render("║" + prompt + "║"))
	sb.WriteByte('\n')

	sb.WriteString(t.theme.Primary.Render("╚" + line + "╝"))
	return sb.String()
}

// UnreadBannerView renders the "messages waiting" banner shown on activity resume.
func (t *TransmissionOverlay) UnreadBannerView(count int, senders []string) string {
	senderList := strings.Join(senders, ", ")
	banner := fmt.Sprintf("⚡ %d unread transmission(s) from %s  [R] Review  [C] Continue", count, senderList)
	return t.theme.Accent.Render(banner)
}

// center pads s to fill width w.
func center(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	pad := w - len(s)
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// HandleTransmissionEvent processes a notification event in the TUI.
// Returns system messages to add to the conversation pane, if any.
func HandleTransmissionEvent(ev state.Event, overlay *TransmissionOverlay, registry meshPeerLookup) (showOverlay bool, systemMsg string) {
	switch ev.Type {
	case state.EventNotificationTriggered:
		payload, ok := ev.Payload.(notify.NotificationPayload)
		if !ok {
			return
		}
		senderName, ownerName := resolvePeerInfo(registry, payload.Message.Sender)
		if payload.IsIdle {
			// Idle: don't show overlay, just return count info.
			return false, ""
		}
		overlay.Show(payload.Message, senderName, ownerName)
		return true, ""

	case state.EventMeshNodeDiscovered:
		nodeID, _ := ev.Payload.(string)
		senderName, ownerName := resolvePeerInfo(registry, nodeID)
		label := senderName
		if ownerName != "" {
			label = fmt.Sprintf("%s [%s]", senderName, ownerName)
		}
		return false, label + " has entered mesh range"

	case state.EventMeshNodeLost:
		nodeID, _ := ev.Payload.(string)
		senderName, ownerName := resolvePeerInfo(registry, nodeID)
		label := senderName
		if ownerName != "" {
			label = fmt.Sprintf("%s [%s]", senderName, ownerName)
		}
		return false, label + " — signal lost"
	}
	return
}

// meshPeerLookup is the subset of PeerRegistry needed by the overlay.
type meshPeerLookup interface {
	GetPeer(nodeID string) (mesh.PeerEntry, bool)
}

// meshMessageStore is the subset of MessageStore needed by the TUI for
// accept/dismiss flows and the unread-on-resume banner.
type meshMessageStore interface {
	MarkRead(id string) error
	MarkDismissed(id string) error
	UnreadCount() int
	ListUnread() []mesh.StoredMessage
}

func resolvePeerInfo(registry meshPeerLookup, nodeID string) (name, owner string) {
	if registry == nil {
		return nodeID, ""
	}
	peer, ok := registry.GetPeer(nodeID)
	if !ok {
		return nodeID, ""
	}
	name = peer.AgentName
	if name == "" {
		name = nodeID
	}
	return name, peer.OwnerName
}
