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

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string
}

func (c *Command) init() {
	c.set = flag.NewSets()
	f := c.set.NewSet("GlobalOptions")
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
	namespace := "default"

	c.once.Do(c.init)
	c.Log.ResetNamed("list")
	defer common.CloseWithError(c.BaseCommand)

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
	// TODO only get pods which have an Envoy config
	pods, err := c.kubernetes.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	kubeConfig, err := common.DefaultKubeConfigPath()
	if err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	// Open the port-forward to each of the pods
	envoyEndpoints := make(map[string]string)
	for _, pod := range pods.Items {
		pf := common.PortForward{
			Namespace:   namespace,
			PodName:     pod.Name,
			RemotePort:  19000,
			UI:          c.UI,
			KubeClient:  c.kubernetes,
			KubeConfig:  kubeConfig,
			KubeContext: "kind-kind",
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
			envoyStatuses[pod] = "Healthy"
		} else {
			envoyStatuses[pod] = "Unhealthy"
		}
	}

	table := map[string][]string{
		"Name":   {},
		"Type":   {},
		"Status": {},
	}
	for _, pod := range pods.Items {
		table["Name"] = append(table["Name"], pod.Name)
		// TODO actually read the proxy type
		table["Type"] = append(table["Type"], "Service")
		table["Status"] = append(table["Status"], envoyStatuses[pod.Name])
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
	tbl := terminal.NewTable("Name", "Type", "Status")
	tbl.Rows = [][]terminal.TableEntry{}

	for i := range table["Name"] {
		var statusColor string
		if table["Status"][i] == "Healthy" {
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
				Value: table["Status"][i],
				Color: statusColor,
			},
		}
		tbl.Rows = append(tbl.Rows, trow)
	}

	c.UI.Table(tbl)
}
