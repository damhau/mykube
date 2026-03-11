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

	"github.com/damien/mykube/cli/internal/e2e"
	"github.com/damien/mykube/cli/internal/handshake"
	"github.com/damien/mykube/cli/internal/kubeconfig"
	"github.com/damien/mykube/cli/internal/relay"
	"github.com/damien/mykube/cli/internal/tunnel"
	"github.com/spf13/cobra"
	"nhooyr.io/websocket"
)

var noShell bool

var clientCmd = &cobra.Command{
	Use:   "client [pairing-code]",
	Short: "Connect to a cluster through the relay (operator laptop side)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runClient,
}

func init() {
	clientCmd.Flags().BoolVar(&noShell, "no-shell", false, "don't spawn a subshell; just print KUBECONFIG and block until interrupted")
	rootCmd.AddCommand(clientCmd)
}

func runClient(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 1. Get pairing code from arg or prompt
	var code string
	if len(args) > 0 {
		code = strings.TrimSpace(args[0])
	} else {
		fmt.Fprint(os.Stderr, "Enter pairing code: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read pairing code: %w", err)
		}
		code = strings.TrimSpace(line)
	}

	// 2. Pair via relay
	httpClient, err := httpClientFromFlags()
	if err != nil {
		return err
	}
	rc := &relay.RelayClient{BaseURL: relayURL, HTTPClient: httpClient}
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

	// 4. E2E key exchange (client is responder, pairing code bound to key derivation)
	encConn, err := e2e.KeyExchange(ctx, wsConn, false, code)
	if err != nil {
		return fmt.Errorf("e2e key exchange: %w", err)
	}
	fmt.Fprintf(os.Stderr, "E2E encryption established.\n")

	// 5. Read handshake from agent (encrypted)
	hs, err := handshake.Receive(ctx, encConn)
	if err != nil {
		return fmt.Errorf("receive handshake: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Connected to cluster: %s\n", hs.ClusterName)

	// 5. Start TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on 127.0.0.1: %w", err)
	}
	defer listener.Close()
	localAddr := listener.Addr().String()
	fmt.Fprintf(os.Stderr, "Listening on %s\n", localAddr)

	// 6. Write temp kubeconfig
	kubeconfigPath, err := kubeconfig.WriteTempKubeconfig(hs.ClusterName, localAddr, hs.CAData, hs.Token, hs.ClientCert, hs.ClientKey)
	if err != nil {
		return fmt.Errorf("write temp kubeconfig: %w", err)
	}
	defer os.Remove(kubeconfigPath)
	fmt.Fprintf(os.Stderr, "Wrote temp kubeconfig: %s\n", kubeconfigPath)

	// 8. Accept TCP connections and bridge through WebSocket (in background)
	go tunnel.ServeClient(ctx, encConn, listener)

	if noShell {
		fmt.Fprintf(os.Stderr, "\n\033[32m●\033[0m Connected to cluster %s\n", hs.ClusterName)
		fmt.Fprintf(os.Stderr, "  KUBECONFIG=%s\n", kubeconfigPath)
		fmt.Fprintf(os.Stderr, "  Listening on %s\n", localAddr)
		fmt.Fprintf(os.Stderr, "  Press Ctrl+C to disconnect.\n\n")
		<-ctx.Done()
	} else {
		// Spawn subshell with KUBECONFIG set
		fmt.Fprintf(os.Stderr, "Spawning shell with KUBECONFIG=%s\n", kubeconfigPath)
		fmt.Fprintf(os.Stderr, "Use kubectl as usual. Type 'exit' to disconnect.\n\n")

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}

		// Create temp rcfile that sources user's bashrc then overrides PS1
		rcFile, err := os.CreateTemp("", "mykube-rc-*")
		if err != nil {
			return fmt.Errorf("create temp rcfile: %w", err)
		}
		defer os.Remove(rcFile.Name())
		home, _ := os.UserHomeDir()
		safeName := kubeconfig.SanitizeClusterName(hs.ClusterName)
		fmt.Fprintf(rcFile, "[ -f %s/.bashrc ] && source %s/.bashrc\n", home, home)
		fmt.Fprintf(rcFile, "export KUBECONFIG='%s'\n", kubeconfigPath)
		fmt.Fprintf(rcFile, "export PS1='[mykube:%s] \\u@\\h:\\w\\$ '\n", safeName)
		rcFile.Close()

		shellCmd := exec.Command(shell, "--rcfile", rcFile.Name())
		shellCmd.Stdin = os.Stdin
		shellCmd.Stdout = os.Stdout
		shellCmd.Stderr = os.Stderr
		shellCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)

		if err := shellCmd.Run(); err != nil {
			// Shell exited (possibly with non-zero), that's OK
			fmt.Fprintf(os.Stderr, "\nShell exited.\n")
		}
	}

	// 9. Cleanup happens via deferred calls
	fmt.Fprintf(os.Stderr, "Disconnected.\n")
	return nil
}
