package read

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-k8s/cli/common/terminal"
)

func PrintClusters(ui terminal.UI, clusters map[string]interface{}) error {
	ui.Output("Clusters:", terminal.WithHeaderStyle())

	tbl := terminal.NewTable("Name", "FQDN", "Endpoints", "Type", "Last Updated")

	if clusters["static_clusters"] != nil {

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
	}
	if clusters["dynamic_active_clusters"] != nil {
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
	}
	ui.Table(tbl)

	return nil
}
