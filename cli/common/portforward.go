package common

import (
	"fmt"

	"github.com/hashicorp/consul-k8s/cli/common/terminal"
)

// Work from https://github.com/gruntwork-io/terratest/blob/master/modules/k8s/tunnel.go

// PortForward allows for Kubernetes Pods to be port-forwarded to localhost.
type PortForward struct {
	// LocalPort is the port on localhost to forward to.
	LocalPort int

	// RemotePort is the port on the Kubernetes Pod to forward to.
	RemotePort int

	// UI is the terminal.UI object to use for output.
	UI terminal.UI

	readyChan chan struct{}
	stopChan  chan struct{}
}

// Open opens a port-forwarding tunnel to the Kubernetes Pod.
func (pf *PortForward) Open() error {
	return nil
}

// Close closes the port-forwarding tunnel.
func (pf *PortForward) Close() {
	close(pf.stopChan)
}

func (pf *PortForward) Endpoint() string {
	return fmt.Sprintf("localhost:%d", pf.LocalPort)
}
