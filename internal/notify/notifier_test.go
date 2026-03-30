package notify_test

import (
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/haptic"
	"github.com/tristanj/dusty/internal/mesh"
	"github.com/tristanj/dusty/internal/notify"
	"github.com/tristanj/dusty/internal/state"
)

// mockSound records PlayFile calls.
type mockSound struct{ called int }

func (m *mockSound) PlayFile(_ string) error {
	m.called++
	return nil
}

func newTestNotifier(t *testing.T, idleTimeout time.Duration) (*notify.Notifier, *state.EventBus, *mockSound) {
	t.Helper()
	bus := state.NewEventBus(8)
	sound := &mockSound{}
	hap := haptic.NewStub()
	n := notify.NewNotifier(notify.NotifierConfig{
		SoundPath:   "test.wav",
		IdleTimeout: idleTimeout,
	}, bus, sound, hap, nil)
	return n, bus, sound
}

func TestNotifierPlaysSound(t *testing.T) {
	n, bus, sound := newTestNotifier(t, 5*time.Minute)
	defer bus.Close()

	ch := bus.Subscribe(state.EventNotificationTriggered)
	n.RecordActivity() // Mark as active.

	n.Notify(mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id1", Sender: "!n1", Value: "hello"})

	<-ch
	if sound.called != 1 {
		t.Errorf("want sound played once, got %d", sound.called)
	}
}

func TestNotifierSuppressesSoundWhenIdle(t *testing.T) {
	n, bus, sound := newTestNotifier(t, 1*time.Millisecond) // Very short idle timeout.
	defer bus.Close()

	ch := bus.Subscribe(state.EventNotificationTriggered)
	time.Sleep(5 * time.Millisecond) // Let idle timeout expire.

	n.Notify(mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id1", Sender: "!n1", Value: "hello"})

	evt := <-ch
	payload, ok := evt.Payload.(notify.NotificationPayload)
	if !ok {
		t.Fatal("expected NotificationPayload")
	}
	if !payload.IsIdle {
		t.Error("expected IsIdle=true")
	}
	if sound.called != 0 {
		t.Errorf("sound should not play when idle, got %d calls", sound.called)
	}
}

func TestNotifierPublishesEvent(t *testing.T) {
	n, bus, _ := newTestNotifier(t, 5*time.Minute)
	defer bus.Close()

	ch := bus.Subscribe(state.EventNotificationTriggered)
	n.RecordActivity()

	n.Notify(mesh.MeshMessage{Type: mesh.TypeMessage, ID: "id1", Sender: "!n1", Value: "hi"})

	evt := <-ch
	if evt.Type != state.EventNotificationTriggered {
		t.Errorf("expected EventNotificationTriggered, got %v", evt.Type)
	}
}

func TestNotifierIdleDetection(t *testing.T) {
	n, bus, _ := newTestNotifier(t, 50*time.Millisecond)
	defer bus.Close()

	n.RecordActivity()
	if n.IsUserIdle() {
		t.Error("should not be idle immediately after activity")
	}
	time.Sleep(100 * time.Millisecond)
	if !n.IsUserIdle() {
		t.Error("should be idle after timeout expires")
	}
}
