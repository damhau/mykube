package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nhooyr.io/websocket"
)

type RelayClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c *RelayClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
	Code      string `json:"code"`
}

type pairResponse struct {
	SessionID string `json:"session_id"`
}

func (c *RelayClient) CreateSession() (sessionID, code string, err error) {
	resp, err := c.httpClient().Post(c.BaseURL+"/api/sessions", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("create session: unexpected status %d", resp.StatusCode)
	}

	var result createSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("create session: decode response: %w", err)
	}
	return result.SessionID, result.Code, nil
}

func (c *RelayClient) PairSession(code string) (sessionID string, err error) {
	body, _ := json.Marshal(map[string]string{"code": code})
	resp, err := c.httpClient().Post(c.BaseURL+"/api/pair", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("pair session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pair session: unexpected status %d", resp.StatusCode)
	}

	var result pairResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("pair session: decode response: %w", err)
	}
	return result.SessionID, nil
}

func (c *RelayClient) dialOpts() *websocket.DialOptions {
	return &websocket.DialOptions{HTTPClient: c.httpClient()}
}

func (c *RelayClient) ConnectAgent(ctx context.Context, sessionID string) (*websocket.Conn, error) {
	wsURL := strings.Replace(c.BaseURL, "http", "ws", 1) + "/ws/agent/" + sessionID
	conn, _, err := websocket.Dial(ctx, wsURL, c.dialOpts())
	if err != nil {
		return nil, fmt.Errorf("connect agent ws: %w", err)
	}
	return conn, nil
}

func (c *RelayClient) ConnectClient(ctx context.Context, sessionID string) (*websocket.Conn, error) {
	wsURL := strings.Replace(c.BaseURL, "http", "ws", 1) + "/ws/client/" + sessionID
	conn, _, err := websocket.Dial(ctx, wsURL, c.dialOpts())
	if err != nil {
		return nil, fmt.Errorf("connect client ws: %w", err)
	}
	return conn, nil
}
