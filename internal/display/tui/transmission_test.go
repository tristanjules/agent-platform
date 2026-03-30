package tui

import (
	"strings"
	"testing"

	"github.com/tristanj/dusty/internal/mesh"
	"github.com/tristanj/dusty/internal/notify"
	"github.com/tristanj/dusty/internal/state"
)

// mockRegistry implements meshPeerLookup for tests.
type mockRegistry struct {
	peers map[string]mesh.PeerEntry
}

func (m *mockRegistry) GetPeer(nodeID string) (mesh.PeerEntry, bool) {
	p, ok := m.peers[nodeID]
	return p, ok
}

// mockMessageStore implements meshMessageStore for tests.
type mockMessageStore struct {
	read      []string
	dismissed []string
	unread    []mesh.StoredMessage
}

func (m *mockMessageStore) MarkRead(id string) error {
	m.read = append(m.read, id)
	return nil
}

func (m *mockMessageStore) MarkDismissed(id string) error {
	m.dismissed = append(m.dismissed, id)
	return nil
}

func (m *mockMessageStore) UnreadCount() int { return len(m.unread) }

func (m *mockMessageStore) ListUnread() []mesh.StoredMessage { return m.unread }

func TestTransmissionOverlayShow(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	if overlay.Visible() {
		t.Fatal("overlay should be hidden initially")
	}

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id1", Sender: "!n1", Value: "hello playa"}
	overlay.Show(msg, "DUSTY-B", "Lourens")
	if !overlay.Visible() {
		t.Fatal("overlay should be visible after Show")
	}
}

func TestTransmissionOverlayAccept(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id2", Sender: "!n1", Value: "test"}
	overlay.Show(msg, "DUSTY-B", "")

	accepted := overlay.Accept()
	if accepted.ID != "id2" {
		t.Errorf("want id2, got %s", accepted.ID)
	}
	if overlay.Visible() {
		t.Error("overlay should be hidden after Accept")
	}
}

func TestTransmissionOverlayDismiss(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id3", Sender: "!n1", Value: "test"}
	overlay.Show(msg, "DUSTY-B", "")
	overlay.Dismiss()

	if overlay.Visible() {
		t.Error("overlay should be hidden after Dismiss")
	}
}

func TestTransmissionOverlayView(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)
	overlay.SetWidth(80)

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id4", Sender: "!n1", Value: "test"}
	overlay.Show(msg, "DUSTY-B", "Lourens")

	view := overlay.View()
	if !strings.Contains(view, "INCOMING TRANSMISSION") {
		t.Error("view should contain INCOMING TRANSMISSION")
	}
	if !strings.Contains(view, "DUSTY-B") {
		t.Error("view should contain sender name")
	}
	if !strings.Contains(view, "Lourens") {
		t.Error("view should contain owner name")
	}
}

func TestHandleTransmissionEventNotification(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id5", Sender: "!n1", Value: "hi"}
	ev := state.NewEvent(state.EventNotificationTriggered, notify.NotificationPayload{Message: msg, IsIdle: false})

	showOverlay, sysMsg := HandleTransmissionEvent(ev, &overlay, nil)
	if !showOverlay {
		t.Error("expected showOverlay=true for active notification")
	}
	if sysMsg != "" {
		t.Errorf("expected no sysMsg, got %q", sysMsg)
	}
}

func TestHandleTransmissionEventIdleSuppressed(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	msg := mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id6", Sender: "!n1", Value: "hi"}
	ev := state.NewEvent(state.EventNotificationTriggered, notify.NotificationPayload{Message: msg, IsIdle: true})

	showOverlay, _ := HandleTransmissionEvent(ev, &overlay, nil)
	if showOverlay {
		t.Error("idle notification should not show overlay")
	}
}

func TestHandleTransmissionEventNodeDiscovered(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)
	registry := &mockRegistry{peers: map[string]mesh.PeerEntry{
		"!n1": {NodeID: "!n1", AgentName: "DUSTY-B", OwnerName: "Lourens"},
	}}

	ev := state.NewEvent(state.EventMeshNodeDiscovered, "!n1")
	showOverlay, sysMsg := HandleTransmissionEvent(ev, &overlay, registry)

	if showOverlay {
		t.Error("node discovered should not show overlay")
	}
	if !strings.Contains(sysMsg, "entered mesh range") {
		t.Errorf("expected 'entered mesh range' in sysMsg, got %q", sysMsg)
	}
	if !strings.Contains(sysMsg, "DUSTY-B") {
		t.Errorf("expected sender name in sysMsg, got %q", sysMsg)
	}
}

func TestHandleTransmissionEventNodeLost(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	ev := state.NewEvent(state.EventMeshNodeLost, "!n2")
	showOverlay, sysMsg := HandleTransmissionEvent(ev, &overlay, nil)

	if showOverlay {
		t.Error("node lost should not show overlay")
	}
	if !strings.Contains(sysMsg, "signal lost") {
		t.Errorf("expected 'signal lost' in sysMsg, got %q", sysMsg)
	}
}

func TestUnreadBannerView(t *testing.T) {
	theme := ThemeByName("phosphor-green")
	overlay := NewTransmissionOverlay(theme)

	banner := overlay.UnreadBannerView(3, []string{"DUSTY-B", "DUSTY-C"})
	if !strings.Contains(banner, "3") {
		t.Error("banner should contain count")
	}
	if !strings.Contains(banner, "DUSTY-B") {
		t.Error("banner should contain sender")
	}
}
