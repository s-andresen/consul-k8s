package list

import "k8s.io/client-go/kubernetes"

// Action uses its configuration to
type Action struct {
	NamespaceFilter string
	Kubeconfig      string
	Kubecontext     string

	kubernetes kubernetes.Interface
}

type PodConfig struct {
	Namespace string
	ProxyType string
}

// Run tries to search the K8s cluster for all Pods which run Envoy proxies and
// returns a table mapping Pod names PodConfig.
func (a *Action) Run() (map[string]PodConfig, error) {

	return nil, nil
}
