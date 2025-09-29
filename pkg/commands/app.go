package commands

import (
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/app"
	"github.com/urfave/cli/v2"
)

var AppCommand = &cli.Command{
	Name:  "app",
	Usage: "Manage projects",
	Subcommands: []*cli.Command{
		app.CreateCommand,
		app.DeployCommand,
		app.UpgradeCommand,
		app.StartCommand,
		app.StopCommand,
		app.TerminateCommand,
		app.ListCommand,
		app.InfoCommand,
		app.LogsCommand,
		app.NameCommand,
		app.ConfigureTLSCommand,
	},
}
