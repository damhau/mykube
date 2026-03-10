package tunnel

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/damien/mykube/cli/internal/e2e"
	"nhooyr.io/websocket"
)

// connRegistry tracks active TCP connections keyed by connection ID.
type connRegistry struct {
	mu    sync.Mutex
	conns map[uint32]net.Conn
}

func newRegistry() *connRegistry {
	return &connRegistry{conns: make(map[uint32]net.Conn)}
}

func (r *connRegistry) add(id uint32, conn net.Conn) {
	r.mu.Lock()
	r.conns[id] = conn
	r.mu.Unlock()
}

func (r *connRegistry) get(id uint32) (net.Conn, bool) {
	r.mu.Lock()
	c, ok := r.conns[id]
	r.mu.Unlock()
	return c, ok
}

func (r *connRegistry) remove(id uint32) {
	r.mu.Lock()
	if c, ok := r.conns[id]; ok {
		c.Close()
		delete(r.conns, id)
	}
	r.mu.Unlock()
}

func (r *connRegistry) closeAll() {
	r.mu.Lock()
	for id, c := range r.conns {
		c.Close()
		delete(r.conns, id)
	}
	r.mu.Unlock()
}

// sendDone sends a "done:<connID>" text message over the WebSocket.
func sendDone(ctx context.Context, wsConn e2e.WSConn, wsMu *sync.Mutex, id uint32) {
	wsMu.Lock()
	wsConn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("done:%d", id)))
	wsMu.Unlock()
}

// writeFrame sends a binary WebSocket message with a 4-byte connID header.
func writeFrame(ctx context.Context, wsConn e2e.WSConn, wsMu *sync.Mutex, id uint32, payload []byte) error {
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], id)
	copy(frame[4:], payload)
	wsMu.Lock()
	err := wsConn.Write(ctx, websocket.MessageBinary, frame)
	wsMu.Unlock()
	return err
}

// tcpToWS reads from a TCP connection and writes framed binary messages to the WebSocket.
// On EOF or error it sends done:<connID> and removes the connection from the registry.
func tcpToWS(ctx context.Context, wsConn e2e.WSConn, wsMu *sync.Mutex, id uint32, tcpConn net.Conn, reg *connRegistry) {
	buf := make([]byte, 32*1024)
	for {
		n, err := tcpConn.Read(buf)
		if n > 0 {
			if writeErr := writeFrame(ctx, wsConn, wsMu, id, buf[:n]); writeErr != nil {
				fmt.Fprintf(os.Stderr, "[bridge] tcp→ws write error (conn %d): %v\n", id, writeErr)
				reg.remove(id)
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				// Don't log "use of closed network connection" — expected when
				// the WS read loop closes TCP after receiving "done"
			}
			sendDone(ctx, wsConn, wsMu, id)
			reg.remove(id)
			return
		}
	}
}

// parseControl parses a text message like "new:42" or "done:42".
// Returns (command, connID, ok).
func parseControl(data string) (string, uint32, bool) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	id, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return "", 0, false
	}
	return parts[0], uint32(id), true
}

// ServeClient accepts TCP connections on the listener and multiplexes them
// through the WebSocket using binary framing with connection IDs.
func ServeClient(ctx context.Context, wsConn e2e.WSConn, listener net.Listener) {
	var wsMu sync.Mutex
	var nextID atomic.Uint32
	reg := newRegistry()
	defer reg.closeAll()

	// Accept TCP connections concurrently
	go func() {
		for {
			tcpConn, err := listener.Accept()
			if err != nil {
				if ctx.Err() == nil {
					fmt.Fprintf(os.Stderr, "[client] accept error: %v\n", err)
				}
				return
			}

			id := nextID.Add(1)
			reg.add(id, tcpConn)

			// Signal the agent side to open a new connection
			wsMu.Lock()
			err = wsConn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("new:%d", id)))
			wsMu.Unlock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[client] ws write error: %v\n", err)
				reg.remove(id)
				return
			}

			fmt.Fprintf(os.Stderr, "[client] New connection %d\n", id)
			go tcpToWS(ctx, wsConn, &wsMu, id, tcpConn, reg)
		}
	}()

	// Single WS read loop — dispatch incoming frames by connID
	for {
		typ, data, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[client] ws read error: %v\n", err)
			}
			return
		}

		if typ == websocket.MessageText {
			cmd, id, ok := parseControl(string(data))
			if !ok {
				continue
			}
			if cmd == "done" {
				reg.remove(id)
			}
			continue
		}

		// Binary frame: [4-byte connID][payload]
		if len(data) < 4 {
			continue
		}
		id := binary.BigEndian.Uint32(data[:4])
		payload := data[4:]
		if conn, ok := reg.get(id); ok {
			if _, err := conn.Write(payload); err != nil {
				fmt.Fprintf(os.Stderr, "[client] ws→tcp write error (conn %d): %v\n", id, err)
				reg.remove(id)
			}
		}
	}
}

// ServeAgent reads control signals and framed data from the WebSocket,
// dials the API server for each new connection, and bridges traffic.
func ServeAgent(ctx context.Context, wsConn e2e.WSConn, apiServerHost string) {
	var wsMu sync.Mutex
	reg := newRegistry()
	defer reg.closeAll()

	for {
		typ, data, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[agent] ws read error: %v\n", err)
			}
			return
		}

		if typ == websocket.MessageText {
			cmd, id, ok := parseControl(string(data))
			if !ok {
				continue
			}
			switch cmd {
			case "new":
				fmt.Fprintf(os.Stderr, "[agent] New connection %d, dialing %s...\n", id, apiServerHost)
				tcpConn, err := net.Dial("tcp", apiServerHost)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[agent] dial error (conn %d): %v\n", id, err)
					sendDone(ctx, wsConn, &wsMu, id)
					continue
				}
				reg.add(id, tcpConn)
				go tcpToWS(ctx, wsConn, &wsMu, id, tcpConn, reg)
			case "done":
				reg.remove(id)
			}
			continue
		}

		// Binary frame: [4-byte connID][payload]
		if len(data) < 4 {
			continue
		}
		id := binary.BigEndian.Uint32(data[:4])
		payload := data[4:]
		if conn, ok := reg.get(id); ok {
			if _, err := conn.Write(payload); err != nil {
				fmt.Fprintf(os.Stderr, "[agent] ws→tcp write error (conn %d): %v\n", id, err)
				reg.remove(id)
			}
		}
	}
}
