// mesh-sim is a virtual LoRa radio hub for local development and testing.
//
// It accepts TCP connections from DUSTY instances configured with
//
//	serial_port = "sim://localhost:9090"
//
// and relays framed Meshtastic packets between all connected nodes, mimicking
// the broadcast nature of LoRa radio (everyone hears everything).
//
// Usage:
//
//	go run ./cmd/mesh-sim            # listen on default port 9090
//	go run ./cmd/mesh-sim -port 9091 # custom port
//
// Quick-start (two terminals):
//
//  1. Terminal 1:  go run ./cmd/mesh-sim
//  2. Terminal 2:  DUSTY_NODE=a go run ./cmd/dusty -config configs/sim-node-a.toml
//  3. Terminal 3:  DUSTY_NODE=b go run ./cmd/dusty -config configs/sim-node-b.toml
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const (
	meshtasticStart1 = 0x94
	meshtasticStart2 = 0xC3
	maxPacketBytes   = 512 // safety cap
)

func main() {
	port := flag.Int("port", 9090, "TCP port to listen on")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("mesh-sim: listen on %s: %v", addr, err)
	}
	log.Printf("mesh-sim: virtual LoRa hub listening on %s", addr)
	log.Printf("mesh-sim: configure nodes with:  serial_port = \"sim://localhost:%d\"", *port)

	hub := newHub()
	go hub.run()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("mesh-sim: accept error: %v", err)
			continue
		}
		log.Printf("mesh-sim: node connected from %s (total: %d)", conn.RemoteAddr(), hub.count()+1)
		client := &client{conn: conn, send: make(chan []byte, 32)}
		hub.register <- client
		go client.readPump(hub)
		go client.writePump()
	}
}

// ─── Hub ────────────────────────────────────────────────────────────────────

type hub struct {
	clients    map[*client]struct{}
	broadcast  chan broadcast
	register   chan *client
	unregister chan *client
	mu         sync.RWMutex
}

type broadcast struct {
	from   *client
	frame  []byte
}

func newHub() *hub {
	return &hub{
		clients:    make(map[*client]struct{}),
		broadcast:  make(chan broadcast, 64),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

func (h *hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			log.Printf("mesh-sim: node disconnected from %s (remaining: %d)", c.conn.RemoteAddr(), h.count())

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				if c == msg.from {
					continue // don't echo back to sender
				}
				select {
				case c.send <- msg.frame:
				default:
					// Client is slow — drop frame for this recipient.
					log.Printf("mesh-sim: dropped frame for slow client %s", c.conn.RemoteAddr())
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ─── Client ─────────────────────────────────────────────────────────────────

type client struct {
	conn net.Conn
	send chan []byte
}

// readPump reads complete Meshtastic frames from the connection and
// forwards them to the hub for broadcast.
func (c *client) readPump(h *hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()

	for {
		frame, err := readFrame(c.conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("mesh-sim: read error from %s: %v", c.conn.RemoteAddr(), err)
			}
			return
		}
		h.broadcast <- broadcast{from: c, frame: frame}
	}
}

// writePump drains the send channel and writes frames to the connection.
func (c *client) writePump() {
	defer c.conn.Close()
	for frame := range c.send {
		if _, err := c.conn.Write(frame); err != nil {
			log.Printf("mesh-sim: write error to %s: %v", c.conn.RemoteAddr(), err)
			return
		}
	}
}

// readFrame reads one complete Meshtastic-framed packet.
// Format: 0x94 0xC3 <size_high> <size_low> <payload...>
func readFrame(r io.Reader) ([]byte, error) {
	// Sync to frame start — scan byte-by-byte until we see 0x94 0xC3.
	var b [1]byte
	for {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return nil, err
		}
		if b[0] != meshtasticStart1 {
			continue
		}
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return nil, err
		}
		if b[0] == meshtasticStart2 {
			break
		}
		// Second byte didn't match — keep scanning from current position.
	}

	// Read 2-byte size.
	var sizeBuf [2]byte
	if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
		return nil, fmt.Errorf("read size: %w", err)
	}
	size := int(binary.BigEndian.Uint16(sizeBuf[:]))
	if size == 0 || size > maxPacketBytes {
		return nil, fmt.Errorf("invalid payload size %d", size)
	}

	// Read payload.
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	// Reassemble the full frame so recipients can feed it directly into their
	// transport's readPacket.
	frame := make([]byte, 4+size)
	frame[0] = meshtasticStart1
	frame[1] = meshtasticStart2
	frame[2] = sizeBuf[0]
	frame[3] = sizeBuf[1]
	copy(frame[4:], payload)
	return frame, nil
}
