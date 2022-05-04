package list

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	// Get all of the pods in the namespace
	// TODO get the Consul gateways too
	pods, err := c.kubernetes.CoreV1().Pods(c.flagNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "consul.hashicorp.com/connect-inject-status=injected",
	})
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	c.flagKubeConfig, err = common.DefaultKubeConfigPath()
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}
	c.flagKubeContext = common.Context

	// Open the port-forward to each of the pods
	envoyEndpoints := make(map[string]string)
	for _, pod := range pods.Items {
		pf := common.PortForward{
			Namespace:   c.flagNamespace,
			PodName:     pod.Name,
			RemotePort:  19000,
			UI:          c.UI,
			KubeClient:  c.kubernetes,
			KubeConfig:  c.flagKubeConfig,
			KubeContext: c.flagKubeContext,
		}

		err := pf.Open()
		defer pf.Close()
		if err != nil {
			c.UI.Output(err.Error())
			return 1
		}

		envoyEndpoints[pod.Name] = pf.Endpoint()
	}

	envoyStatuses := make(map[string]string, len(pods.Items))
	for pod, endpoint := range envoyEndpoints {
		resp, err := http.Get(fmt.Sprintf("%s/ready", endpoint))
		if err != nil {
			c.UI.Output(err.Error())
			return 1
		}

		if resp.StatusCode == 200 {
			envoyStatuses[pod] = "True"
		} else {
			envoyStatuses[pod] = "False"
		}
	}

	table := map[string][]string{
		"Name":  {},
		"Type":  {},
		"Ready": {},
	}
	for _, pod := range pods.Items {
		table["Name"] = append(table["Name"], pod.Name)
		// TODO actually read the proxy type
		table["Type"] = append(table["Type"], "Service")
		table["Ready"] = append(table["Ready"], envoyStatuses[pod.Name])
	}

	c.PrintTable(table)

	return 0
}

func (c *Command) Synopsis() string {
	return ""
}

func (c *Command) Help() string {
	return ""
}

func (c *Command) PrintTable(table map[string][]string) {
	// TODO would be cool to generalize this.
	tbl := terminal.NewTable("Name", "Type", "Ready")
	tbl.Rows = [][]terminal.TableEntry{}

	for i := range table["Name"] {
		var statusColor string
		if table["Ready"][i] == "True" {
			statusColor = terminal.Green
		} else {
			statusColor = terminal.Red
		}

		trow := []terminal.TableEntry{
			{
				Value: table["Name"][i],
			},
			{
				Value: table["Type"][i],
			},
			{
				Value: table["Ready"][i],
				Color: statusColor,
			},
		}
		tbl.Rows = append(tbl.Rows, trow)
	}

	c.UI.Table(tbl)
}
