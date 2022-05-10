package list

import (
	"context"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	kubeconfig  = ""
	kubecontext = ""
)

type Command struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	set *flag.Sets

	flagNamespace string

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string
}

func (c *Command) init() {
	c.set = flag.NewSets()

	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    "namespace",
		Target:  &c.flagNamespace,
		Default: "default",
		Usage:   "The namespace to list proxies in.",
		Aliases: []string{"n"},
	})

	f = c.set.NewSet("GlobalOptions")
	f.StringVar(&flag.StringVar{
		Name:    "kubeconfig",
		Aliases: []string{"c"},
		Target:  &c.flagKubeConfig,
		Default: "",
		Usage:   "Set the path to kubeconfig file.",
	})
	f.StringVar(&flag.StringVar{
		Name:    "context",
		Target:  &c.flagKubeContext,
		Default: "",
		Usage:   "Set the Kubernetes context to use.",
	})

	c.Init()
}

func (c *Command) Run(args []string) int {
	c.once.Do(c.init)
	c.Log.ResetNamed("list")
	defer common.CloseWithError(c.BaseCommand)
	if err := c.set.Parse(args); err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	// TODO I shouldn't use the Helm CLI here...
	settings := helmCLI.New()

	// Set up the kubernetes client to use for non Helm SDK calls to the Kubernetes API
	// The Helm SDK will use settings.RESTClientGetter for its calls as well, so this will
	// use a consistent method to target the right cluster for both Helm SDK and non Helm SDK calls.
	if c.kubernetes == nil {
		restConfig, err := settings.RESTClientGetter().ToRESTConfig()
		if err != nil {
			c.UI.Output("Error retrieving Kubernetes authentication:\n%v", err, terminal.WithErrorStyle())
			return 1
		}
		c.kubernetes, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			c.UI.Output("Error initializing Kubernetes client:\n%v", err, terminal.WithErrorStyle())
			return 1
		}
	}

	gatewaypods, err := c.kubernetes.CoreV1().Pods(c.flagNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component in (ingress-gateway, api-gateway, mesh-gateway, terminating-gateway)",
	})
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	sidecarpods, err := c.kubernetes.CoreV1().Pods(c.flagNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "consul.hashicorp.com/connect-inject-status=injected",
	})
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	pods := append(gatewaypods.Items, sidecarpods.Items...)

	tbl := terminal.NewTable("Name", "Type")
	for _, pod := range pods {
		component := pod.Labels["component"]

		var podType string
		if component == "ingress-gateway" {
			podType = "Ingress Gateway"
		} else if component == "api-gateway" {
			podType = "API Gateway"
		} else if component == "mesh-gateway" {
			podType = "Mesh Gateway"
		} else if component == "terminating-gateway" {
			podType = "Terminating Gateway"
		} else {
			podType = "Sidecar"
		}

		trow := []terminal.TableEntry{
			{
				Value: pod.Name,
			},
			{
				Value: podType,
			},
		}

		tbl.Rows = append(tbl.Rows, trow)
	}
	c.UI.Table(tbl)

	return 0
}

func (c *Command) Synopsis() string {
	return ""
}

func (c *Command) Help() string {
	return ""
}
