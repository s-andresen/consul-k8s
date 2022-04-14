package envoy

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
