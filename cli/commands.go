package main

import (
	"context"

	"github.com/hashicorp/consul-k8s/cli/cmd/install"
	"github.com/hashicorp/consul-k8s/cli/cmd/proxy/clusters"
	"github.com/hashicorp/consul-k8s/cli/cmd/proxy/config"
	"github.com/hashicorp/consul-k8s/cli/cmd/proxy/list"
	"github.com/hashicorp/consul-k8s/cli/cmd/proxy/listeners"
	"github.com/hashicorp/consul-k8s/cli/cmd/proxy/secrets"
	"github.com/hashicorp/consul-k8s/cli/cmd/status"
	"github.com/hashicorp/consul-k8s/cli/cmd/uninstall"
	"github.com/hashicorp/consul-k8s/cli/cmd/upgrade"
	cmdversion "github.com/hashicorp/consul-k8s/cli/cmd/version"
	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/version"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
)

func initializeCommands(ctx context.Context, log hclog.Logger) (*common.BaseCommand, map[string]cli.CommandFactory) {

	baseCommand := &common.BaseCommand{
		Ctx: ctx,
		Log: log,
	}

	commands := map[string]cli.CommandFactory{
		"install": func() (cli.Command, error) {
			return &install.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"uninstall": func() (cli.Command, error) {
			return &uninstall.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &status.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"upgrade": func() (cli.Command, error) {
			return &upgrade.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &cmdversion.Command{
				BaseCommand: baseCommand,
				Version:     version.GetHumanVersion(),
			}, nil
		},
		"proxy list": func() (cli.Command, error) {
			return &list.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"proxy config": func() (cli.Command, error) {
			return &config.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"proxy secrets": func() (cli.Command, error) {
			return &secrets.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"proxy listeners": func() (cli.Command, error) {
			return &listeners.Command{
				BaseCommand: baseCommand,
			}, nil
		},
		"proxy clusters": func() (cli.Command, error) {
			return &clusters.Command{
				BaseCommand: baseCommand,
			}, nil
		},
	}

	return baseCommand, commands
}
