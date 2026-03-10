  CRITICAL (4)                                                                                                                                                                                                                                                                                                                             
                                                                                                                                                                                                                                                                                                                                           
  ┌─────┬─────────────────────────────┬──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐                                                                                                         
  │  #  │          Component          │                                                                                          Issue                                                                                           │                                                                                                         
  ├─────┼─────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤                                                                                                         
  │ 1   │ CLI handshake/protocol.go   │ K8s credentials exposed to relay — Bearer tokens, client certs, and private keys are sent as plaintext JSON over the WebSocket. The relay can read everything. No end-to-end encryption. │
  ├─────┼─────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 2   │ CLI kubeconfig/writer.go:16 │ insecure-skip-tls-verify: true — The generated kubeconfig skips TLS verification. CA data is already present so this is unnecessary and enables MITM.                                    │
  ├─────┼─────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 3   │ Relay routes/api.py         │ No authentication on any endpoint — All HTTP and WebSocket endpoints are completely unauthenticated. Anyone on the internet can create sessions, attempt pairing, connect WebSockets.    │
  ├─────┼─────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 4   │ Relay session_store.py:27   │ Weak pairing code — 6-digit numeric code (1M combinations) using random (non-crypto PRNG) instead of secrets. Brute-forceable, especially with no rate limiting.                         │
  └─────┴─────────────────────────────┴──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  HIGH (6)

  ┌─────┬────────────────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  #  │           Component            │                                                                                             Issue                                                                                              │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 5   │ CLI kubeconfig/writer.go:52    │ World-readable temp file — os.Create uses mode 0666 (before umask). Credentials written to predictable /tmp/mykube-*.yaml path readable by other local users.                                  │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 6   │ CLI kubeconfig/writer.go:49    │ Path traversal — clusterName from remote peer only has : replaced. A name like ../../etc/cron.d/evil writes outside /tmp/.                                                                     │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 7   │ CLI cmd/client.go:107          │ Shell injection via cluster name — ClusterName from remote peer is interpolated into a bash RC file (PS1 assignment). Malicious names can escape single quotes and execute arbitrary commands. │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 8   │ CLI kubeconfig/writer.go:10-36 │ YAML injection — text/template does no escaping. A ClusterName with " can break out of YAML strings and inject exec credential plugins (arbitrary code execution).                             │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 9   │ Relay routes/api.py            │ No rate limiting — No throttling on session creation or pairing attempts. Enables brute-force of pairing codes and memory exhaustion via mass session creation.                                │
  ├─────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 10  │ Relay routes/ws.py             │ No WebSocket Origin validation — Any website can open a cross-origin WebSocket to the relay if it knows the session_id.                                                                        │
  └─────┴────────────────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  MEDIUM (7)

  ┌─────┬───────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  #  │         Component         │                                                               Issue                                                               │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 11  │ Relay                     │ No WS message size limit — Attacker can send multi-GB frames to exhaust memory.                                                   │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 12  │ Relay session_store.py    │ Unbounded session count — No cap on total sessions, enabling memory DoS.                                                          │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 13  │ Relay routes/ws.py        │ Session hijacking — No check preventing a second agent/client from overwriting the existing WebSocket connection.                 │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 14  │ CLI tunnel/bridge.go      │ ConnID spoofing — No integrity check on binary frames. A compromised relay can inject/forge frames with arbitrary connection IDs. │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 15  │ CLI tunnel/bridge.go      │ Unbounded WS reads — No explicit SetReadLimit() on the WebSocket connection.                                                      │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 16  │ CLI handshake/protocol.go │ No handshake validation — No length or content checks on received fields.                                                         │
  ├─────┼───────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 17  │ Relay                     │ No CORS middleware configured.                                                                                                    │
  └─────┴───────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  LOW (4)

  ┌─────┬──────────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
  │  #  │        Component         │                                                       Issue                                                        │
  ├─────┼──────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 18  │ CLI kubeconfig/writer.go │ Symlink attack — Predictable /tmp/ path enables symlink-based file overwrite. Use os.CreateTemp or O_EXCL.         │
  ├─────┼──────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 19  │ Relay routes/ws.py:26    │ Busy-wait loops — 50ms polling for peer connection; runs indefinitely if peer never connects. CPU exhaustion risk. │
  ├─────┼──────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 20  │ Relay session_store.py:4 │ random instead of secrets — Non-crypto PRNG for security-sensitive code generation.                                │
  ├─────┼──────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
  │ 21  │ Relay routes/ws.py:90    │ Broad except Exception: pass — Silently swallows all errors during WebSocket close.                                │
  └─────┴──────────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  ---
  Top priorities to address

  1. End-to-end encryption — Add a key exchange (e.g., X25519 + AES-GCM) between agent and client so the relay cannot read credentials. This eliminates issues #1, #6, and #14.
  2. Remove insecure-skip-tls-verify (#2) — The CA data is already in the kubeconfig.
  3. Sanitize ClusterName — Strip to alphanumeric/dash to fix path traversal (#6), shell injection (#7), and YAML injection (#8). Or use encoding/json + gopkg.in/yaml.v3 for safe serialization.
  4. Fix temp file permissions — Use os.OpenFile with mode 0600 and O_EXCL (#5, #18).
  5. Use secrets module for pairing codes and increase code length/alphabet (#4, #20).
  6. Add rate limiting on the relay (#9) — per-IP limits on session creation and pairing.
  7. Enforce single agent/client per session (#13) — reject second connections.


# Status

  ┌─────┬───────────────────────────────────────────┬───────────────────────────────────────────────────────┐                                                                                                                                                                                                                              
  │  #  │                   Issue                   │                        Status                         │                                                                                                                                                                                                                              
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 1   │ K8s credentials exposed to relay          │ Fixed — E2E encryption (X25519 + AES-256-GCM)         │                                                                                                                                                                                                                              
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 2   │ insecure-skip-tls-verify                  │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 3   │ No authentication on endpoints            │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 4   │ Weak pairing code / non-crypto PRNG       │ Fixed — secrets module, 8-digit codes                 │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 5   │ World-readable temp file                  │ Fixed — os.CreateTemp (mode 0600, random path)        │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 6   │ Path traversal via ClusterName            │ Fixed — SanitizeClusterName strips to [a-zA-Z0-9._-]  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 7   │ Shell injection via ClusterName           │ Fixed — same sanitization applied in RC file          │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 8   │ YAML injection via ClusterName            │ Fixed — sanitized name used in template               │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 9   │ No rate limiting                          │ Fixed — 20 req/min per IP on /api/*                   │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 10  │ No WebSocket Origin validation            │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 11  │ No WS message size limit                  │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 12  │ Unbounded session count                   │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 13  │ Session hijacking (duplicate connections) │ Fixed — reject if agent/client already connected      │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 14  │ ConnID spoofing                           │ Fixed — E2E encryption means relay can't forge frames │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 15  │ Unbounded WS reads                        │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 16  │ No handshake validation                   │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 17  │ No CORS middleware                        │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 18  │ Symlink attack on temp file               │ Fixed — os.CreateTemp with O_EXCL                     │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 19  │ Busy-wait polling loops                   │ Open                                                  │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 20  │ random instead of secrets                 │ Fixed — uses secrets.choice                           │
  ├─────┼───────────────────────────────────────────┼───────────────────────────────────────────────────────┤
  │ 21  │ Broad except Exception: pass              │ Open                                                  │
  └─────┴───────────────────────────────────────────┴───────────────────────────────────────────────────────┘