package ingressconfig

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"github.com/hashicorp/consul-k8s/cli/format"
	"k8s.io/client-go/kubernetes"
)

// Command is the ingress-config command.
type Command struct {
	*common.BaseCommand

	set *flag.Sets

	// Command options
	flagFullConfig bool
	flagFormat     string

	// Global options
	flagKubeConfig  string
	flagKubeContext string

	kubernetes kubernetes.Interface

	once sync.Once
	help string
}

func (c *Command) Run(args []string) int {
	c.once.Do(c.init)
	c.Log.ResetNamed("ingress-config")
	defer common.CloseWithError(c.BaseCommand)

	c.UI.Output("Not implemented")

	return 0
}

func (c *Command) Help() string {
	c.once.Do(c.init)
	return c.Synopsis() + "\n\nUsage: consul-k8s ingress-config [flags]\n\n" + c.help
}

func (c *Command) Synopsis() string {
	return "Get the proxy configuration for Consul Ingress."
}

func (c *Command) init() {
	// Set up the flags.
	c.set = flag.NewSets()

	f := c.set.NewSet("Command Options")
	f.BoolVar(&flag.BoolVar{
		Name:   "full-config",
		Usage:  "Return the full proxy configuration.",
		Target: &c.flagFullConfig,
	})
	f.StringVar(&flag.StringVar{
		Name:    "format",
		Usage:   "The output format (JSON, YAML).",
		Aliases: []string{"o"},
		Target:  &c.flagFormat,
	})

	f = c.set.NewSet("Global Options")
	f.StringVar(&flag.StringVar{
		Name:    "kubeconfig",
		Usage:   "The path to the Kubernetes config file.",
		Aliases: []string{"c"},
		Target:  &c.flagKubeConfig,
	})
	f.StringVar(&flag.StringVar{
		Name:   "context",
		Usage:  "The name of the Kubernetes context to use.",
		Target: &c.flagKubeContext,
	})

	c.help = c.set.Help()

	c.Init()
}

func (c *Command) validateFlags() error {
	if (len(c.set.Args())) > 0 {
		return fmt.Errorf("non-flag arguments given: %s", strings.Join(c.set.Args(), ", "))
	}

	return nil
}

func (c *Command) setupKubernetes() error {
	if c.kubernetes != nil {
		return nil
	}

	var err error
	c.kubernetes, err = common.CreateKubernetesClient(c.flagKubeConfig, c.flagKubeContext)
	return err
}

func (c *Command) fetchConfig() (string, error) {
	// This will use the Kubernetes API in the final version.
	output, err := exec.Command(
		"kubectl", "exec", "", "--namespace", "",
		"-c", "envoy-sidecar", "--", "wget", "-qO-", "127.0.0.1:19000/config_dump",
	).Output()

	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (c *Command) outputConfig(config string) {
	if !c.flagFullConfig {
		c.UI.Output(format.FormatEnvoyConfig(config))
		return
	}

	c.UI.Output(config)
}
