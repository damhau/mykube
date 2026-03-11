# Self-Hosting the Relay

The relay is a lightweight FastAPI application that you can run anywhere.

## Requirements

- Python 3.11+
- [uv](https://docs.astral.sh/uv/) (recommended) or pip

## Running locally

```bash
cd relay
uv run python -m mykube_relay
```

The relay starts on `http://0.0.0.0:8000` by default.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_TTL_WAITING` | `120` | Seconds before an unpaired session expires |
| `SESSION_TTL_PAIRED` | `3600` | Seconds before a paired session expires |
| `MAX_PAIR_ATTEMPTS` | `5` | Max wrong pairing attempts before session is destroyed |
| `HOST` | `0.0.0.0` | Listen address |
| `PORT` | `8000` | Listen port |

## Docker

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY relay/ .

RUN pip install uv && uv sync

EXPOSE 8000
CMD ["uv", "run", "python", "-m", "mykube_relay"]
```

```bash
docker build -t mykube-relay .
docker run -p 8000:8000 mykube-relay
```

## Using your relay

Point both server and client to your relay URL:

```bash
mykube server --relay-url https://relay.example.com
mykube client --relay-url https://relay.example.com
```

## Deployment considerations

- **TLS**: Put the relay behind a reverse proxy (nginx, Caddy, etc.) with TLS. WebSocket connections require `wss://` in production.
- **Memory**: Sessions are stored in-memory. The relay uses minimal resources but will lose all sessions on restart.
- **Scaling**: The relay is single-instance by design (in-memory session store). For high availability, consider sticky sessions or a shared session backend.
- **Monitoring**: The relay logs session creation, pairing, and WebSocket connections to stdout.
