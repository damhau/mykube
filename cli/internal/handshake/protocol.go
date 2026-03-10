package handshake

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/damien/mykube/cli/internal/e2e"
	"nhooyr.io/websocket"
)

type Handshake struct {
	ClusterName string `json:"cluster_name"`
	CAData      string `json:"ca_data"`
	Token       string `json:"token,omitempty"`
	ClientCert  string `json:"client_cert,omitempty"`
	ClientKey   string `json:"client_key,omitempty"`
}

func (h *Handshake) Send(ctx context.Context, conn e2e.WSConn) error {
	data, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshal handshake: %w", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}
	return nil
}

func Receive(ctx context.Context, conn e2e.WSConn) (*Handshake, error) {
	typ, data, err := conn.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("read handshake: %w", err)
	}
	if typ != websocket.MessageText {
		return nil, fmt.Errorf("expected text message for handshake, got %v", typ)
	}
	var h Handshake
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("unmarshal handshake: %w", err)
	}
	return &h, nil
}
