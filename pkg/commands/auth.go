package commands

import (
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/auth"
	"github.com/urfave/cli/v2"
)

var AuthCommand = &cli.Command{
	Name:  "auth",
	Usage: "Manage authentication with private keys stored in OS keyring",
	Subcommands: []*cli.Command{
		auth.GenerateCommand,
		auth.LoginCommand,
		auth.LogoutCommand,
		auth.WhoamiCommand,
		auth.ListCommand,
	},
}
