# Contributing

## Project structure

```
cli/                          Go CLI (Cobra)
  cmd/                        Commands (root, server, client)
  internal/
    e2e/                      X25519 + HKDF + AES-256-GCM + SAS verification
    handshake/                Cluster metadata exchange
    kubeconfig/               Kubeconfig loading and writing
    relay/                    Relay HTTP/WS client
    tunnel/                   TCP <-> WebSocket multiplexing

relay/                        Python relay server (FastAPI)
  src/mykube_relay/
    routes/                   HTTP API + WebSocket handlers
    session_store.py          In-memory session management
    rate_limit.py             Per-IP rate limiting
```

## Building from source

### CLI (Go)

Requires Go 1.25+.

```bash
cd cli
go build ./...
```

Build with a version tag:

```bash
go build -ldflags="-s -w -X github.com/damien/mykube/cli/cmd.Version=v0.7.0" -o mykube
```

For static binaries, set `CGO_ENABLED=0`:

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o mykube
```

### Relay (Python)

Requires Python 3.11+ and [uv](https://docs.astral.sh/uv/).

```bash
cd relay
uv sync
```

## Running tests

### CLI

```bash
cd cli
go test ./...
```

### Relay

```bash
cd relay
uv run pytest
```

Tests use `pytest-asyncio` with `asyncio_mode = "auto"`.

## Linting

### CLI

```bash
cd cli
go vet ./...
```

### Relay

```bash
cd relay
uv run ruff check .
```

## Running locally

### Relay

```bash
cd relay
uv run python -m mykube_relay
```

Starts on `http://0.0.0.0:8000`.

### CLI (against local relay)

```bash
# Terminal 1: server
cd cli
go run . server --relay-url http://localhost:8000

# Terminal 2: client
cd cli
go run . client --relay-url http://localhost:8000
```
