package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var relayURL = "https://mykube.onrender.com"

var rootCmd = &cobra.Command{
	Use:   "mykube",
	Short: "Tunnel kubectl through a WebSocket relay",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&relayURL, "relay-url", relayURL, "URL of the relay server")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
