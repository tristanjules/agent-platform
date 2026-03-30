package mesh

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// startTestHub runs a minimal in-process echo hub for sim transport tests.
// It returns the listen address and a stop function.
func startTestHub(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr = ln.Addr().String()

	// Simple hub: two clients, relay frames between them.
	clients := make(chan net.Conn, 8)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			clients <- conn
		}
	}()

	stop = func() { ln.Close() }
	_ = clients
	return addr, stop
}

func TestSimPortConnects(t *testing.T) {
	// Start a TCP listener acting as the hub.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	// Accept one connection in background.
	accepted := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		close(accepted)
		conn.Close()
	}()

	port, err := openSimPort(addr)
	if err != nil {
		t.Fatalf("openSimPort: %v", err)
	}
	defer port.Close()

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Error("hub did not accept connection within 1s")
	}
}

func TestSimPortReadWrite(t *testing.T) {
	// Create a pair of connected TCP sockets (loopback hub).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	// Hub side: echo back whatever it receives.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			_, _ = conn.Write(buf[:n])
		}
	}()

	port, err := openSimPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer port.Close()

	// Write a test frame.
	msg := MeshMessage{Type: TypeMessage, ID: "abc12345", Sender: "!sim01", Value: "hello sim"}
	payload, _ := Encode(msg)
	frame := framePacket(payload)
	if _, err := port.Write(frame); err != nil {
		t.Fatal(err)
	}

	// Read it back (echoed).
	buf := make([]byte, len(frame))
	_ = port.SetReadTimeout(500 * time.Millisecond)
	n, err := port.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if n != len(frame) {
		t.Errorf("want %d bytes echoed, got %d", len(frame), n)
	}
}

func TestIsSimAddr(t *testing.T) {
	cases := []struct {
		path    string
		want    string
		wantOK  bool
	}{
		{"sim://localhost:9090", "localhost:9090", true},
		{"sim://127.0.0.1:9091", "127.0.0.1:9091", true},
		{"/dev/ttyUSB0", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		addr, ok := isSimAddr(tc.path)
		if ok != tc.wantOK {
			t.Errorf("%q: want ok=%v, got %v", tc.path, tc.wantOK, ok)
		}
		if addr != tc.want {
			t.Errorf("%q: want addr=%q, got %q", tc.path, tc.want, addr)
		}
	}
}

func TestSimTransportSendReceive(t *testing.T) {
	// Use the real mesh-sim hub logic (inlined) with two simPort clients.
	// Hub: forward packets from A to B and vice versa.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	// Minimal two-client relay hub.
	type pair struct{ a, b net.Conn }
	pairCh := make(chan pair, 1)
	go func() {
		connA, err := ln.Accept()
		if err != nil {
			return
		}
		connB, err := ln.Accept()
		if err != nil {
			connA.Close()
			return
		}
		pairCh <- pair{connA, connB}
		// Relay A→B and B→A.
		relay := func(src, dst net.Conn) {
			buf := make([]byte, 512)
			for {
				n, err := src.Read(buf)
				if err != nil || n == 0 {
					return
				}
				_, _ = dst.Write(buf[:n])
			}
		}
		go relay(connA, connB)
		go relay(connB, connA)
	}()

	portA, err := openSimPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer portA.Close()
	portB, err := openSimPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer portB.Close()

	// Wait for hub to pair them.
	select {
	case <-pairCh:
	case <-time.After(time.Second):
		t.Fatal("hub did not pair clients")
	}

	// Use transports backed by the sim ports.
	cfgA := TransportConfig{QueueSize: 4, RateLimit: 5 * time.Millisecond}
	trA := NewTransport(cfgA)
	trA.port = portA
	trA.openPort = func(string, int) (SerialPort, error) { return portA, nil }

	cfgB := TransportConfig{QueueSize: 4, RateLimit: 5 * time.Millisecond}
	trB := NewTransport(cfgB)
	trB.port = portB
	trB.openPort = func(string, int) (SerialPort, error) { return portB, nil }

	// Start loops.
	go trA.sendLoop()
	go trB.receiveLoop()
	go trB.sendLoop()
	defer trA.Close()
	defer trB.Close()

	// A sends a message; B should receive it.
	msg := MeshMessage{Type: TypeMessage, ID: "sim00001", Sender: "!aaaaaaaa", Value: "hello from A"}
	payload, _ := Encode(msg)
	if err := trA.Send("!bbbbbbbb", payload); err != nil {
		t.Fatal(err)
	}

	var received MeshMessage
	select {
	case received = <-trB.Incoming():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("node B did not receive message from A")
	}
	if received.ID != msg.ID {
		t.Errorf("want ID %q, got %q", msg.ID, received.ID)
	}
	if received.Value != msg.Value {
		t.Errorf("want value %q, got %q", msg.Value, received.Value)
	}
	fmt.Printf("✓ Node B received: %q\n", received.Value)
}
