package mesh

import (
	"fmt"
	"testing"
	"time"
)

// TestSimE2E_TwoNodesViaHub connects two transports through a real
// mesh-sim hub (must be running on localhost:9090) and verifies that
// a message sent by node A arrives at node B.
//
// Run with: go test ./internal/mesh -run TestSimE2E -v
// Requires: go run ./cmd/mesh-sim  (in another terminal)
func TestSimE2E_TwoNodesViaHub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	hubAddr := "localhost:9090"

	// --- Node A ---
	portA, err := openSimPort(hubAddr)
	if err != nil {
		t.Skipf("mesh-sim hub not running at %s: %v (start with: go run ./cmd/mesh-sim)", hubAddr, err)
	}
	trA := NewTransport(TransportConfig{
		PortPath: "sim://" + hubAddr, BaudRate: 115200,
		QueueSize: 8, RateLimit: 10 * time.Millisecond,
	})
	trA.port = portA
	trA.openPort = func(string, int) (SerialPort, error) { return portA, nil }
	go trA.receiveLoop()
	go trA.sendLoop()
	defer trA.Close()

	// --- Node B ---
	portB, err := openSimPort(hubAddr)
	if err != nil {
		t.Fatalf("second connection to hub failed: %v", err)
	}
	trB := NewTransport(TransportConfig{
		PortPath: "sim://" + hubAddr, BaudRate: 115200,
		QueueSize: 8, RateLimit: 10 * time.Millisecond,
	})
	trB.port = portB
	trB.openPort = func(string, int) (SerialPort, error) { return portB, nil }
	go trB.receiveLoop()
	go trB.sendLoop()
	defer trB.Close()

	time.Sleep(50 * time.Millisecond)

	// --- A sends to B ---
	msg := MeshMessage{
		Type:   TypeMessage,
		ID:     NewMessageID(),
		Sender: "!aaaaaaaa",
		Target: "!bbbbbbbb",
		Value:  "hello from node A",
	}
	payload, err := Encode(msg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	t.Logf("Sending %d bytes (msg ID=%s): %q", len(payload), msg.ID, msg.Value)

	if err := trA.Send("!bbbbbbbb", payload); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case received := <-trB.Incoming():
		t.Logf("Node B received: ID=%s Value=%q", received.ID, received.Value)
		if received.ID != msg.ID {
			t.Errorf("ID mismatch: want %q got %q", msg.ID, received.ID)
		}
		if received.Value != msg.Value {
			t.Errorf("Value mismatch: want %q got %q", msg.Value, received.Value)
		}
		fmt.Printf("✓ E2E: Node B received from A: %q\n", received.Value)
	case <-time.After(3 * time.Second):
		t.Fatal("TIMEOUT: Node B did not receive message from A within 3s")
	}
}
