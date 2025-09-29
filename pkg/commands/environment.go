package commands

import (
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/environment"
	"github.com/urfave/cli/v2"
)

var EnvironmentCommand = &cli.Command{
	Name:    "environment",
	Aliases: []string{"env"},
	Usage:   "Manage deployment environment",
	Subcommands: []*cli.Command{
		environment.SetCommand,
		environment.ListCommand,
		environment.ShowCommand,
	},
}
