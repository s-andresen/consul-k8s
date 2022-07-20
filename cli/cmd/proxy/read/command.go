package read

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/strings/slices"
)

// adminPort is the port where the Envoy admin API is exposed.
const adminPort int = 19000

const (
	Table = "table"
	JSON  = "json"
	Raw   = "raw"
)

type ReadCommand struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	set *flag.Sets

	// Command Flags
	flagNamespace string
	flagPodName   string
	flagOutput    string

	// Output Filtering Opts
	flagClusters  bool
	flagListeners bool
	flagRoutes    bool
	flagEndpoints bool
	flagSecrets   bool
	flagFQDN      string
	flagAddress   string
	flagPort      int

	// Global Flags
	flagKubeConfig  string
	flagKubeContext string

	fetchConfig func(context.Context, common.PortForwarder) (*EnvoyConfig, error)

	restConfig *rest.Config

	once sync.Once
	help string
}

func (c *ReadCommand) init() {
	if c.fetchConfig == nil {
		c.fetchConfig = FetchConfig
	}

	c.set = flag.NewSets()
	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    "namespace",
		Target:  &c.flagNamespace,
		Usage:   "The namespace where the target Pod can be found.",
		Aliases: []string{"n"},
	})
	f.StringVar(&flag.StringVar{
		Name:    "output",
		Target:  &c.flagOutput,
		Usage:   "Output the Envoy configuration as 'table', 'json', or 'raw'.",
		Default: Table,
		Aliases: []string{"o"},
	})

	f = c.set.NewSet("Output Filtering Options")
	f.BoolVar(&flag.BoolVar{
		Name:   "clusters",
		Target: &c.flagClusters,
		Usage:  "Filter output to only show clusters.",
	})
	f.BoolVar(&flag.BoolVar{
		Name:   "listeners",
		Target: &c.flagListeners,
		Usage:  "Filter output to only show listeners.",
	})
	f.BoolVar(&flag.BoolVar{
		Name:   "routes",
		Target: &c.flagRoutes,
		Usage:  "Filter output to only show routes.",
	})
	f.BoolVar(&flag.BoolVar{
		Name:   "endpoints",
		Target: &c.flagEndpoints,
		Usage:  "Filter output to only show endpoints.",
	})
	f.BoolVar(&flag.BoolVar{
		Name:   "secrets",
		Target: &c.flagSecrets,
		Usage:  "Filter output to only show secrets.",
	})
	f.StringVar(&flag.StringVar{
		Name:   "fqdn",
		Target: &c.flagFQDN,
		Usage:  "Filter cluster output to only clusters with a fully qualified domain name which contains the given value.",
	})
	f.StringVar(&flag.StringVar{
		Name:   "address",
		Target: &c.flagAddress,
		Usage:  "Filter clusters, endpoints, and listeners output to only those with endpoint addresses which contain the given value.",
	})
	f.IntVar(&flag.IntVar{
		Name:    "port",
		Target:  &c.flagPort,
		Usage:   "Filter endpoints output to only endpoints with the given port number.",
		Default: -1,
	})

	f = c.set.NewSet("GlobalOptions")
	f.StringVar(&flag.StringVar{
		Name:    "kubeconfig",
		Aliases: []string{"c"},
		Target:  &c.flagKubeConfig,
		Usage:   "Set the path to kubeconfig file.",
	})
	f.StringVar(&flag.StringVar{
		Name:   "context",
		Target: &c.flagKubeContext,
		Usage:  "Set the Kubernetes context to use.",
	})

	c.help = c.set.Help()
}

func (c *ReadCommand) Run(args []string) int {
	c.once.Do(c.init)
	c.Log.ResetNamed("read")
	defer common.CloseWithError(c.BaseCommand)

	if err := c.parseFlags(args); err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		c.UI.Output("\n" + c.Help())
		return 1
	}

	if err := c.validateFlags(); err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		c.UI.Output("\n" + c.Help())
		return 1
	}

	if err := c.initKubernetes(); err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		return 1
	}

	pf := common.PortForward{
		Namespace:  c.flagNamespace,
		PodName:    c.flagPodName,
		RemotePort: adminPort,
		KubeClient: c.kubernetes,
		RestConfig: c.restConfig,
	}

	config, err := c.fetchConfig(c.Ctx, &pf)
	if err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		return 1
	}

	err = c.outputConfig(config)
	if err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		return 1
	}
	return 0
}

