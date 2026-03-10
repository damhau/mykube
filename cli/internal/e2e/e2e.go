package e2e

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"nhooyr.io/websocket"
)

// WSConn is the minimal interface for WebSocket read/write operations.
// Both *websocket.Conn and *EncryptedConn satisfy this interface.
type WSConn interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
}

// EncryptedConn wraps a raw WebSocket connection with AES-256-GCM encryption.
// All messages are sent as binary with the original message type encoded inside
// the encrypted envelope.
type EncryptedConn struct {
	inner *websocket.Conn
	aead  cipher.AEAD
}

func (e *EncryptedConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	_, data, err := e.inner.Read(ctx)
	if err != nil {
		return 0, nil, err
	}
	nonceSize := e.aead.NonceSize()
	if len(data) < 1+nonceSize {
		return 0, nil, fmt.Errorf("e2e: message too short")
	}
	origType := websocket.MessageType(data[0])
	nonce := data[1 : 1+nonceSize]
	ciphertext := data[1+nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("e2e: decrypt failed: %w", err)
	}
	return origType, plaintext, nil
}

func (e *EncryptedConn) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	nonceSize := e.aead.NonceSize()
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("e2e: generate nonce: %w", err)
	}
	ciphertext := e.aead.Seal(nil, nonce, p, nil)
	frame := make([]byte, 1+nonceSize+len(ciphertext))
	frame[0] = byte(typ)
	copy(frame[1:1+nonceSize], nonce)
	copy(frame[1+nonceSize:], ciphertext)
	return e.inner.Write(ctx, websocket.MessageBinary, frame)
}

// KeyExchange performs X25519 ECDH key exchange over the WebSocket and returns
// an encrypted connection. isInitiator should be true for the agent (sends pubkey
// first) and false for the client (receives pubkey first).
func KeyExchange(ctx context.Context, ws *websocket.Conn, isInitiator bool) (*EncryptedConn, error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("e2e: generate key: %w", err)
	}
	pubMsg := "e2e:" + base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes())

	var peerPubBytes []byte

	if isInitiator {
		if err := ws.Write(ctx, websocket.MessageText, []byte(pubMsg)); err != nil {
			return nil, fmt.Errorf("e2e: send pubkey: %w", err)
		}
		peerPubBytes, err = readPubKey(ctx, ws)
		if err != nil {
			return nil, err
		}
	} else {
		peerPubBytes, err = readPubKey(ctx, ws)
		if err != nil {
			return nil, err
		}
		if err := ws.Write(ctx, websocket.MessageText, []byte(pubMsg)); err != nil {
			return nil, fmt.Errorf("e2e: send pubkey: %w", err)
		}
	}

	peerPub, err := ecdh.X25519().NewPublicKey(peerPubBytes)
	if err != nil {
		return nil, fmt.Errorf("e2e: invalid peer pubkey: %w", err)
	}
	sharedSecret, err := priv.ECDH(peerPub)
	if err != nil {
		return nil, fmt.Errorf("e2e: ECDH: %w", err)
	}

	key := sha256.Sum256(sharedSecret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("e2e: create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("e2e: create GCM: %w", err)
	}

	return &EncryptedConn{inner: ws, aead: aead}, nil
}

func readPubKey(ctx context.Context, ws *websocket.Conn) ([]byte, error) {
	typ, data, err := ws.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("e2e: read pubkey: %w", err)
	}
	if typ != websocket.MessageText {
		return nil, fmt.Errorf("e2e: expected text message for pubkey, got %v", typ)
	}
	msg := string(data)
	if !strings.HasPrefix(msg, "e2e:") {
		return nil, fmt.Errorf("e2e: expected e2e: prefix, got: %s", msg)
	}
	pubBytes, err := base64.RawURLEncoding.DecodeString(msg[4:])
	if err != nil {
		return nil, fmt.Errorf("e2e: decode pubkey: %w", err)
	}
	return pubBytes, nil
}
