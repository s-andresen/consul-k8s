package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPortForward_Open(t *testing.T) {
	cases := map[string]struct {
		DoCreatePod bool
		Namespace   string
		PodName     string
		RemotePort  int
	}{
		"Open port forward to pod that exists": {
			DoCreatePod: true,
			Namespace:   "consul",
			PodName:     "pod",
			RemotePort:  8080,
		},
		"Open port forward to pod that doesn't exist": {
			DoCreatePod: false,
			Namespace:   "consul",
			PodName:     "pod",
			RemotePort:  8080,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.DoCreatePod {
				// Create the Pod
			}

			pf := &PortForward{
				Namespace:  tc.Namespace,
				PodName:    tc.PodName,
				RemotePort: tc.RemotePort,
			}

			err := pf.Open()
			defer pf.Close()

			if tc.DoCreatePod {
				// Delete the Pod
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPortForward_Endpoint_BeforeOpen(t *testing.T) {
	pf := &PortForward{}

	// Endpoint must return an error if called before Open()
	_, err := pf.Endpoint()
	require.Error(t, err)
}

func TestPortForward_Endpoint_AfterOpen(t *testing.T) {
	pf := &PortForward{}

	err := pf.Open()
	require.NoError(t, err)

	// Endpoint must return a valid endpoint if called after Open()
	endpoint, err := pf.Endpoint()
	require.NoError(t, err)
	require.NotEqual(t, endpoint, "")
}

func TestPortForward_Endpoint_AfterClose(t *testing.T) {

}

func TestPortForward_Close(t *testing.T) {
}

func TestPortForward_allocateLocalPort(t *testing.T) {
}

func TestPortForward_loadApiClientConfig(t *testing.T) {
}
