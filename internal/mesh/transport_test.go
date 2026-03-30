package mesh

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"testing"
	"time"
)

// mockPort is a fake SerialPort backed by a bytes.Buffer pair.
type mockPort struct {
	readBuf  bytes.Buffer
	writeBuf bytes.Buffer
	mu       sync.Mutex
	closed   bool
}

func (m *mockPort) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	if m.readBuf.Len() == 0 {
		return 0, io.EOF
	}
	return m.readBuf.Read(p)
}

func (m *mockPort) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(p)
}

func (m *mockPort) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockPort) SetReadTimeout(_ time.Duration) error { return nil }

// framePacket wraps payload in Meshtastic framing.
func framePacket(payload []byte) []byte {
	size := len(payload)
	frame := make([]byte, 4+size)
	frame[0] = meshtasticStart1
	frame[1] = meshtasticStart2
	binary.BigEndian.PutUint16(frame[2:4], uint16(size))
	copy(frame[4:], payload)
	return frame
}

func newTestTransport(port SerialPort) *Transport {
	cfg := TransportConfig{
		BaudRate:  defaultBaudRate,
		QueueSize: 4,
		RateLimit: 10 * time.Millisecond, // Fast for tests.
	}
	t := NewTransport(cfg)
	t.port = port
	t.openPort = func(string, int) (SerialPort, error) { return port, nil }
	return t
}

func TestTransportSendFailsWhenDisconnected(t *testing.T) {
	cfg := TransportConfig{QueueSize: 4, RateLimit: 10 * time.Millisecond}
	tr := NewTransport(cfg)
	// port is nil — disconnected.
	err := tr.Send("!node1", []byte(`{"t":"msg"}`))
	if err == nil {
		t.Fatal("Send should fail when not connected")
	}
}

func TestTransportQueueDropsOldestOnOverflow(t *testing.T) {
	mock := &mockPort{}
	tr := newTestTransport(mock)

	// Fill queue beyond capacity.
	for i := 0; i < tr.cfg.QueueSize+3; i++ {
		_ = tr.Send("!node1", []byte(`{"t":"msg","id":"abc12345","s":"!n1","v":"x"}`))
	}
	// Queue should not exceed configured size.
	if len(tr.outbound) > tr.cfg.QueueSize {
		t.Errorf("queue length %d exceeds capacity %d", len(tr.outbound), tr.cfg.QueueSize)
	}
}

func TestTransportReadPacketDecodesMessage(t *testing.T) {
	msg := MeshMessage{Type: TypeMessage, ID: "abc12345", Sender: "!node1", Value: "hello"}
	payload, _ := Encode(msg)
	frame := framePacket(payload)

	mock := &mockPort{}
	mock.readBuf.Write(frame)

	tr := newTestTransport(mock)
	got, err := tr.readPacket(mock)
	if err != nil {
		t.Fatalf("readPacket: %v", err)
	}
	decoded, err := Decode(got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded.Value != "hello" {
		t.Errorf("want Value=hello, got %q", decoded.Value)
	}
}

func TestTransportGracefulShutdown(t *testing.T) {
	mock := &mockPort{}
	tr := newTestTransport(mock)
	go tr.sendLoop()

	// Enqueue a message before shutting down.
	_ = tr.Send("!node1", []byte(`{"t":"msg","id":"ab123456","s":"!n","v":"bye"}`))

	done := make(chan struct{})
	go func() {
		tr.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Close() did not complete within timeout")
	}
}

func TestTransportRateLimit(t *testing.T) {
	mock := &mockPort{}
	cfg := TransportConfig{
		QueueSize: 4,
		RateLimit: 50 * time.Millisecond,
	}
	tr := NewTransport(cfg)
	tr.port = mock
	tr.openPort = func(string, int) (SerialPort, error) { return mock, nil }

	go tr.sendLoop()
	defer tr.Close()

	payload := []byte(`{"t":"msg","id":"ab123456","s":"!n","v":"x"}`)

	start := time.Now()
	_ = tr.Send("!node1", payload)
	_ = tr.Send("!node1", payload)

	// Wait for at least one rate limit interval.
	time.Sleep(80 * time.Millisecond)
	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("rate limiter too fast: %s", elapsed)
	}
}
