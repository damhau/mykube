package kubeconfig

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const kubeconfigTemplate = `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: "{{ .CAData }}"
    server: "https://{{ .ServerAddr }}"
    insecure-skip-tls-verify: true
  name: "{{ .ClusterName }}"
contexts:
- context:
    cluster: "{{ .ClusterName }}"
    user: mykube-user
  name: "{{ .ClusterName }}"
current-context: "{{ .ClusterName }}"
users:
- name: mykube-user
  user:
{{- if .Token }}
    token: "{{ .Token }}"
{{- end }}
{{- if .ClientCert }}
    client-certificate-data: "{{ .ClientCert }}"
{{- end }}
{{- if .ClientKey }}
    client-key-data: "{{ .ClientKey }}"
{{- end }}
`

type kubeconfigData struct {
	ClusterName string
	ServerAddr  string
	CAData      string
	Token       string
	ClientCert  string
	ClientKey   string
}

// SanitizeClusterName strips a cluster name to safe characters only
// (alphanumeric, dash, underscore, dot). Returns "cluster" if the result is empty.
func SanitizeClusterName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		s = "cluster"
	}
	return s
}

// WriteTempKubeconfig writes a temporary kubeconfig file (mode 0600) pointing
// to the local proxy. Returns the file path.
func WriteTempKubeconfig(clusterName, serverAddr, caData, token, clientCert, clientKey string) (string, error) {
	safeName := SanitizeClusterName(clusterName)

	f, err := os.CreateTemp("", fmt.Sprintf("mykube-%s-*.yaml", safeName))
	if err != nil {
		return "", fmt.Errorf("create temp kubeconfig: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("kubeconfig").Parse(kubeconfigTemplate)
	if err != nil {
		return "", fmt.Errorf("parse kubeconfig template: %w", err)
	}

	data := kubeconfigData{
		ClusterName: safeName,
		ServerAddr:  serverAddr,
		CAData:      caData,
		Token:       token,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}

	return f.Name(), nil
}
