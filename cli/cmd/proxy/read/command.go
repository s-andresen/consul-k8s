package read

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/kubernetes"
)

var (
	kubecontext = "teckert@hashicorp.com@thomas-eks-test.us-east-2.eksctl.io"
)

func defaultKubeConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.kube/config", nil
}

type Command struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	set *flag.Sets

	flagPodName   string
	flagNamespace string

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string
}

func (c *Command) init() {
	kubeconfig, err := defaultKubeConfigPath()
	if err != nil {
		panic(err)
	}

	c.set = flag.NewSets()
	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    "pod",
		Aliases: []string{"p"},
		Target:  &c.flagPodName,
	})
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
		Default: kubeconfig,
		Usage:   "Set the path to kubeconfig file.",
	})
	f.StringVar(&flag.StringVar{
		Name:    "context",
		Target:  &c.flagKubeContext,
		Default: kubecontext,
		Usage:   "Set the Kubernetes context to use.",
	})

	c.help = c.set.Help()

	c.Init()
}

func (c *Command) Run(args []string) int {
	c.once.Do(c.init)
	c.Log.ResetNamed("read")
	defer common.CloseWithError(c.BaseCommand)

	if err := c.set.Parse(args); err != nil {
		c.UI.Output(err.Error())
		return 1
	}

	if c.flagPodName == "" {
		c.UI.Output(c.help)
		return 1
	}

	settings := helmCLI.New()
	if c.flagKubeConfig != "" {
		settings.KubeConfig = c.flagKubeConfig
	}
	if c.flagKubeContext != "" {
		settings.KubeContext = c.flagKubeContext
	}
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

	pf := common.PortForward{
		Namespace:   c.flagNamespace,
		PodName:     c.flagPodName,
		RemotePort:  19000,
		KubeClient:  c.kubernetes,
		KubeConfig:  settings.KubeConfig,
		KubeContext: settings.KubeContext,
	}
	if err := pf.Open(); err != nil {
		c.UI.Output("Error opening port forward:\n%v", err, terminal.WithErrorStyle())
		return 1
	}
	defer pf.Close()

	endpoint, err := pf.Endpoint()
	if err != nil {
		c.UI.Output("Error getting endpoint:\n%v", err, terminal.WithErrorStyle())
		return 1
	}

	response, err := http.Get(fmt.Sprintf("%s/config_dump", endpoint))
	if err != nil {
		c.UI.Output("Error getting config dump:\n%v", err, terminal.WithErrorStyle())
		return 1
	}
	defer response.Body.Close()

	config, err := io.ReadAll(response.Body)
	if err != nil {
		c.UI.Output("Error reading config dump:\n%v", err, terminal.WithErrorStyle())
		return 1
	}

	c.UI.Output(fmt.Sprintf("%s Proxy Configuration", c.flagPodName), terminal.WithHeaderStyle())
	err = c.Print(config)
	if err != nil {
		c.UI.Output("Error printing config:\n%v", err, terminal.WithErrorStyle())
		return 1
	}

	return 0
}

func (c *Command) Synopsis() string {
	return ""
}

func (c *Command) Help() string {
	return ""
}

