package mesh

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
)

// Meshtastic serial framing constants.
// Each packet is prefixed with a 4-byte header: START1 START2 SIZE_HIGH SIZE_LOW.
const (
	meshtasticStart1 = 0x94
	meshtasticStart2 = 0xC3

	defaultBaudRate    = 115200
	defaultQueueSize   = 16
	defaultRateLimit   = 30 * time.Second // 1 message per 30 seconds.
	drainShutdownTimeout = 5 * time.Second

	reconnectBaseDelay = 1 * time.Second
	reconnectMaxDelay  = 60 * time.Second
)

// SerialPort is the interface used by Transport for serial I/O.
// Satisfied by go.bug.st/serial Port; also allows injection of test fakes.
type SerialPort interface {
	io.ReadWriteCloser
	SetReadTimeout(t time.Duration) error
}

// TransportConfig holds configuration for a Transport instance.
type TransportConfig struct {
	PortPath  string
	BaudRate  int
	QueueSize int
	RateLimit time.Duration
}

// Transport manages the serial connection to a Meshtastic LoRa node,
// including send queuing, rate limiting, receive loop, and reconnection.
type Transport struct {
	cfg      TransportConfig
	port     SerialPort
	incoming chan MeshMessage
	outbound chan outboundMsg
	dedup    *Deduplicator
	mu       sync.RWMutex
	stopCh   chan struct{}
	doneCh   chan struct{}

	// openPort is injected for testing; defaults to openRealSerial.
	openPort func(path string, baud int) (SerialPort, error)
}

type outboundMsg struct {
	nodeID  string
	payload []byte
}

// NewTransport creates a Transport. Call Open to connect.
func NewTransport(cfg TransportConfig) *Transport {
	if cfg.BaudRate == 0 {
		cfg.BaudRate = defaultBaudRate
	}
	if cfg.QueueSize == 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.RateLimit == 0 {
		cfg.RateLimit = defaultRateLimit
	}
	return &Transport{
		cfg:      cfg,
		incoming: make(chan MeshMessage, 64),
		outbound: make(chan outboundMsg, cfg.QueueSize),
		dedup:    NewDeduplicator(256),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		openPort: openRealSerial,
	}
}

// Open establishes the serial connection and starts the send/receive loops.
func (t *Transport) Open() error {
	port, err := t.openPort(t.cfg.PortPath, t.cfg.BaudRate)
	if err != nil {
		return fmt.Errorf("mesh transport: open %s: %w", t.cfg.PortPath, err)
	}
	t.mu.Lock()
	t.port = port
	t.mu.Unlock()

	go t.receiveLoop()
	go t.sendLoop()
	return nil
}

// Send enqueues a message for transmission to the given nodeID.
// Returns an error if the transport is not connected or the queue is full.
func (t *Transport) Send(nodeID string, payload []byte) error {
	t.mu.RLock()
	connected := t.port != nil
	t.mu.RUnlock()
	if !connected {
		return fmt.Errorf("mesh transport: not connected")
	}
	msg := outboundMsg{nodeID: nodeID, payload: payload}
	select {
	case t.outbound <- msg:
		return nil
	default:
		// Queue full — drop oldest to make room.
		select {
		case <-t.outbound:
		default:
		}
		t.outbound <- msg
		return nil
	}
}

// Incoming returns a channel delivering decoded incoming messages.
func (t *Transport) Incoming() <-chan MeshMessage {
	return t.incoming
}

// Close drains the outbound queue (up to drainShutdownTimeout) then shuts down.
func (t *Transport) Close() {
	close(t.stopCh)
	// Wait for send loop to flush.
	select {
	case <-t.doneCh:
	case <-time.After(drainShutdownTimeout):
	}
	t.mu.Lock()
	if t.port != nil {
		t.port.Close()
		t.port = nil
	}
	t.mu.Unlock()
	close(t.incoming)
}

