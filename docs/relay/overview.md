# Relay Overview

The relay is a lightweight Python service that brokers WebSocket connections between mykube servers and clients. It is **untrusted by design** — it never sees decrypted traffic.

## What the relay does

1. **Session management** — creates sessions with pairing codes, matches servers to clients
2. **WebSocket forwarding** — forwards binary and text frames between paired server/client connections
3. **Rate limiting** — throttles API requests to prevent abuse (20 req/min per IP)
4. **Connection enforcement** — ensures only one server and one client per session

## What the relay does NOT do

- Decrypt or inspect traffic (all data is encrypted end-to-end)
- Store credentials or sensitive data
- Authenticate users (sessions are matched by pairing codes)
- Persist any data (everything is in-memory)

## Trust model

The relay is designed to be completely untrusted:

- **Confidentiality**: All traffic is encrypted with AES-256-GCM. The relay can't read it.
- **Integrity**: Frames are authenticated. The relay can't modify or forge data.
- **Availability**: The relay can drop or delay messages, but it can't compromise security.

The worst a malicious relay can do is refuse to forward messages (denial of service).

## Public relay

A free public relay is available at `mykube.onrender.com` and is used by default. No setup required.

For organizations that prefer to control the relay infrastructure, see [Self-Hosting](self-hosting.md).

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/sessions` | Create a new session (returns session ID + pairing code) |
| `POST` | `/api/pair` | Pair a client with a session using the pairing code |
| `GET` | `/ws/agent/{id}` | WebSocket endpoint for the server (agent) side |
| `GET` | `/ws/client/{id}` | WebSocket endpoint for the client side |
