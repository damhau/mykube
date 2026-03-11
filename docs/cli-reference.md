# CLI Reference

## Global flags

These flags apply to both `server` and `client` commands.

| Flag | Default | Description |
|------|---------|-------------|
| `--relay-url` | `https://mykube.onrender.com` | Relay server URL |
| `--proxy-ca` | — | Path to PEM CA cert for TLS-intercepting proxies |

## `mykube server`

Start the server side on a machine with cluster access. Reads the local kubeconfig, creates a session on the relay, and waits for a client to pair.

```bash
mykube server [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig file |
| `--relay-url` | `https://mykube.onrender.com` | Relay server URL |
| `--proxy-ca` | — | Path to PEM CA cert for TLS-intercepting proxies |

### Examples

```bash
# Use default kubeconfig and public relay
mykube server

# Use a specific kubeconfig
mykube server --kubeconfig /path/to/kubeconfig

# Use a custom relay
mykube server --relay-url https://relay.example.com
```

## `mykube client`

Connect to a server via pairing code. After key exchange and verification, spawns a subshell with kubectl configured.

```bash
mykube client [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--relay-url` | `https://mykube.onrender.com` | Relay server URL |
| `--proxy-ca` | — | Path to PEM CA cert for TLS-intercepting proxies |
| `--no-shell` | `false` | Don't spawn a subshell; print KUBECONFIG path and block until Ctrl+C |

### Examples

```bash
# Connect using the public relay
mykube client

# Use a custom relay
mykube client --relay-url https://relay.example.com

# Behind a corporate TLS-intercepting proxy
mykube client --proxy-ca /path/to/proxy-ca.pem

# Headless mode (no subshell)
mykube client --no-shell
```
