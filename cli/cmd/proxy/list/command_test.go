package list

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// getInitializedCommand sets up a command struct for tests.
func getInitializedCommand(t *testing.T) (*ListCommand, *bytes.Buffer) {
	t.Helper()
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "test",
		Level:  hclog.Debug,
		Output: os.Stdout,
	})

	buffer := new(bytes.Buffer)
	baseCommand := &common.BaseCommand{
		Log: log,
		UI:  terminal.NewUI(context.TODO(), buffer),
	}

	c := &ListCommand{
		BaseCommand: baseCommand,
	}
	c.init()
	return c, buffer
}

func TestArgumentParsing(t *testing.T) {
	cases := map[string]struct {
		args []string
		out  int
	}{
		"No args": {
			args: []string{},
			out:  0,
		},
		"Nonexistent flag passed, -foo bar": {
			args: []string{"-foo", "bar"},
			out:  1,
		},
		"Invalid argument passed, -namespace YOLO": {
			args: []string{"-namespace", "YOLO"},
			out:  1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := getInitializedCommand(t)
			c.kubernetes = fake.NewSimpleClientset()
			out := c.Run(tc.args)
			require.Equal(t, tc.out, out)
		})
	}
}

func TestFetchPods(t *testing.T) {
	cases := map[string]struct {
		namespace    string
		pods         []v1.Pod
		expectedPods int
	}{
		"No pods": {
			namespace:    "default",
			pods:         []v1.Pod{},
			expectedPods: 0,
		},
		"Gateway pods": {
			namespace: "default",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"component": "ingress-gateway",
							"chart":     "consul-helm",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mesh-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"component": "mesh-gateway",
							"chart":     "consul-helm",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "terminating-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"component": "terminating-gateway",
							"chart":     "consul-helm",
						},
					},
				},
			},
			expectedPods: 3,
		},
		"API Gateway Pods": {
			namespace: "default",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "api-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"api-gateway.consul.hashicorp.com/managed": "true",
						},
					},
				},
			},
			expectedPods: 1,
		},
		"Sidecar Pods": {
			namespace: "default",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "default",
						Labels: map[string]string{
							"consul.hashicorp.com/connect-inject-status": "injected",
						},
					},
				},
			},
			expectedPods: 1,
		},
		"All kinds of Pods": {
			namespace: "default",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "default",
						Labels: map[string]string{
							"consul.hashicorp.com/connect-inject-status": "injected",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mesh-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"component": "mesh-gateway",
							"chart":     "consul-helm",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "api-gateway",
						Namespace: "default",
						Labels: map[string]string{
							"api-gateway.consul.hashicorp.com/managed": "true",
						},
					},
				},
			},
			expectedPods: 3,
		},
		"Pods in multiple namespaces": {
			namespace: "",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "api-gateway",
						Namespace: "consul",
						Labels: map[string]string{
							"api-gateway.consul.hashicorp.com/managed": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "default",
						Labels: map[string]string{
							"consul.hashicorp.com/connect-inject-status": "injected",
						},
					},
				},
			},
			expectedPods: 2,
		},
		"Pods which should not be fetched": {
			namespace: "default",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dont-fetch",
						Namespace: "default",
						Labels:    map[string]string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "default",
						Labels: map[string]string{
							"consul.hashicorp.com/connect-inject-status": "injected",
						},
					},
				},
			},
			expectedPods: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset(&v1.PodList{Items: tc.pods})

			c, _ := getInitializedCommand(t)
			c.kubernetes = client
			c.flagNamespace = tc.namespace
			pods, err := c.fetchPods()

			require.NoError(t, err)
			require.Equal(t, tc.expectedPods, len(pods))
		})
	}
}

func TestEndToEnd(t *testing.T) {
	// These regular expressions must be present in the output.
	expected := []string{
		"Namespace.*Name.*Type",
		"consul.*mesh-gateway.*Mesh Gateway",
		"consul.*terminating-gateway.*Terminating Gateway",
		"default.*ingress-gateway.*Ingress Gateway",
		"consul.*api-gateway.*Sidecar",
		"default.*pod1.*Sidecar",
	}

	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-gateway",
				Namespace: "default",
				Labels: map[string]string{
					"component": "ingress-gateway",
					"chart":     "consul-helm",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mesh-gateway",
				Namespace: "consul",
				Labels: map[string]string{
					"component": "mesh-gateway",
					"chart":     "consul-helm",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "terminating-gateway",
				Namespace: "consul",
				Labels: map[string]string{
					"component": "terminating-gateway",
					"chart":     "consul-helm",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-gateway",
				Namespace: "consul",
				Labels: map[string]string{
					"api-gateway.consul.hashicorp.com/managed": "true",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dont-fetch",
				Namespace: "default",
				Labels:    map[string]string{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels: map[string]string{
					"consul.hashicorp.com/connect-inject-status": "injected",
				},
			},
		},
	}
	client := fake.NewSimpleClientset(&v1.PodList{Items: pods})

	c, buf := getInitializedCommand(t)
	c.kubernetes = client

	out := c.Run([]string{"-A"})
	require.Equal(t, 0, out)

	actual := buf.String()

	for _, expression := range expected {
		require.Regexp(t, expression, actual)
	}
}
