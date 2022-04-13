package validation

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-k8s/cli/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListConsulSecrets attempts to find secrets with the Consul label.
func ListConsulSecrets(ctx context.Context, client kubernetes.Interface, namespace string) (*v1.SecretList, error) {
	secrets, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", common.CLILabelKey, common.CLILabelValue),
	})

	return secrets, err
}

// PodExists attempts to check if a Kubernetes Pod exists with the given name
// in the given namespace.
func PodExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod == nil {
		return fmt.Errorf("pod %s/%s not found", namespace, name)
	}

	return nil
}
