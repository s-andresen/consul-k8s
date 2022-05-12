package list

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/common/terminal"
)

// Command is the proxy list command.
type Command struct {
	*common.BaseCommand

	set *flag.Sets

	flagNamespace     string
	flagAllNamespaces bool

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string

	action *Action
}

// init registers the command with the CLI.
func (c *Command) init() {
	c.set = flag.NewSets()

	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name:    "namespace",
		Aliases: []string{"n"},
		Target:  &c.flagNamespace,
		Default: "default",
		Usage:   "The namespace to list proxies in.",
	})
	f.BoolVar(&flag.BoolVar{
		Name:    "all-namespaces",
		Aliases: []string{"a"},
		Target:  &c.flagAllNamespaces,
		Usage:   "List proxies in all Kubernetes namespaces.",
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

	c.Init()
}

func (c *Command) Run(args []string) int {
	c.once.Do(c.init)
	c.Log.ResetNamed("proxy list")
	defer common.CloseWithError(c.BaseCommand)

	if err := c.set.Parse(args); err != nil {
		c.UI.Output(fmt.Sprintf("Unable to parse arguments. %s", err.Error()), terminal.WithErrorStyle())
		return 1
	}

	if err := c.validateArgs(); err != nil {
		c.UI.Output(fmt.Sprintf("Unable to validate arguments. %s", err.Error()), terminal.WithErrorStyle())
		return 1
	}

	// Configure the action.
	c.action = &Action{}

	if err := c.action.Run(); err != nil {
		c.UI.Output(err.Error(), terminal.WithErrorStyle())
		return 1
	}

	return 0
}

func (c *Command) Synopsis() string {
	return "List all Kubernetes Pods running sidecar proxies in a namespace."
}

func (c *Command) Help() string {
	c.once.Do(c.init)
	return c.help
}

func (c *Command) validateArgs() error {
	return nil
}
