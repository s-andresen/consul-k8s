package envoy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"k8s.io/client-go/tools/clientcmd"
)

const envoyPort = 19000

// FetchConfig attempts to fetch the Envoy config from a Pod.
func FetchConfig(ctx context.Context, kubeconfig string, namespace, name string) ([]byte, error) {
	localPort := 19000
	readyCh := make(chan struct{})
	stopCh := make(chan struct{})

	// Port forward localhost to the Envoy port.
	if err := portForward(kubeconfig, localPort, readyCh, stopCh); err != nil {
		return nil, err
	}

	// Request config from the local endpoint.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/config_dump", localPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// portForward sets up port forwarding from the given localPort to the given podPort.
func portForward(kubeconfig, namespace, name string, localPort int, readyCh, stopCh chan struct{}) error {
	var wg sync.WaitGroup
	wg.Add(1)

	kcfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, name)
	hostIP := "localhost"

	go func() {

	}()

	wg.Wait()

	return nil
}
