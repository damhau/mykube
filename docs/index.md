# mykube

**Access a remote Kubernetes cluster from anywhere — no VPN, no public IP required.**

mykube tunnels kubectl traffic through a WebSocket relay with end-to-end encryption. The **server** runs on the cluster side, the **client** runs on your laptop, and a simple pairing code connects them.

```
  Laptop                        Relay (public)                     Cluster
 +----------+    WSS tunnel    +--------------+    WSS tunnel    +--------------+
 | kubectl   |--> 127.0.0.1 --| mykube-relay |--------------------| mykube server |--> kube-apiserver
 +----------+   (local proxy)  +--------------+                  +--------------+
```

## Why mykube?

- **Zero infrastructure** — no VPN servers, no bastion hosts, no firewall rules
- **End-to-end encrypted** — X25519 + HKDF + AES-256-GCM; the relay never sees your credentials
- **MITM protection** — pairing code is cryptographically bound into key derivation with SAS verification
- **Just works** — single binary, one command on each side, `kubectl` works as usual
- **Corporate-friendly** — full HTTP/HTTPS proxy support including TLS-intercepting proxies

## Quick start

### Server side (on a machine with cluster access)

```bash
mykube server
```

A pairing code is displayed (e.g. `71124175`).

### Client side (your laptop)

```bash
mykube client
# Enter pairing code: 71124175
```

A subshell opens with `KUBECONFIG` set. Use `kubectl` as usual:

```bash
[mykube:my-cluster] user@host:~$ kubectl get nodes
```

Type `exit` to disconnect.

[Get started](getting-started.md){ .md-button .md-button--primary }
[How it works](how-it-works.md){ .md-button }