func (c *Command) Print(config []byte) error {
	var envCfg map[string]interface{}

	json.Unmarshal(config, &envCfg)

	for _, cfg := range envCfg["configs"].([]interface{}) {
		a := cfg.(map[string]interface{})
		cfgType := a["@type"].(string)

		switch cfgType {
		case "type.googleapis.com/envoy.admin.v3.ClustersConfigDump":
			err := c.PrintClusters(a)
			if err != nil {
				return err
			}
		case "type.googleapis.com/envoy.admin.v3.ListenersConfigDump":
			err := c.PrintListeners(a)
			if err != nil {
				return err
			}
		case "type.googleapis.com/envoy.admin.v3.SecretsConfigDump":
			err := c.PrintSecrets(a)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Command) PrintClusters(clusters map[string]interface{}) error {
	c.UI.Output("Clusters:", terminal.WithHeaderStyle())

	tbl := terminal.NewTable("Name", "FQDN", "Endpoints", "Type", "Last Updated")
	if clusters["clusters"] == nil {
		c.UI.Table(tbl)
		return nil
	}

	for _, cluster := range clusters["static_clusters"].([]interface{}) {
		a := cluster.(map[string]interface{})
		b := a["cluster"].(map[string]interface{})
		fqdn := b["name"].(string)
		var name string
		if strings.Contains(fqdn, ".") {
			name = strings.Split(fqdn, ".")[0]
		} else {
			name = fqdn
		}
		load_assignment := b["load_assignment"].(map[string]interface{})
		eps := load_assignment["endpoints"].([]interface{})

		endpoints := make([]string, 0)
		for _, ep := range eps {
			lb_endpoints := ep.(map[string]interface{})["lb_endpoints"].([]interface{})
			for _, lb_ep := range lb_endpoints {
				e := lb_ep.(map[string]interface{})
				f := e["endpoint"].(map[string]interface{})
				address := f["address"].(map[string]interface{})
				sockaddr := address["socket_address"].(map[string]interface{})
				addr := sockaddr["address"].(string)
				portv := int(sockaddr["port_value"].(float64))
				endpoints = append(endpoints, fmt.Sprintf("%s:%d", addr, portv))
			}
		}

		typ := b["type"].(string)
		lupdated := a["last_updated"].(string)
		trow := []terminal.TableEntry{
			{
				Value: name,
			},
			{
				Value: fqdn,
			},
			{
				Value: strings.Join(endpoints, ", "),
			},
			{
				Value: typ,
			},
			{
				Value: lupdated,
			},
		}
		tbl.Rows = append(tbl.Rows, trow)
	}
	for _, cluster := range clusters["dynamic_active_clusters"].([]interface{}) {
		a := cluster.(map[string]interface{})
		b := a["cluster"].(map[string]interface{})
		fqdn := b["name"].(string)
		var name string
		if strings.Contains(fqdn, ".") {
			name = strings.Split(fqdn, ".")[0]
		} else {
			name = fqdn
		}

		endpts := ""
		if b["load_assignment"] != nil {
			load_assignment := b["load_assignment"].(map[string]interface{})
			eps := load_assignment["endpoints"].([]interface{})
			endpoints := make([]string, 0)
			for _, ep := range eps {
				lb_endpoints := ep.(map[string]interface{})["lb_endpoints"].([]interface{})
				for _, lb_ep := range lb_endpoints {
					e := lb_ep.(map[string]interface{})
					f := e["endpoint"].(map[string]interface{})
					address := f["address"].(map[string]interface{})
					sockaddr := address["socket_address"].(map[string]interface{})
					addr := sockaddr["address"].(string)
					portv := int(sockaddr["port_value"].(float64))
					endpoints = append(endpoints, fmt.Sprintf("%s:%d", addr, portv))
				}
			}
			endpts = strings.Join(endpoints, ", ")
		}

		typ := b["type"].(string)
		lupdated := a["last_updated"].(string)
		trow := []terminal.TableEntry{
			{
				Value: name,
			},
			{
				Value: fqdn,
			},
			{
				Value: endpts,
			},
			{
				Value: typ,
			},
			{
				Value: lupdated,
			},
		}
		tbl.Rows = append(tbl.Rows, trow)
	}
	c.UI.Table(tbl)

	return nil
}

func (c *Command) PrintListeners(listeners map[string]interface{}) error {
	c.UI.Output("Listeners:", terminal.WithHeaderStyle())

	tbl := terminal.NewTable("Name", "Address:Port", "Direction", "Filter Chain Match", "Destination Cluster", "Last Updated")
	c.UI.Table(tbl)
	return nil
}

func (c *Command) PrintSecrets(secrets map[string]interface{}) error {
	c.UI.Output("Secrets:", terminal.WithHeaderStyle())

	fmt.Printf("%+v\n", secrets)

	tbl := terminal.NewTable("Name", "Type", "Status", "Valid", "Valid from", "Valid to")
	if secrets["secrets"] == nil {
		c.UI.Table(tbl)
		return nil
	}

	for _, secret := range secrets["secrets"].([]interface{}) {
		secret_ := secret.(map[string]interface{})
		name := secret_["name"].(string)
		typ := ""
		status := ""
		valid := ""
		validfrom := ""
		validto := ""

		trow := []terminal.TableEntry{
			{
				Value: name,
			},
			{
				Value: typ,
			},
			{
				Value: status,
			},
			{
				Value: valid,
			},
			{
				Value: validfrom,
			},
			{
				Value: validto,
			},
		}
		tbl.Rows = append(tbl.Rows, trow)
	}

	c.UI.Table(tbl)
	return nil
}
