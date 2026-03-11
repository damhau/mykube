# Security

## Encryption

All traffic between the client and server is **end-to-end encrypted**. The relay only sees opaque binary blobs.

| Layer | Algorithm |
|-------|-----------|
| Key exchange | X25519 ECDH |
| Key derivation | HKDF-SHA256 (pairing code as salt) |
| Encryption | AES-256-GCM |

## MITM protection

The pairing code is cryptographically bound into key derivation via the HKDF salt. After key exchange, both sides derive and exchange a **SAS (Short Authentication String)** verification tag over the encrypted channel. If a man-in-the-middle substitutes public keys, the derived keys will differ and the SAS tags won't match — the connection is aborted.

This is inspired by [Magic Wormhole](https://magic-wormhole.readthedocs.io/en/latest/welcome.html)'s approach of binding a short code into the cryptographic handshake.

## Pairing codes

- 8-digit numeric codes generated with Python's `secrets` module (cryptographic PRNG)
- Maximum 5 attempts per session — after 5 wrong codes, the session is destroyed
- Sessions expire after 120 seconds if unpaired

## Relay trust model

The relay is **untrusted by design**:

- It never sees decrypted traffic
- It cannot forge or inject frames (all data is authenticated by AES-256-GCM)
- It only forwards WebSocket messages between paired sessions
- A single agent and single client connection is enforced per session

You can use the public relay at `mykube.onrender.com` or [self-host your own](relay/self-hosting.md).

## Additional protections

| Protection | Details |
|-----------|---------|
| Rate limiting | 20 requests/min per IP on API endpoints |
| Session hijacking | Duplicate agent/client connections are rejected |
| Temp file permissions | Kubeconfig written with mode `0600` to a random path, deleted on exit |
| Input sanitization | Cluster names stripped to `[a-zA-Z0-9._-]` before use in paths and shell commands |
| WebSocket message size | 16 MiB limit on both CLI and relay |

## Known issues tracker

The table below tracks all identified security issues and their status.

### Fixed

| # | Issue | Fix |
|---|-------|-----|
| 1 | K8s credentials exposed to relay | E2E encryption (X25519 + HKDF + AES-256-GCM + SAS) |
| 4 | Weak pairing code / non-crypto PRNG | `secrets` module, 8-digit codes |
| 5 | World-readable temp file | `os.CreateTemp` (mode 0600, random path) |
| 6 | Path traversal via ClusterName | `SanitizeClusterName` strips to `[a-zA-Z0-9._-]` |
| 7 | Shell injection via ClusterName | Same sanitization applied in RC file |
| 8 | YAML injection via ClusterName | Sanitized name used in template |
| 9 | No rate limiting | 20 req/min per IP on `/api/*` |
| 11 | No WS message size limit | 16 MiB limit on CLI and relay |
| 13 | Session hijacking (duplicate connections) | Reject if agent/client already connected |
| 14 | ConnID spoofing | E2E encryption means relay can't forge frames |
| 15 | Unbounded WS reads | `SetReadLimit(16 MiB)` on both sides |
| 18 | Symlink attack on temp file | `os.CreateTemp` with `O_EXCL` |
| 20 | `random` instead of `secrets` | Uses `secrets.choice` |

### Open

| # | Issue | Severity |
|---|-------|----------|
| 2 | `insecure-skip-tls-verify` in generated kubeconfig | Critical |
| 3 | No authentication on relay endpoints | Critical |
| 10 | No WebSocket Origin validation | High |
| 12 | Unbounded session count | Medium |
| 16 | No handshake field validation | Medium |
| 17 | No CORS middleware | Medium |
| 19 | Busy-wait polling loops | Low |
| 21 | Broad `except Exception: pass` | Low |
