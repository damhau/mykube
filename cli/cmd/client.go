package cmd

import (
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
	var code string
	if _, err := fmt.Scanln(&code); err != nil {
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

	// 7. Accept TCP connections and bridge through WebSocket (in background)
	go tunnel.ServeClient(ctx, wsConn, listener)

	// 8. Spawn subshell with KUBECONFIG set
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
	fmt.Fprintf(rcFile, "[ -f %s/.bashrc ] && source %s/.bashrc\n", home, home)
	fmt.Fprintf(rcFile, "export KUBECONFIG='%s'\n", kubeconfigPath)
	fmt.Fprintf(rcFile, "export PS1='[mykube:%s] \\u@\\h:\\w\\$ '\n", hs.ClusterName)
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

	// 9. Cleanup happens via deferred calls
	fmt.Fprintf(os.Stderr, "Disconnected.\n")
	return nil
}
