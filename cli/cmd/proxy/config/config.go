package config

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/kubernetes"
)

type Command struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	set *flag.Sets

	flagPodName string

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string
}

func (c *Command) init() {
	c.set = flag.NewSets()
	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    "pod",
		Aliases: []string{"p"},
		Target:  &c.flagPodName,
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

	c.help = c.set.Help()

	c.Init()
}

func (c *Command) Run(args []string) int {
	namespace := "default"

	c.once.Do(c.init)
	c.Log.ResetNamed("config")
	defer common.CloseWithError(c.BaseCommand)

	if err := c.set.Parse(args); err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	if c.flagPodName == "" {
		c.UI.Output(c.help)
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

	c.UI.Output("Pod: %s", c.flagPodName)
	kubeConfig, err := common.DefaultKubeConfigPath()
	if err != nil {
		c.UI.Output("Error retrieving default kubeconfig path:\n%v", err, terminal.WithErrorStyle())
		return 1
	}

	pf := common.PortForward{
		Namespace:   namespace,
		PodName:     c.flagPodName,
		RemotePort:  19000,
		UI:          c.UI,
		KubeClient:  c.kubernetes,
		KubeConfig:  kubeConfig,
		KubeContext: "kind-kind",
	}
	err = pf.Open()
	if err != nil {
		c.UI.Output("Error opening port-forward:\n%v", err, terminal.WithErrorStyle())
		return 1
	}
	defer pf.Close()

	response, err := http.Get(fmt.Sprintf("%s/config_dump", pf.Endpoint()))
	if err != nil {
		c.UI.Output("Error retrieving config dump:\n%v", err, terminal.WithErrorStyle())
		return 1
	}
	defer response.Body.Close()

	config, err := io.ReadAll(response.Body)
	if err != nil {
		c.UI.Output("Error reading config dump:\n%v", err, terminal.WithErrorStyle())
		return 1
	}

	c.UI.Output("%s", string(config))
	return 0
}

func (c *Command) Help() string {
	return c.help
}

func (c *Command) Synopsis() string {
	return ""
}
