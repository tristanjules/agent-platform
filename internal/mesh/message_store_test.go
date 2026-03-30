package mesh_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/mesh"
)

func newTestMessageStore(t *testing.T) *mesh.MessageStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "messages.json")
	ms, err := mesh.NewMessageStore(path, 200)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func testMsg(id, sender string) mesh.MeshMessage {
	return mesh.MeshMessage{
		Type:   mesh.TypeMessage,
		ID:     id,
		Sender: sender,
		Value:  "hello",
	}
}

func TestMessageStoreReceiveAndGet(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))

	sm, ok := ms.GetMessage("id1")
	if !ok {
		t.Fatal("message should be retrievable")
	}
	if sm.Status != mesh.MessageUnread {
		t.Errorf("want Unread, got %s", sm.Status)
	}
}

func TestMessageStoreMarkRead(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))

	if err := ms.MarkRead("id1"); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	sm, _ := ms.GetMessage("id1")
	if sm.Status != mesh.MessageRead {
		t.Errorf("want Read, got %s", sm.Status)
	}
	if sm.ReadAt == nil {
		t.Error("ReadAt should be set")
	}
}

func TestMessageStoreMarkDismissed(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))

	if err := ms.MarkDismissed("id1"); err != nil {
		t.Fatalf("MarkDismissed: %v", err)
	}
	sm, _ := ms.GetMessage("id1")
	if sm.Status != mesh.MessageDismissed {
		t.Errorf("want Dismissed, got %s", sm.Status)
	}
}

func TestMessageStoreDismissedCanBeReplayed(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.MarkDismissed("id1")
	if err := ms.MarkRead("id1"); err != nil {
		t.Fatalf("dismissed → read should be valid: %v", err)
	}
	sm, _ := ms.GetMessage("id1")
	if sm.Status != mesh.MessageRead {
		t.Errorf("want Read after replay, got %s", sm.Status)
	}
}

func TestMessageStoreInvalidTransitionReadToDismissed(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.MarkRead("id1")
	if err := ms.MarkDismissed("id1"); err == nil {
		t.Fatal("should not be able to dismiss an already-read message")
	}
}

func TestMessageStoreListUnread(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.StoreReceived(testMsg("id2", "!n1"))
	ms.StoreReceived(testMsg("id3", "!n1"))
	ms.MarkRead("id1")

	unread := ms.ListUnread()
	if len(unread) != 2 {
		t.Errorf("want 2 unread, got %d", len(unread))
	}
}

func TestMessageStoreUnreadCount(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.StoreReceived(testMsg("id2", "!n1"))
	ms.MarkDismissed("id1")

	if count := ms.UnreadCount(); count != 1 {
		t.Errorf("want 1 unread, got %d", count)
	}
}

func TestMessageStoreListFromPeer(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.StoreReceived(testMsg("id2", "!n2"))
	ms.StoreReceived(testMsg("id3", "!n1"))

	msgs := ms.ListFromPeer("!n1", 10)
	if len(msgs) != 2 {
		t.Errorf("want 2 messages from !n1, got %d", len(msgs))
	}
}

func TestMessageStoreStoreSent(t *testing.T) {
	ms := newTestMessageStore(t)
	ms.StoreSent("!n1", "sentid1", mesh.TypeMessage, "hello from me")

	sm, ok := ms.GetMessage("sentid1")
	if !ok {
		t.Fatal("sent message should be stored")
	}
	if sm.Status != mesh.MessageSent {
		t.Errorf("want Sent, got %s", sm.Status)
	}
	if sm.SentAt == nil {
		t.Error("SentAt should be set")
	}
}

func TestMessageStoreRetentionPrunesReadFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "msg.json")
	ms, _ := mesh.NewMessageStore(path, 3) // Small limit.

	ms.StoreReceived(testMsg("id1", "!n1"))
	time.Sleep(time.Millisecond)
	ms.StoreReceived(testMsg("id2", "!n1"))
	time.Sleep(time.Millisecond)
	ms.MarkRead("id1")
	ms.MarkRead("id2")

	// Adding id3 and id4 should trigger pruning of oldest read messages.
	ms.StoreReceived(testMsg("id3", "!n1"))
	ms.StoreReceived(testMsg("id4", "!n1"))

	all := ms.ListAll(100)
	if len(all) > 3 {
		t.Errorf("store should not exceed max size 3, got %d", len(all))
	}
}

func TestMessageStoreUnreadNeverPruned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "msg.json")
	ms, _ := mesh.NewMessageStore(path, 3)

	ms.StoreReceived(testMsg("id1", "!n1"))
	ms.StoreReceived(testMsg("id2", "!n1"))
	ms.StoreReceived(testMsg("id3", "!n1"))
	ms.StoreReceived(testMsg("id4", "!n1")) // All unread, can't prune any.

	unread := ms.ListUnread()
	if len(unread) < 3 {
		t.Errorf("unread messages should never be pruned; got only %d", len(unread))
	}
}

func TestMessageStorePersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "msg.json")

	ms1, _ := mesh.NewMessageStore(path, 200)
	ms1.StoreReceived(testMsg("id1", "!n1"))
	ms1.MarkDismissed("id1")

	ms2, _ := mesh.NewMessageStore(path, 200)
	sm, ok := ms2.GetMessage("id1")
	if !ok {
		t.Fatal("message should persist")
	}
	if sm.Status != mesh.MessageDismissed {
		t.Errorf("persisted status should be Dismissed, got %s", sm.Status)
	}
}
