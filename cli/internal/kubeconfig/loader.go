package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

// LoadCurrentContext loads the current kubeconfig context and extracts connection details.
func LoadCurrentContext(kubeconfigPath string) (clusterName, serverURL, caData, token, clientCert, clientKey string, err error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{Precedence: filepath.SplitList(kubeconfigPath)}
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("load kubeconfig: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return "", "", "", "", "", "", fmt.Errorf("no current context set in kubeconfig")
	}

	ctxObj, ok := rawConfig.Contexts[currentContext]
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("context %q not found", currentContext)
	}

	clusterName = ctxObj.Cluster
	cluster, ok := rawConfig.Clusters[clusterName]
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("cluster %q not found", clusterName)
	}

	serverURL = cluster.Server
	if len(cluster.CertificateAuthorityData) > 0 {
		caData = base64.StdEncoding.EncodeToString(cluster.CertificateAuthorityData)
	}

	authInfo, ok := rawConfig.AuthInfos[ctxObj.AuthInfo]
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("auth info %q not found", ctxObj.AuthInfo)
	}

	token = authInfo.Token
	if len(authInfo.ClientCertificateData) > 0 {
		clientCert = base64.StdEncoding.EncodeToString(authInfo.ClientCertificateData)
	}
	if len(authInfo.ClientKeyData) > 0 {
		clientKey = base64.StdEncoding.EncodeToString(authInfo.ClientKeyData)
	}

	return clusterName, serverURL, caData, token, clientCert, clientKey, nil
}
