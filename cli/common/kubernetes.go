package common

import "os"

// GetKubecontext attempts to get the current Kubecontext for the user first
// from the KUBECONTEXT environment variable, then from Kubeconfig as a fallback.
func GetKubecontext() (string, error) {
	// TODO implement this to fetch from env first, then from kubeconfig
	kctx := "teckert@hashicorp.com@thomas-eks-test.us-east-2.eksctl.io"

	return kctx, nil
}

func GetKubeconfig() (string, error) {
	// TODO implement this to fetch this correctly
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.kube/config", nil
}
