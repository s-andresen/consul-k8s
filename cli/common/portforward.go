package common

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	Namespace  string
	PodName    string
	RemotePort int

	UI terminal.UI

	// TODO these three items tend to travel together, maybe abstract into a struct?
	KubeClient  kubernetes.Interface
	KubeConfig  string
	KubeContext string

	localPort int
	stopChan  chan struct{}
	readyChan chan struct{}
}

func (pf *PortForward) Open() error {
	// TODO replace this with "freeport"
	// Generate random port between 49152 and 65535 (dynamic ports)
	pf.localPort = rand.Intn(65535-49152) + 49152
	pf.stopChan = make(chan struct{}, 1)
	pf.readyChan = make(chan struct{}, 1)

	config, err := loadApiClientConfig(pf.KubeConfig, pf.KubeContext)

	postEndpoint := pf.KubeClient.CoreV1().RESTClient().Post()
	portForwardCreateURL := postEndpoint.
		Resource("pods").
		Namespace(pf.Namespace).
		Name(pf.PodName).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", portForwardCreateURL)

	ports := []string{fmt.Sprintf("%d:%d", pf.localPort, pf.RemotePort)}
	portforwarder, err := portforward.New(dialer, ports, pf.stopChan, pf.readyChan, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- portforwarder.ForwardPorts()
	}()

	select {
	case err := <-errChan:
		return err
	case <-portforwarder.Ready:
		return nil
	}
}

func (pf *PortForward) Endpoint() string {
	// Need to check if open first
	return fmt.Sprintf("http://localhost:%d", pf.localPort)
}

func (pf *PortForward) Close() {
	close(pf.stopChan)
}

func loadApiClientConfig(kubeConfig, kubeContext string) (*restclient.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}

	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig},
		&overrides)

	return config.ClientConfig()
}
