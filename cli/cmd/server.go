package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/damien/mykube/cli/internal/e2e"
	"github.com/damien/mykube/cli/internal/handshake"
	"github.com/damien/mykube/cli/internal/kubeconfig"
	"github.com/damien/mykube/cli/internal/relay"
	"github.com/damien/mykube/cli/internal/tunnel"
	"github.com/spf13/cobra"
	"nhooyr.io/websocket"
)

var kubeconfigPath string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run cluster-side agent that tunnels kube-apiserver through the relay",
	RunE:  runServer,
}

func init() {
	defaultKubeconfig := os.Getenv("KUBECONFIG")
	if defaultKubeconfig == "" {
		home, _ := os.UserHomeDir()
		defaultKubeconfig = home + "/.kube/config"
	}
	serverCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", defaultKubeconfig, "path to kubeconfig file")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load kubeconfig once
	clusterName, serverURL, caData, token, clientCert, clientKey, err := kubeconfig.LoadCurrentContext(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Loaded kubeconfig for cluster %q (server: %s)\n", clusterName, serverURL)

	serverHost, err := extractHost(serverURL)
	if err != nil {
		return fmt.Errorf("parse server url: %w", err)
	}

	httpClient, err := httpClientFromFlags()
	if err != nil {
		return err
	}
	rc := &relay.RelayClient{BaseURL: relayURL, HTTPClient: httpClient}

	hs := &handshake.Handshake{
		ClusterName: clusterName,
		CAData:      caData,
		Token:       token,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
	}

	for {
		if err := serveOnce(ctx, rc, hs, serverHost); err != nil {
			if ctx.Err() != nil {
				break
			}
			fmt.Fprintf(os.Stderr, "Error: %v\nRetrying in 5s...\n", err)
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
			}
		}
		if ctx.Err() != nil {
			break
		}
	}

	fmt.Fprintf(os.Stderr, "Server stopped.\n")
	return nil
}

func serveOnce(ctx context.Context, rc *relay.RelayClient, hs *handshake.Handshake, serverHost string) error {
	sessionID, code, err := rc.CreateSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n========================================\n")
	fmt.Fprintf(os.Stderr, "  Pairing code:  %s\n", code)
	fmt.Fprintf(os.Stderr, "========================================\n\n")
	fmt.Fprintf(os.Stderr, "Waiting for client to connect (session: %s)...\n", sessionID)

	wsConn, err := rc.ConnectAgent(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("connect agent ws: %w", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	// Wait for paired signal
	_, _, err = wsConn.Read(ctx)
	if err != nil {
		return fmt.Errorf("waiting for client: %w", err)
	}

	// E2E key exchange (pairing code bound to key derivation)
	encConn, err := e2e.KeyExchange(ctx, wsConn, true, code)
	if err != nil {
		return fmt.Errorf("e2e key exchange: %w", err)
	}

	// Send handshake
	if err := hs.Send(ctx, encConn); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\033[32m●\033[0m Client connected — tunneling to %s\n", serverHost)
	tunnel.ServeAgent(ctx, encConn, serverHost)
	fmt.Fprintf(os.Stderr, "\033[31m●\033[0m Client disconnected.\n")

	return nil
}

func extractHost(serverURL string) (string, error) {
	// Remove scheme
	host := serverURL
	for _, prefix := range []string{"https://", "http://"} {
		if len(host) > len(prefix) && host[:len(prefix)] == prefix {
			host = host[len(prefix):]
			break
		}
	}
	// Add default port if missing
	_, _, err := net.SplitHostPort(host)
	if err != nil {
		// Likely missing port
		host = host + ":443"
	}
	return host, nil
}