func (c *ReadCommand) Help() string {
	c.once.Do(c.init)
	return fmt.Sprintf("%s\n\nUsage: consul-k8s proxy read <pod-name> [flags]\n\n%s", c.Synopsis(), c.help)
}

func (c *ReadCommand) Synopsis() string {
	return "Inspect the Envoy configuration for a given Pod."
}

func (c *ReadCommand) parseFlags(args []string) error {
	// Separate positional arguments from keyed arguments.
	positional := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		positional = append(positional, arg)
	}
	keyed := args[len(positional):]

	if len(positional) != 1 {
		return fmt.Errorf("Exactly one positional argument is required: <pod-name>")
	}
	c.flagPodName = positional[0]

	if err := c.set.Parse(keyed); err != nil {
		return err
	}

	return nil
}

func (c *ReadCommand) validateFlags() error {
	if errs := validation.ValidateNamespaceName(c.flagNamespace, false); c.flagNamespace != "" && len(errs) > 0 {
		return fmt.Errorf("invalid namespace name passed for -namespace/-n: %v", strings.Join(errs, "; "))
	}
	if outputs := []string{Table, JSON, Raw}; !slices.Contains(outputs, c.flagOutput) {
		return fmt.Errorf("-output must be one of %s.", strings.Join(outputs, ", "))
	}
	return nil
}

// areTablesFiltered returns true if a table filtering flag was passed in.
func (c *ReadCommand) areTablesFiltered() bool {
	return c.flagClusters || c.flagEndpoints || c.flagListeners || c.flagRoutes || c.flagSecrets
}

func (c *ReadCommand) initKubernetes() (err error) {
	settings := helmCLI.New()

	if c.flagKubeConfig == "" {
		settings.KubeConfig = c.flagKubeConfig
	}

	if c.flagKubeContext == "" {
		settings.KubeContext = c.flagKubeContext
	}

	if c.restConfig == nil {
		if c.restConfig, err = settings.RESTClientGetter().ToRESTConfig(); err != nil {
			return fmt.Errorf("error creating Kubernetes REST config %v", err)
		}
	}

	if c.kubernetes == nil {
		if c.kubernetes, err = kubernetes.NewForConfig(c.restConfig); err != nil {
			return fmt.Errorf("error creating Kubernetes client %v", err)
		}
	}

	if c.flagNamespace == "" {
		c.flagNamespace = settings.Namespace()
	}

	return nil
}

func (c *ReadCommand) outputConfig(config *EnvoyConfig) error {
	switch c.flagOutput {
	case Table:
		c.outputAsTables(config)
	case JSON:
		return c.outputAsJSON(config)
	case Raw:
		c.UI.Output(string(config.rawCfg))
	}

	return nil
}

func (c *ReadCommand) outputAsJSON(config *EnvoyConfig) error {
	cfg := make(map[string]interface{})
	if !c.areTablesFiltered() || c.flagClusters {
		cfg["clusters"] = FilterClusters(config.Clusters, c.flagFQDN, c.flagAddress, c.flagPort)
	}
	if !c.areTablesFiltered() || c.flagEndpoints {
		cfg["endpoints"] = FilterEndpoints(config.Endpoints, c.flagAddress, c.flagPort)
	}
	if !c.areTablesFiltered() || c.flagListeners {
		cfg["listeners"] = FilterListeners(config.Listeners, c.flagAddress, c.flagPort)
	}
	if !c.areTablesFiltered() || c.flagRoutes {
		cfg["routes"] = config.Routes
	}
	if !c.areTablesFiltered() || c.flagSecrets {
		cfg["secrets"] = config.Secrets
	}

	out, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}

	c.UI.Output(string(out))
	return nil
}

