package commands

import (
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/billing"
	"github.com/urfave/cli/v2"
)

var BillingCommand = &cli.Command{
	Name:  "billing",
	Usage: "Manage billing and subscription",
	Subcommands: []*cli.Command{
		billing.SubscribeCommand,
		billing.CancelCommand,
		billing.StatusCommand,
	},
}
