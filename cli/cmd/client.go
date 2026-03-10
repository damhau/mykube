package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/damien/mykube/cli/internal/handshake"
	"github.com/damien/mykube/cli/internal/kubeconfig"
	"github.com/damien/mykube/cli/internal/relay"
	"github.com/damien/mykube/cli/internal/tunnel"
	"github.com/spf13/cobra"
	"nhooyr.io/websocket"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Connect to a cluster through the relay (operator laptop side)",
	RunE:  runClient,
}

func init() {
	rootCmd.AddCommand(clientCmd)
}

func runClient(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 1. Prompt for pairing code
	fmt.Fprint(os.Stderr, "Enter pairing code: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read pairing code: %w", err)
	}
	code = strings.TrimSpace(code)

	// 2. Pair via relay
	rc := &relay.RelayClient{BaseURL: relayURL}
	sessionID, err := rc.PairSession(code)
	if err != nil {
		return fmt.Errorf("pair session: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Paired! Session: %s\n", sessionID)

	// 3. Connect WebSocket as client
	wsConn, err := rc.ConnectClient(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("connect client ws: %w", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "client shutting down")

	// 4. Read handshake from agent
	hs, err := handshake.Receive(ctx, wsConn)
	if err != nil {
		return fmt.Errorf("receive handshake: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Connected to cluster: %s\n", hs.ClusterName)

	// 5. Write temp kubeconfig
	kubeconfigPath, err := kubeconfig.WriteTempKubeconfig(hs.ClusterName, hs.CAData, hs.Token, hs.ClientCert, hs.ClientKey)
	if err != nil {
		return fmt.Errorf("write temp kubeconfig: %w", err)
	}
	defer os.Remove(kubeconfigPath)
	fmt.Fprintf(os.Stderr, "Wrote temp kubeconfig: %s\n", kubeconfigPath)

	// 6. Start TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:16443")
	if err != nil {
		return fmt.Errorf("listen on 127.0.0.1:16443: %w", err)
	}
	defer listener.Close()
	fmt.Fprintf(os.Stderr, "Listening on 127.0.0.1:16443\n")

	// 7. Accept TCP connections and bridge through WebSocket
	go func() {
		for {
			tcpConn, err := listener.Accept()
			if err != nil {
				return
			}
			go tunnel.Bridge(ctx, wsConn, tcpConn)
		}
	}()

	// 8. Spawn subshell with KUBECONFIG set
	fmt.Fprintf(os.Stderr, "Spawning shell with KUBECONFIG=%s\n", kubeconfigPath)
	fmt.Fprintf(os.Stderr, "Use kubectl as usual. Type 'exit' to disconnect.\n\n")

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellCmd := exec.CommandContext(ctx, shell)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)

	if err := shellCmd.Run(); err != nil {
		// Shell exited (possibly with non-zero), that's OK
		fmt.Fprintf(os.Stderr, "\nShell exited.\n")
	}

	// 9. Cleanup happens via deferred calls
	fmt.Fprintf(os.Stderr, "Disconnected.\n")
	return nil
}
