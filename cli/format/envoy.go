package format

import (
	"encoding/json"
	"fmt"
)

func FormatEnvoyConfig(config string) (string, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		return "", err
	}

	configs := parsed["configs"].([]interface{})
	var out string
	for _, c := range configs {
		out += fmt.Sprintf("%s\n", format(c.(map[string]interface{})))
	}

	// Just bypass formatting for now.
	return config, nil
}

func format(cfg map[string]interface{}) string {
	var out string

	switch {
	case cfg["@type"] == "type.googleapis.com/envoy.admin.v3.BootstrapConfigDump":
		out += formatBootstrapConfigDump(cfg)
	}

	return out
}

func formatBootstrapConfigDump(cfg map[string]interface{}) string {
	var out string

	out += "Bootstrap Config\n"
	out += "----------------\n"

	return out
}