func (c *ReadCommand) outputAsTables(config *EnvoyConfig) {
	c.UI.Output(fmt.Sprintf("Envoy configuration for %s in namespace %s:", c.flagPodName, c.flagNamespace))
	if c.flagFQDN != "" || c.flagAddress != "" || c.flagPort != -1 {
		c.UI.Output("Filters applied", terminal.WithHeaderStyle())

		if c.flagFQDN != "" {
			c.UI.Output(fmt.Sprintf("Fully qualified domain names must contain `%s`", c.flagFQDN), terminal.WithInfoStyle())
		}
		if c.flagAddress != "" {
			c.UI.Output(fmt.Sprintf("Endpoint addresses must contain `%s`", c.flagAddress), terminal.WithInfoStyle())
		}
		if c.flagPort != -1 {
			c.UI.Output(fmt.Sprintf("Endpoint addresses must have the port `%d`", c.flagPort), terminal.WithInfoStyle())
		}
	}

	c.outputClustersTable(FilterClusters(config.Clusters, c.flagFQDN, c.flagAddress, c.flagPort))
	c.outputEndpointsTable(FilterEndpoints(config.Endpoints, c.flagAddress, c.flagPort))
	c.outputListenersTable(FilterListeners(config.Listeners, c.flagAddress, c.flagPort))
	c.outputRoutesTable(config.Routes)
	c.outputSecretsTable(config.Secrets)
}

func (c *ReadCommand) outputClustersTable(clusters []Cluster) {
	if c.areTablesFiltered() && !c.flagClusters {
		return
	}

	c.UI.Output(fmt.Sprintf("Clusters (%d)", len(clusters)), terminal.WithHeaderStyle())
	table := terminal.NewTable("Name", "FQDN", "Endpoints", "Type", "Last Updated")
	for _, cluster := range clusters {
		table.AddRow([]string{cluster.Name, cluster.FullyQualifiedDomainName, strings.Join(cluster.Endpoints, ", "),
			cluster.Type, cluster.LastUpdated}, []string{})
	}
	c.UI.Table(table)
	c.UI.Output("")
}

func (c *ReadCommand) outputEndpointsTable(endpoints []Endpoint) {
	if c.areTablesFiltered() && !c.flagEndpoints {
		return
	}

	c.UI.Output(fmt.Sprintf("Endpoints (%d)", len(endpoints)), terminal.WithHeaderStyle())
	table := terminal.NewTable("Address:Port", "Cluster", "Weight", "Status")
	for _, endpoint := range endpoints {
		var statusColor string
		if endpoint.Status == "HEALTHY" {
			statusColor = "green"
		} else {
			statusColor = "red"
		}

		table.AddRow(
			[]string{endpoint.Address, endpoint.Cluster, fmt.Sprintf("%.2f", endpoint.Weight), endpoint.Status},
			[]string{"", "", "", statusColor})
	}
	c.UI.Table(table)
	c.UI.Output("")
}

func (c *ReadCommand) outputListenersTable(listeners []Listener) {
	if c.areTablesFiltered() && !c.flagListeners {
		return
	}

	c.UI.Output(fmt.Sprintf("Listeners (%d)", len(listeners)), terminal.WithHeaderStyle())
	table := terminal.NewTable("Name", "Address:Port", "Direction", "Filter Chain Match", "Filters", "Last Updated")
	for _, listener := range listeners {
		for index, filter := range listener.FilterChain {
			// Print each element of the filter chain in a separate line
			// without repeating the name, address, etc.
			filters := strings.Join(filter.Filters, "\n")
			if index == 0 {
				table.AddRow(
					[]string{listener.Name, listener.Address, listener.Direction, filter.FilterChainMatch, filters, listener.LastUpdated},
					[]string{})
			} else {
				table.AddRow(
					[]string{"", "", "", filter.FilterChainMatch, filters},
					[]string{})
			}
		}
	}
	c.UI.Table(table)
	c.UI.Output("")
}

func (c *ReadCommand) outputRoutesTable(routes []Route) {
	if c.areTablesFiltered() && !c.flagRoutes {
		return
	}

	c.UI.Output(fmt.Sprintf("Routes (%d)", len(routes)), terminal.WithHeaderStyle())
	table := terminal.NewTable("Name", "Destination Cluster", "Last Updated")
	for _, route := range routes {
		table.AddRow([]string{route.Name, route.DestinationCluster, route.LastUpdated}, []string{})
	}
	c.UI.Table(table)
	c.UI.Output("")
}

func (c *ReadCommand) outputSecretsTable(secrets []Secret) {
	if c.areTablesFiltered() && !c.flagSecrets {
		return
	}

	c.UI.Output(fmt.Sprintf("Secrets (%d)", len(secrets)), terminal.WithHeaderStyle())
	table := terminal.NewTable("Name", "Type", "Last Updated")
	for _, secret := range secrets {
		table.AddRow([]string{secret.Name, secret.Type, secret.LastUpdated}, []string{})
	}
	c.UI.Table(table)
	c.UI.Output("")
}
