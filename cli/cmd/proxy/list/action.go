package list

type Action struct {
	namespace     string
	allNamespaces bool
	kubeconfig    string
	kubecontext   string
}

// Run lists Kubernetes Pods with sidecar proxies using the command configuration.
func (a *Action) Run() error {
	return nil
}
