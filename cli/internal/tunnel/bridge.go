package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"nhooyr.io/websocket"
)

// ServeAgent reads "new" signals from the WebSocket, dials the API server for each one,
// and bridges traffic until the connection ends. Loops until context is cancelled.
func ServeAgent(ctx context.Context, wsConn *websocket.Conn, apiServerHost string) {
	var wsMu sync.Mutex

	for {
		typ, data, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[agent] ws read error: %v\n", err)
			}
			return
		}
		if typ != websocket.MessageText || string(data) != "new" {
			continue
		}

		fmt.Fprintf(os.Stderr, "[agent] New connection, dialing %s...\n", apiServerHost)
		tcpConn, err := net.Dial("tcp", apiServerHost)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[agent] dial error: %v\n", err)
			wsMu.Lock()
			wsConn.Write(ctx, websocket.MessageText, []byte("done"))
			wsMu.Unlock()
			continue
		}

		bridgeOneConnection(ctx, wsConn, tcpConn, &wsMu)
		fmt.Fprintf(os.Stderr, "[agent] Connection ended.\n")
	}
}

// ServeClient accepts TCP connections on the listener and bridges each one
// through the WebSocket. One connection at a time (MVP).
func ServeClient(ctx context.Context, wsConn *websocket.Conn, listener net.Listener) {
	var wsMu sync.Mutex

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[client] accept error: %v\n", err)
			}
			return
		}

		wsMu.Lock()
		err = wsConn.Write(ctx, websocket.MessageText, []byte("new"))
		wsMu.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[client] ws write error: %v\n", err)
			tcpConn.Close()
			return
		}

		bridgeOneConnection(ctx, wsConn, tcpConn, &wsMu)
	}
}

// bridgeOneConnection bridges a single TCP connection through the WebSocket.
// Both sides always send "done" when their TCP goroutine exits — this ensures
// the remote side's WS read loop can unblock and return.
// The WebSocket is NOT closed — it's reused for the next connection.
func bridgeOneConnection(ctx context.Context, wsConn *websocket.Conn, tcpConn net.Conn, wsMu *sync.Mutex) {
	tcpDone := make(chan struct{})

	// TCP → WS goroutine: reads from TCP, writes binary to WS.
	// Always sends "done" when exiting (whether EOF, error, or closed by us).
	go func() {
		defer close(tcpDone)
		buf := make([]byte, 32*1024)
		for {
			n, err := tcpConn.Read(buf)
			if n > 0 {
				wsMu.Lock()
				writeErr := wsConn.Write(ctx, websocket.MessageBinary, buf[:n])
				wsMu.Unlock()
				if writeErr != nil {
					fmt.Fprintf(os.Stderr, "[bridge] tcp→ws write error: %v\n", writeErr)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					// Don't log "use of closed network connection" — that's expected
					// when the WS read loop closes TCP after receiving "done"
				}
				wsMu.Lock()
				wsConn.Write(ctx, websocket.MessageText, []byte("done"))
				wsMu.Unlock()
				return
			}
		}
	}()

	// WS → TCP: read from WS, write to TCP. Exits on "done" text message.
	for {
		typ, data, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[bridge] ws read error: %v\n", err)
			}
			tcpConn.Close()
			<-tcpDone
			return
		}
		if typ == websocket.MessageText && string(data) == "done" {
			tcpConn.Close()
			<-tcpDone
			return
		}
		if _, err := tcpConn.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "[bridge] ws→tcp write error: %v\n", err)
			tcpConn.Close()
			<-tcpDone
			return
		}
	}
}
