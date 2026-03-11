# Getting Started

## Install

Download a prebuilt binary from [GitHub Releases](https://github.com/damhau/mykube/releases/latest), or use one of the methods below.

=== "Linux (amd64)"

    ```bash
    curl -Lo mykube https://github.com/damhau/mykube/releases/latest/download/mykube-linux-amd64
    chmod +x mykube
    sudo mv mykube /usr/local/bin/
    ```

=== "Linux (arm64)"

    ```bash
    curl -Lo mykube https://github.com/damhau/mykube/releases/latest/download/mykube-linux-arm64
    chmod +x mykube
    sudo mv mykube /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -Lo mykube https://github.com/damhau/mykube/releases/latest/download/mykube-darwin-arm64
    chmod +x mykube
    sudo mv mykube /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -Lo mykube https://github.com/damhau/mykube/releases/latest/download/mykube-darwin-amd64
    chmod +x mykube
    sudo mv mykube /usr/local/bin/
    ```

=== "Windows"

    Download `mykube-windows-amd64.exe` from the [releases page](https://github.com/damhau/mykube/releases/latest) and add it to your `PATH`.

## First connection

### 1. Start the server

On a machine with access to your Kubernetes cluster:

```bash
mykube server
```

This reads your current kubeconfig, creates a session on the relay, and displays a pairing code:

```
Pairing code: 71124175
Waiting for client to connect...
```

### 2. Connect with the client

On your laptop:

```bash
mykube client
```

Enter the pairing code when prompted. After key exchange and SAS verification, a subshell opens:

```
Enter pairing code: 71124175
Verifying secure channel... OK
Connected to cluster "my-cluster"

[mykube:my-cluster] user@host:~$
```

### 3. Use kubectl

Inside the subshell, `kubectl` is configured to tunnel through mykube:

```bash
kubectl get nodes
kubectl get pods -A
kubectl logs deployment/my-app
kubectl exec -it pod/my-app -- /bin/sh
```

All commands work as if you were on the cluster network.

### 4. Disconnect

Type `exit` or press `Ctrl+D` to leave the subshell. The temporary kubeconfig is deleted and the tunnel is closed.

## Headless mode

If you don't want a subshell (e.g. for scripts or CI), use `--no-shell`:

```bash
mykube client --no-shell
```

This prints the `KUBECONFIG` path and blocks until `Ctrl+C`. You can use it in another terminal:

```bash
export KUBECONFIG=/tmp/mykube-my-cluster-abc123.yaml
kubectl get nodes
```
