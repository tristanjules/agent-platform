// Package mesh provides agent-to-agent communication over LoRa mesh radio
// using Meshtastic-compatible hardware connected via USB serial.
package mesh

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"
)

// MaxPayloadBytes is the maximum encoded JSON payload size per message.
const MaxPayloadBytes = 200

// Message types sent over the mesh.
const (
	TypeCmd      = "cmd" // Command (action, value).
	TypeAck      = "ack" // Acknowledgement of a prior message.
	TypeHeartbeat = "hb"  // Presence heartbeat.
	TypeLocation = "loc" // GPS location update.
	TypeMessage  = "msg" // Plain text message.
	TypeSync     = "syn" // State sync.
	TypeDiscover = "dis" // Discovery handshake.
)

// MeshMessage is the application-layer payload transmitted inside a Meshtastic packet.
// Keys are deliberately terse to fit within the 200-byte budget.
type MeshMessage struct {
	Type    string `json:"t"`           // Message type (cmd, ack, hb, loc, msg, syn, dis).
	ID      string `json:"id"`          // Unique 8-char message ID for deduplication.
	Sender  string `json:"s"`           // Sender node ID (e.g. "!abcd1234").
	Target  string `json:"to,omitempty"` // Target node ID; empty = broadcast.
	Action  string `json:"a,omitempty"` // Action/command (used with TypeCmd).
	Value   string `json:"v,omitempty"` // Value/content.
	Lat     int64  `json:"la,omitempty"` // Latitude fixed-point (degrees × 1,000,000).
	Lon     int64  `json:"lo,omitempty"` // Longitude fixed-point (degrees × 1,000,000).
	AgentName string `json:"ag,omitempty"` // Agent name (used in discovery).
	Owner   string `json:"ow,omitempty"` // Owner name (used in discovery).
	Persona string `json:"pe,omitempty"` // Agent persona (used in discovery).
}

// Encode marshals the message to JSON. Returns an error if the encoded size
// exceeds MaxPayloadBytes.
func Encode(m MeshMessage) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("mesh: encode: %w", err)
	}
	if len(data) > MaxPayloadBytes {
		return nil, fmt.Errorf("mesh: payload too large: %d bytes (max %d)", len(data), MaxPayloadBytes)
	}
	return data, nil
}

// Decode unmarshals a JSON payload into a MeshMessage. Returns an error for
// malformed JSON or unknown message types.
func Decode(data []byte) (MeshMessage, error) {
	var m MeshMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return MeshMessage{}, fmt.Errorf("mesh: decode: %w", err)
	}
	if !isKnownType(m.Type) {
		return MeshMessage{}, fmt.Errorf("mesh: unknown message type %q", m.Type)
	}
	return m, nil
}

// isKnownType returns true for recognised message type strings.
func isKnownType(t string) bool {
	switch t {
	case TypeCmd, TypeAck, TypeHeartbeat, TypeLocation, TypeMessage, TypeSync, TypeDiscover:
		return true
	}
	return false
}

// NewMessageID generates a random 8-character alphanumeric ID.
func NewMessageID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// EncodeLocation converts decimal degrees to fixed-point integers (× 1,000,000).
func EncodeLocation(lat, lon float64) (latFixed, lonFixed int64) {
	return int64(lat * 1_000_000), int64(lon * 1_000_000)
}

// DecodeLocation converts fixed-point integers back to decimal degrees.
func DecodeLocation(latFixed, lonFixed int64) (lat, lon float64) {
	return float64(latFixed) / 1_000_000, float64(lonFixed) / 1_000_000
}

// TruncateToFit trims text so that, when JSON-encoded inside a MeshMessage
// with the given fixed fields already allocated, the total payload stays within
// MaxPayloadBytes. Truncation happens at a word boundary where possible.
func TruncateToFit(text string, overhead int) string {
	budget := MaxPayloadBytes - overhead
	if budget <= 0 {
		return ""
	}
	if len(text) <= budget {
		return text
	}
	// Walk backward from budget to find a space (word boundary).
	truncated := text[:budget]
	if idx := strings.LastIndexByte(truncated, ' '); idx > 0 {
		truncated = truncated[:idx]
	}
	// Ensure we don't cut in the middle of a multi-byte rune.
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

// Deduplicator tracks recently seen message IDs using a sliding window.
type Deduplicator struct {
	window []string
	size   int
	seen   map[string]struct{}
}

// NewDeduplicator creates a Deduplicator with the given window size.
func NewDeduplicator(size int) *Deduplicator {
	return &Deduplicator{
		window: make([]string, 0, size),
		size:   size,
		seen:   make(map[string]struct{}, size),
	}
}

// IsDuplicate returns true if the message ID has been seen recently.
// If not a duplicate, the ID is recorded for future checks.
func (d *Deduplicator) IsDuplicate(id string) bool {
	if _, ok := d.seen[id]; ok {
		return true
	}
	// Add to window.
	if len(d.window) >= d.size {
		// Evict oldest.
		oldest := d.window[0]
		d.window = d.window[1:]
		delete(d.seen, oldest)
	}
	d.window = append(d.window, id)
	d.seen[id] = struct{}{}
	return false
}
