package kubeconfig

import (
	"fmt"
	"os"
	"text/template"
)

const kubeconfigTemplate = `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: {{ .CAData }}
    server: https://127.0.0.1:16443
  name: {{ .ClusterName }}
contexts:
- context:
    cluster: {{ .ClusterName }}
    user: mykube-user
  name: {{ .ClusterName }}
current-context: {{ .ClusterName }}
users:
- name: mykube-user
  user:
{{- if .Token }}
    token: {{ .Token }}
{{- end }}
{{- if .ClientCert }}
    client-certificate-data: {{ .ClientCert }}
{{- end }}
{{- if .ClientKey }}
    client-key-data: {{ .ClientKey }}
{{- end }}
`

type kubeconfigData struct {
	ClusterName string
	CAData      string
	Token       string
	ClientCert  string
	ClientKey   string
}

// WriteTempKubeconfig writes a temporary kubeconfig file pointing to the local proxy.
func WriteTempKubeconfig(clusterName, caData, token, clientCert, clientKey string) (string, error) {
	path := fmt.Sprintf("/tmp/mykube-%s.yaml", clusterName)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create temp kubeconfig: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("kubeconfig").Parse(kubeconfigTemplate)
	if err != nil {
		return "", fmt.Errorf("parse kubeconfig template: %w", err)
	}

	data := kubeconfigData{
		ClusterName: clusterName,
		CAData:      caData,
		Token:       token,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}

	return path, nil
}
