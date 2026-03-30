package mesh_test

import (
	"strings"
	"testing"

	"github.com/tristanj/dusty/internal/mesh"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	m := mesh.MeshMessage{
		Type:   mesh.TypeCmd,
		ID:     "abc12345",
		Sender: "!node1",
		Action: "nav",
		Value:  "grid-4-7",
	}
	data, err := mesh.Encode(m)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := mesh.Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Type != m.Type || got.Action != m.Action || got.Value != m.Value {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestEncodeRejectsOversizedPayload(t *testing.T) {
	m := mesh.MeshMessage{
		Type:   mesh.TypeMessage,
		ID:     "abc12345",
		Sender: "!node1",
		Value:  strings.Repeat("x", 300), // Way too large.
	}
	_, err := mesh.Encode(m)
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
}

func TestDecodeRejectsUnknownType(t *testing.T) {
	_, err := mesh.Decode([]byte(`{"t":"xyz","id":"abc12345","s":"!node1"}`))
	if err == nil {
		t.Fatal("expected error for unknown message type")
	}
}

func TestDecodeRejectsMalformedJSON(t *testing.T) {
	_, err := mesh.Decode([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestNewMessageIDIsEightChars(t *testing.T) {
	id := mesh.NewMessageID()
	if len(id) != 8 {
		t.Errorf("expected 8-char ID, got %q (len %d)", id, len(id))
	}
}

func TestNewMessageIDIsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := mesh.NewMessageID()
		if seen[id] {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}

func TestEncodeDecodeLocation(t *testing.T) {
	lat, lon := -31.416668, 19.233334
	la, lo := mesh.EncodeLocation(lat, lon)
	if la != -31416668 {
		t.Errorf("encoded lat: want -31416668, got %d", la)
	}
	if lo != 19233334 {
		t.Errorf("encoded lon: want 19233334, got %d", lo)
	}
	gotLat, gotLon := mesh.DecodeLocation(la, lo)
	if gotLat != lat {
		t.Errorf("decoded lat: want %f, got %f", lat, gotLat)
	}
	if gotLon != lon {
		t.Errorf("decoded lon: want %f, got %f", lon, gotLon)
	}
}

func TestLocationMessageFitsPayload(t *testing.T) {
	la, lo := mesh.EncodeLocation(-31.416668, 19.233334)
	m := mesh.MeshMessage{
		Type:   mesh.TypeLocation,
		ID:     "abc12345",
		Sender: "!abcd1234",
		Lat:    la,
		Lon:    lo,
	}
	data, err := mesh.Encode(m)
	if err != nil {
		t.Fatalf("location message should fit in 200 bytes: %v", err)
	}
	if len(data) > 200 {
		t.Errorf("encoded location is %d bytes, exceeds 200", len(data))
	}
}

func TestDeduplicatorRejectsDuplicate(t *testing.T) {
	d := mesh.NewDeduplicator(256)
	if d.IsDuplicate("abc12345") {
		t.Fatal("first occurrence should not be a duplicate")
	}
	if !d.IsDuplicate("abc12345") {
		t.Fatal("second occurrence should be a duplicate")
	}
}

func TestDeduplicatorWindowEviction(t *testing.T) {
	d := mesh.NewDeduplicator(3) // Small window for testing.
	d.IsDuplicate("id1")
	d.IsDuplicate("id2")
	d.IsDuplicate("id3")
	// id1 should be evicted when id4 is added.
	d.IsDuplicate("id4")
	// id1 is no longer in window, so not a duplicate.
	if d.IsDuplicate("id1") {
		t.Fatal("id1 should have been evicted from window")
	}
}

func TestTruncateToFitWordBoundary(t *testing.T) {
	// Large text with spaces.
	text := strings.Repeat("hello world ", 20) // 240 chars
	result := mesh.TruncateToFit(text, 50)     // 150-byte budget for text.
	if len(result) > 150 {
		t.Errorf("truncated text is %d bytes, want ≤ 150", len(result))
	}
	// Should end at a word boundary (space removed).
	if strings.HasSuffix(result, " ") {
		t.Error("truncated text should not end with a space")
	}
}

func TestTruncateToFitShortTextUnchanged(t *testing.T) {
	text := "short"
	result := mesh.TruncateToFit(text, 10)
	if result != text {
		t.Errorf("short text should be unchanged, got %q", result)
	}
}

func TestAllMessageTypesAreValid(t *testing.T) {
	types := []string{
		mesh.TypeCmd, mesh.TypeAck, mesh.TypeHeartbeat,
		mesh.TypeLocation, mesh.TypeMessage, mesh.TypeSync, mesh.TypeDiscover,
	}
	for _, typ := range types {
		m := mesh.MeshMessage{Type: typ, ID: "abc12345", Sender: "!node1"}
		data, err := mesh.Encode(m)
		if err != nil {
			t.Errorf("Encode(%q): %v", typ, err)
			continue
		}
		if _, err := mesh.Decode(data); err != nil {
			t.Errorf("Decode(%q): %v", typ, err)
		}
	}
}
