package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	// 1. Load kubeconfig
	clusterName, serverURL, caData, token, clientCert, clientKey, err := kubeconfig.LoadCurrentContext(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Loaded kubeconfig for cluster %q (server: %s)\n", clusterName, serverURL)

	// 2. Create session via relay
	rc := &relay.RelayClient{BaseURL: relayURL}
	sessionID, code, err := rc.CreateSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n========================================\n")
	fmt.Fprintf(os.Stderr, "  Pairing code:  %s\n", code)
	fmt.Fprintf(os.Stderr, "========================================\n\n")
	fmt.Fprintf(os.Stderr, "Waiting for client to connect (session: %s)...\n", sessionID)

	// 3. Connect WebSocket as agent
	wsConn, err := rc.ConnectAgent(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("connect agent ws: %w", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "server shutting down")

	// 4. Wait for client signal (first message from relay)
	_, _, err = wsConn.Read(ctx)
	if err != nil {
		return fmt.Errorf("waiting for client signal: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Client connected! Sending handshake...\n")

	// 5. Send handshake
	hs := &handshake.Handshake{
		ClusterName: clusterName,
		CAData:      caData,
		Token:       token,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
	}
	if err := hs.Send(ctx, wsConn); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	// 6. Parse server URL for TCP dial
	serverHost, err := extractHost(serverURL)
	if err != nil {
		return fmt.Errorf("parse server url: %w", err)
	}

	// 7. Dial kube-apiserver and bridge
	fmt.Fprintf(os.Stderr, "Dialing kube-apiserver at %s...\n", serverHost)
	tcpConn, err := net.Dial("tcp", serverHost)
	if err != nil {
		return fmt.Errorf("dial kube-apiserver: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Bridging traffic...\n")
	tunnel.Bridge(ctx, wsConn, tcpConn)

	fmt.Fprintf(os.Stderr, "Connection closed.\n")
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
