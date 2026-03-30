package mesh

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// simPort wraps a net.Conn as a SerialPort.
// It connects to a mesh-sim hub (see cmd/mesh-sim) which broadcasts
// framed packets between all connected nodes.
type simPort struct {
	conn net.Conn
}

func (s *simPort) Read(p []byte) (int, error)  { return s.conn.Read(p) }
func (s *simPort) Write(p []byte) (int, error) { return s.conn.Write(p) }
func (s *simPort) Close() error                { return s.conn.Close() }
func (s *simPort) SetReadTimeout(t time.Duration) error {
	if t == 0 {
		return s.conn.SetReadDeadline(time.Time{}) // disable
	}
	return s.conn.SetReadDeadline(time.Now().Add(t))
}

// openSimPort connects to a mesh-sim hub at addr (e.g. "localhost:9090").
// Used when serial_port starts with "sim://".
func openSimPort(addr string) (SerialPort, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("sim: connect to hub at %s: %w", addr, err)
	}
	return &simPort{conn: conn}, nil
}

// isSimAddr returns true and the hub address if path is a sim:// URL.
func isSimAddr(path string) (string, bool) {
	const prefix = "sim://"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix), true
	}
	return "", false
}
