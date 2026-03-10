package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var relayURL = "https://mykube.onrender.com"
var proxyCA string

var rootCmd = &cobra.Command{
	Use:   "mykube",
	Short: "Tunnel kubectl through a WebSocket relay",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&relayURL, "relay-url", relayURL, "URL of the relay server")
	rootCmd.PersistentFlags().StringVar(&proxyCA, "proxy-ca", "", "path to PEM CA certificate to trust for TLS proxy")
}

func httpClientFromFlags() (*http.Client, error) {
	if proxyCA == "" {
		return http.DefaultClient, nil
	}
	caCert, err := os.ReadFile(proxyCA)
	if err != nil {
		return nil, fmt.Errorf("read proxy CA: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	pool.AppendCertsFromPEM(caCert)
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
			Proxy:           http.ProxyFromEnvironment,
		},
	}, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