// receiveLoop reads Meshtastic-framed packets from serial and delivers decoded messages.
func (t *Transport) receiveLoop() {
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}

		t.mu.RLock()
		port := t.port
		t.mu.RUnlock()
		if port == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		data, err := t.readPacket(port)
		if err != nil {
			log.Printf("[mesh] receive error: %v", err)
			t.reconnect()
			continue
		}

		msg, err := Decode(data)
		if err != nil {
			log.Printf("[mesh] dropping malformed packet: %v", err)
			continue
		}

		// Deduplication.
		if t.dedup.IsDuplicate(msg.ID) {
			continue
		}

		select {
		case t.incoming <- msg:
		default:
			log.Printf("[mesh] incoming channel full, dropping message %s", msg.ID)
		}
	}
}

// sendLoop dequeues outbound messages and writes them at the configured rate.
func (t *Transport) sendLoop() {
	defer close(t.doneCh)
	limiter := time.NewTicker(t.cfg.RateLimit)
	defer limiter.Stop()

	for {
		select {
		case <-t.stopCh:
			// Drain remaining messages within the timeout.
			drain := time.After(drainShutdownTimeout)
		drainLoop:
			for {
				select {
				case msg := <-t.outbound:
					t.writePacket(msg.payload)
				case <-drain:
					break drainLoop
				default:
					break drainLoop
				}
			}
			return

		case msg := <-t.outbound:
			// Wait for rate limit tick before sending.
			<-limiter.C
			t.writePacket(msg.payload)
		}
	}
}

// readPacket reads one Meshtastic-framed packet from the port.
// Format: 0x94 0xC3 <size_high> <size_low> <payload...>
func (t *Transport) readPacket(port SerialPort) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(port, header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if header[0] != meshtasticStart1 || header[1] != meshtasticStart2 {
		return nil, fmt.Errorf("invalid framing: %02x %02x", header[0], header[1])
	}
	size := binary.BigEndian.Uint16(header[2:4])
	payload := make([]byte, size)
	if _, err := io.ReadFull(port, payload); err != nil {
		return nil, fmt.Errorf("read payload (%d bytes): %w", size, err)
	}
	return payload, nil
}

// writePacket frames payload with the Meshtastic header and writes it to serial.
func (t *Transport) writePacket(payload []byte) {
	t.mu.RLock()
	port := t.port
	t.mu.RUnlock()
	if port == nil {
		return
	}

	size := len(payload)
	frame := make([]byte, 4+size)
	frame[0] = meshtasticStart1
	frame[1] = meshtasticStart2
	frame[2] = byte(size >> 8)
	frame[3] = byte(size)
	copy(frame[4:], payload)

	if _, err := port.Write(frame); err != nil {
		log.Printf("[mesh] write error: %v", err)
		t.reconnect()
	}
}

// reconnect closes the current port and attempts to reopen with exponential backoff.
func (t *Transport) reconnect() {
	t.mu.Lock()
	if t.port != nil {
		t.port.Close()
		t.port = nil
	}
	t.mu.Unlock()

	delay := reconnectBaseDelay
	for {
		select {
		case <-t.stopCh:
			return
		case <-time.After(delay):
		}

		port, err := t.openPort(t.cfg.PortPath, t.cfg.BaudRate)
		if err != nil {
			log.Printf("[mesh] reconnect failed: %v (retry in %s)", err, delay)
			delay *= 2
			if delay > reconnectMaxDelay {
				delay = reconnectMaxDelay
			}
			continue
		}

		t.mu.Lock()
		t.port = port
		t.mu.Unlock()
		log.Printf("[mesh] reconnected to %s", t.cfg.PortPath)
		return
	}
}

// openRealSerial opens a real serial port, or a sim:// hub connection.
// If path starts with "sim://" (e.g. "sim://localhost:9090") it connects
// to a mesh-sim hub instead of opening a real serial device.
func openRealSerial(path string, baud int) (SerialPort, error) {
	if addr, ok := isSimAddr(path); ok {
		return openSimPort(addr)
	}
	mode := &serial.Mode{BaudRate: baud}
	port, err := serial.Open(path, mode)
	if err != nil {
		return nil, err
	}
	return port, nil
}
