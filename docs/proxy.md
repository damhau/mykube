# Proxy Support

mykube fully supports HTTP/HTTPS proxies, including corporate TLS-intercepting proxies.

## Standard proxy variables

mykube respects the standard environment variables:

| Variable | Description |
|----------|-------------|
| `HTTP_PROXY` | Proxy URL for HTTP connections |
| `HTTPS_PROXY` | Proxy URL for HTTPS connections |
| `NO_PROXY` | Comma-separated list of hosts to bypass the proxy |

All HTTP and WebSocket traffic to the relay will be routed through the proxy automatically.

```bash
export HTTPS_PROXY=http://proxy.corp.example.com:8080
mykube client
```

## TLS-intercepting proxies

Corporate environments often use TLS-intercepting proxies (e.g. Squid, Zscaler, BlueCoat) that decrypt and re-encrypt HTTPS traffic. These proxies present their own CA certificate, which mykube won't trust by default.

Use `--proxy-ca` to provide the proxy's CA certificate:

```bash
mykube client --proxy-ca /path/to/proxy-ca.pem
mykube server --proxy-ca /path/to/proxy-ca.pem
```

The CA file should be a PEM-encoded certificate. This flag applies to both the HTTP API calls and the WebSocket connection to the relay.

!!! note
    Even with a TLS-intercepting proxy, your kubectl traffic remains **end-to-end encrypted** between client and server. The proxy can only see the outer WebSocket connection to the relay, not the encrypted tunnel contents.

## Finding your proxy CA

=== "Zscaler"

    ```bash
    # Usually available at:
    /usr/local/share/ca-certificates/zscaler-root-ca.pem
    # Or export from your browser's certificate store
    ```

=== "Corporate IT"

    Ask your IT department for the proxy CA certificate in PEM format. It's often distributed via system policy or available on an internal portal.

=== "Squid"

    The CA certificate is configured in `squid.conf` under `ssl_bump`. The admin can provide the public CA cert.
