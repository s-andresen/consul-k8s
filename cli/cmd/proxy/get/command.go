package get

import (
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/envoy"
	"github.com/hashicorp/consul-k8s/cli/validation"
	"k8s.io/client-go/kubernetes"
)

// Command is the command to get an Envoy config from a Pod
type Command struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	flagName      string
	flagNamespace string

	flagKubeConfig  string
	flagKubeContext string

	set *flag.Sets

	once sync.Once
	help string
}

func (c *Command) init() {
	c.set = flag.NewSets()
	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    flagName.Name,
		Target:  &c.flagName,
		Aliases: flagName.Aliases,
		Usage:   flagName.Usage,
	})
	f.StringVar(&flag.StringVar{
		Name:    flagNamespace.Name,
		Target:  &c.flagNamespace,
		Aliases: flagNamespace.Aliases,
		Usage:   flagNamespace.Usage,
		Default: flagNamespace.Default.(string),
	})

	f = c.set.NewSet("Global Options")
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
	c.once.Do(c.init)
	c.Log.ResetNamed("proxy get")
	defer common.CloseWithError(c.BaseCommand)

	if err := c.set.Parse(args); err != nil {
		c.Log.Error(err.Error())
		return 1
	}

	if err := validation.PodExists(c.Ctx, c.kubernetes, c.flagNamespace, c.flagName); err != nil {
		c.Log.Error(err.Error())
		return 1
	}

	config, err := envoy.FetchConfig(c.Ctx, "", c.flagNamespace, c.flagName)
	if err != nil {
		c.Log.Error(err.Error())
		return 1
	}

	c.Log.Info(string(config))

	return 0
}

// Help returns a description of the command and how it is used.
func (c *Command) Help() string {
	c.once.Do(c.init)
	return c.Synopsis() + "\n\nUsage: consul-k8s proxy get [flags]\n\n" + c.help
}

// Synopsis returns a one-line command summary.
func (c *Command) Synopsis() string {
	return "Fetch an Envoy config from a Kubernetes Pod."
}
