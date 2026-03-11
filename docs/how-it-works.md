# How It Works

## Architecture

mykube has three components:

- **Server** (cluster side) — a Go binary that reads the local kubeconfig and tunnels kube-apiserver access through the relay
- **Client** (laptop side) — a Go binary that pairs with the server, receives credentials, and provides a local kubectl endpoint
- **Relay** (public) — a Python FastAPI app that brokers WebSocket connections between server and client

```
  Laptop                        Relay (public)                     Cluster
 +----------+    WSS tunnel    +--------------+    WSS tunnel    +--------------+
 | kubectl   |--> 127.0.0.1 --| mykube-relay |--------------------| mykube server |--> kube-apiserver
 +----------+   (local proxy)  +--------------+                  +--------------+
```

The relay is **untrusted** — it only forwards opaque encrypted blobs between the two sides.

## Protocol flow

### 1. Session creation

The server calls `POST /api/sessions` on the relay. The relay returns a `session_id` and an 8-digit pairing code.

### 2. Pairing

The client calls `POST /api/pair` with the pairing code. The relay associates the client with the session.

### 3. WebSocket connection

Both sides open WebSocket connections to the relay:

- Server: `/ws/agent/{session_id}`
- Client: `/ws/client/{session_id}`

The relay forwards messages between the two WebSockets.

### 4. Key exchange

Both sides perform an **X25519 ECDH** key exchange:

1. Each side generates an ephemeral X25519 key pair
2. Public keys are exchanged over the WebSocket
3. A shared secret is derived via X25519
4. An AES-256-GCM session key is derived using **HKDF-SHA256** with the pairing code as salt

Binding the pairing code into key derivation means an attacker who intercepts the key exchange but doesn't know the code will derive a different key.

### 5. SAS verification

After key derivation, both sides compute a **Short Authentication String** (a verification tag) from the shared secret and pairing code. They exchange these tags over the encrypted channel. If the tags don't match, it means a man-in-the-middle has substituted public keys, and the connection is aborted.

This approach is inspired by [Magic Wormhole](https://magic-wormhole.readthedocs.io/en/latest/welcome.html).

### 6. Credential transfer

The server sends the cluster metadata (name, CA certificate, credentials) encrypted over the tunnel. The client decrypts it and writes a temporary kubeconfig.

### 7. TCP multiplexing

The client starts a local TCP listener (on `127.0.0.1`). When `kubectl` connects, the connection is multiplexed over the single encrypted WebSocket tunnel.

**Binary framing**: each WebSocket binary frame contains a 4-byte connection ID followed by the payload. Text frames are used for control messages (`new:{id}` to open a connection, `done:{id}` to close it).

This allows multiple concurrent kubectl operations (e.g. `kubectl exec`, `kubectl port-forward`, parallel API calls) over a single WebSocket.
