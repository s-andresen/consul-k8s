package get

type FlagConfig struct {
	Name    string
	Default interface{}
	Aliases []string
	Usage   string
}

// Flag names and defaults
var (
	flagName = FlagConfig{
		Name:    "name",
		Aliases: []string{"n"},
		Usage:   "The name of the Pod to get the Envoy config from.",
	}
	flagNamespace = FlagConfig{
		Name:    "namespace",
		Default: "default",
		Aliases: []string{"ns"},
		Usage:   "Namespace to look for the Envoy configuration in.",
	}
)
