# mykube

Access a remote Kubernetes cluster from anywhere through a WebSocket relay — no VPN, no public IP required.

The **server** (cluster side) tunnels the kube-apiserver through the relay.
The **client** (laptop side) pairs with a code, gets a local kubeconfig, and uses `kubectl` normally.

```
  Laptop                        Relay (public)                     Cluster
 ┌──────────┐    WSS tunnel    ┌──────────────┐    WSS tunnel    ┌──────────────┐
 │ kubectl   ├───► 127.0.0.1 ──┤  mykube-relay ├────────────────┤ mykube server │──► kube-apiserver
 └──────────┘   (local proxy)  └──────────────┘                  └──────────────┘
```

All traffic between client and server is **end-to-end encrypted** (X25519 + AES-256-GCM). The relay only sees opaque binary blobs.

## Quick start

### Server side (on a machine with cluster access)

```bash
mykube server --kubeconfig ~/.kube/config
```

A pairing code is displayed (e.g. `A3K9FM72`).

### Client side (your laptop)

```bash
mykube client
# Enter pairing code: A3K9FM72
```

A subshell opens with `KUBECONFIG` set. Use `kubectl` as usual:

```bash
[mykube:my-cluster] user@host:~$ kubectl get nodes
```

Type `exit` to disconnect.

## Install

Download a binary from [Releases](https://github.com/damien/mykube/releases), or build from source:

```bash
curl -o mykube https://github.com/damhau/mykube/releases/download/v0.3.0/mykube-linux-amd64
chmod +x mykube
mv mykube /usr/local/bin
```

## How it works

1. `mykube server` loads the kubeconfig credentials, creates a session on the relay, and displays a pairing code
2. `mykube client` sends the pairing code to the relay to join the session
3. Both sides perform an **X25519 ECDH key exchange** and derive an AES-256-GCM session key
4. The server sends cluster metadata (name, CA, credentials) **encrypted** over the tunnel
5. The client writes a temporary kubeconfig pointing to a local TCP listener and spawns a shell
6. `kubectl` connects to `127.0.0.1` which is multiplexed through the encrypted WebSocket tunnel to the kube-apiserver

Multiple concurrent TCP connections (e.g. `kubectl exec`, `port-forward`) are multiplexed over a single WebSocket using binary framing with connection IDs.

## CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--relay-url` | `https://mykube.onrender.com` | Relay server URL |
| `--proxy-ca` | — | Path to PEM CA cert for TLS-intercepting proxies |
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Kubeconfig path (server only) |

## Relay server

The relay is a lightweight FastAPI app that brokers WebSocket connections between agents and clients. It never sees decrypted traffic.

```bash
cd relay
uv run python -m mykube_relay
```

Runs on `http://0.0.0.0:8000` by default. Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_TTL_WAITING` | `120` | Seconds before an unpaired session expires |
| `SESSION_TTL_PAIRED` | `3600` | Seconds before a paired session expires |
| `MAX_PAIR_ATTEMPTS` | `5` | Max wrong pairing attempts before session is destroyed |
| `HOST` | `0.0.0.0` | Listen address |
| `PORT` | `8000` | Listen port |

### Running tests

```bash
cd relay
uv run pytest
```

## Security

- **E2E encryption**: X25519 key exchange + AES-256-GCM. The relay cannot read tunnel traffic.
- **Pairing codes**: 8-character alphanumeric codes from a cryptographic PRNG (`secrets`), with max 5 attempts per session.
- **Rate limiting**: API endpoints are rate-limited per IP (20 req/min).
- **Single connection enforcement**: Only one agent and one client can connect per session.
- **Temp files**: Kubeconfig is written with mode 0600 to a random path and deleted on exit.
- **Input sanitization**: Cluster names are stripped to `[a-zA-Z0-9._-]` before use in file paths and shell commands.

## Project structure

```
cli/                          Go CLI (Cobra)
  cmd/                        Commands (root, server, client)
  internal/
    e2e/                      X25519 + AES-256-GCM encryption
    handshake/                Cluster metadata exchange
    kubeconfig/               Kubeconfig loading and writing
    relay/                    Relay HTTP/WS client
    tunnel/                   TCP ↔ WebSocket multiplexing

relay/                        Python relay server (FastAPI)
  src/mykube_relay/
    routes/                   HTTP API + WebSocket handlers
    session_store.py          In-memory session management
    rate_limit.py             Per-IP rate limiting
```

## License

MIT
