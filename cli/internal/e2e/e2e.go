package e2e

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
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

const (
	hkdfInfo    = "mykube-e2e-v1"
	sasInfo     = "mykube-sas-v1"
	sasTagLen   = 32
	hkdfKeyLen  = 32 // AES-256
)

// KeyExchange performs X25519 ECDH key exchange over the WebSocket and returns
// an encrypted connection. The pairingCode is bound into key derivation via HKDF
// so that a MITM who doesn't know the code produces a different SAS tag, causing
// the verification step to fail. isInitiator should be true for the agent (sends
// pubkey first) and false for the client (receives pubkey first).
func KeyExchange(ctx context.Context, ws *websocket.Conn, isInitiator bool, pairingCode string) (*EncryptedConn, error) {
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

	// Derive session key using HKDF with the pairing code as salt.
	salt := sha256.Sum256([]byte(pairingCode))
	sessionKey := make([]byte, hkdfKeyLen)
	kdf := hkdf.New(sha256.New, sharedSecret, salt[:], []byte(hkdfInfo))
	if _, err := io.ReadFull(kdf, sessionKey); err != nil {
		return nil, fmt.Errorf("e2e: derive session key: %w", err)
	}

	// Derive SAS verification tag (separate HKDF context).
	sasTag := make([]byte, sasTagLen)
	sasKdf := hkdf.New(sha256.New, sharedSecret, salt[:], []byte(sasInfo))
	if _, err := io.ReadFull(sasKdf, sasTag); err != nil {
		return nil, fmt.Errorf("e2e: derive SAS tag: %w", err)
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("e2e: create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("e2e: create GCM: %w", err)
	}

	enc := &EncryptedConn{inner: ws, aead: aead}

	// Exchange and verify SAS tags over the encrypted channel.
	if err := verifySAS(ctx, enc, sasTag, isInitiator); err != nil {
		return nil, err
	}

	return enc, nil
}

// verifySAS exchanges SAS tags over the encrypted connection and verifies they
// match. The initiator sends first, then reads; the responder reads first, then
// sends. A mismatch means a MITM is present.
func verifySAS(ctx context.Context, enc *EncryptedConn, localTag []byte, isInitiator bool) error {
	sasMsg := []byte("sas:" + base64.RawURLEncoding.EncodeToString(localTag))

	if isInitiator {
		if err := enc.Write(ctx, websocket.MessageText, sasMsg); err != nil {
			return fmt.Errorf("e2e: send SAS tag: %w", err)
		}
		peerTag, err := readSASTag(ctx, enc)
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare(localTag, peerTag) != 1 {
			return fmt.Errorf("e2e: SAS verification failed — possible MITM attack")
		}
	} else {
		peerTag, err := readSASTag(ctx, enc)
		if err != nil {
			return err
		}
		if err := enc.Write(ctx, websocket.MessageText, sasMsg); err != nil {
			return fmt.Errorf("e2e: send SAS tag: %w", err)
		}
		if subtle.ConstantTimeCompare(localTag, peerTag) != 1 {
			return fmt.Errorf("e2e: SAS verification failed — possible MITM attack")
		}
	}
	return nil
}

func readSASTag(ctx context.Context, enc *EncryptedConn) ([]byte, error) {
	typ, data, err := enc.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("e2e: read SAS tag: %w", err)
	}
	if typ != websocket.MessageText {
		return nil, fmt.Errorf("e2e: expected text message for SAS, got %v", typ)
	}
	msg := string(data)
	if !strings.HasPrefix(msg, "sas:") {
		return nil, fmt.Errorf("e2e: expected sas: prefix, got: %s", msg)
	}
	tag, err := base64.RawURLEncoding.DecodeString(msg[4:])
	if err != nil {
		return nil, fmt.Errorf("e2e: decode SAS tag: %w", err)
	}
	return tag, nil
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
