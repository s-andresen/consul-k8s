package common

import "encoding/json"

type EnvoyConfig struct {
	Bootstrap    map[string]interface{}
	Clusters     map[string]interface{}
	Endpoints    interface{}
	Listeners    map[string]interface{}
	ScopedRoutes interface{}
	Routes       interface{}
	Secrets      interface{}
}

func NewEnvoyConfig(raw []byte) EnvoyConfig {
	var config map[string]interface{}
	var envCfg EnvoyConfig

	json.Unmarshal(raw, &config)

	for _, c := range config["configs"].([]interface{}) {
		a := c.(map[string]interface{})
		cfgType := a["@type"].(string)

		switch cfgType {
		case "type.googleapis.com/envoy.admin.v3.BootstrapConfigDump":
			envCfg.Bootstrap = a
		case "type.googleapis.com/envoy.admin.v3.ClustersConfigDump":
			envCfg.Clusters = a
		case "type.googleapis.com/envoy.admin.v3.EndpointsConfigDump":
			envCfg.Endpoints = a
		case "type.googleapis.com/envoy.admin.v3.ListenersConfigDump":
			envCfg.Listeners = a
		case "type.googleapis.com/envoy.admin.v3.RoutesConfigDump":
			envCfg.Routes = a
		case "type.googleapis.com/envoy.admin.v3.ScopedRoutesConfigDump":
			envCfg.ScopedRoutes = a
		case "type.googleapis.com/envoy.admin.v3.SecretsConfigDump":
			envCfg.Secrets = a
		}
	}

	return envCfg
}
