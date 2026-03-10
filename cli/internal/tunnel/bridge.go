package tunnel

import (
	"bytes"
	"context"
	"io"
	"net"

	"nhooyr.io/websocket"
)

// Bridge bidirectionally copies data between a WebSocket connection and a TCP connection.
// It closes both connections when either direction encounters an error or the context is cancelled.
func Bridge(ctx context.Context, wsConn *websocket.Conn, tcpConn net.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{}, 2)

	// TCP -> WebSocket (binary frames)
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := tcpConn.Read(buf)
			if n > 0 {
				if writeErr := wsConn.Write(ctx, websocket.MessageBinary, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> TCP
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, data, err := wsConn.Read(ctx)
			if err != nil {
				return
			}
			if _, err := io.Copy(tcpConn, bytes.NewReader(data)); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to finish
	<-done
	cancel()
	tcpConn.Close()
	wsConn.Close(websocket.StatusNormalClosure, "bridge closed")
}
