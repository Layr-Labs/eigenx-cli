package common

import "github.com/urfave/cli/v2"

// Common flag definitions
var (
	EnvironmentFlag = &cli.StringFlag{
		Name:  "environment",
		Usage: "Deployment environment to use",
	}

	RpcUrlFlag = &cli.StringFlag{
		Name:    "rpc-url",
		Usage:   "RPC URL to connect to blockchain",
		EnvVars: []string{"RPC_URL"},
	}

	PrivateKeyFlag = &cli.StringFlag{
		Name:    "private-key",
		Usage:   "Private key for signing transactions",
		EnvVars: []string{"PRIVATE_KEY"},
	}

	ForceFlag = &cli.BoolFlag{
		Name:  "force",
		Usage: "Force operation without confirmation",
	}

	EnvFlag = &cli.StringFlag{
		Name:  "env-file",
		Usage: "Environment file to use",
		Value: ".env",
	}

	ImageNameFlag = &cli.StringFlag{
		Name:  "image-name",
		Usage: "Override app/image name (auto-detected from context if not provided)",
	}

	FileFlag = &cli.StringFlag{
		Name:    "dockerfile",
		Aliases: []string{"f"},
		Usage:   "Path to Dockerfile",
	}

	TemplateFlag = &cli.StringFlag{
		Name:  "template",
		Usage: "Template repository URL",
	}

	TemplateVersionFlag = &cli.StringFlag{
		Name:  "template-version",
		Usage: "Template version/tag to use",
	}

	AllFlag = &cli.BoolFlag{
		Name:  "all",
		Usage: "Show all apps including terminated ones",
	}

	AddressCountFlag = &cli.IntFlag{
		Name:  "address-count",
		Usage: "Number of addresses to fetch",
		Value: 1,
	}

	NameFlag = &cli.StringFlag{
		Name:  "name",
		Usage: "Friendly name for the app",
	}

	LogVisibilityFlag = &cli.StringFlag{
		Name:  "log-visibility",
		Usage: "Log visibility setting: public, private, or off",
	}

	InstanceTypeFlag = &cli.StringFlag{
		Name:  "instance-type",
		Usage: "Machine instance type to use (g1-standard-4t, g1-standard-8t)",
	}

	WatchFlag = &cli.BoolFlag{
		Name:    "watch",
		Aliases: []string{"w"},
		Usage:   "Continuously fetch and display updates",
	}
)

// GlobalFlags defines flags that apply to the entire application (global flags).
var GlobalFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "Enable verbose logging",
	},
	&cli.BoolFlag{
		Name:  "enable-telemetry",
		Usage: "Enable telemetry collection on first run without prompting",
	},
	&cli.BoolFlag{
		Name:  "disable-telemetry",
		Usage: "Disable telemetry collection on first run without prompting",
	},
}

func ForceFlagWithUsage(usage string) *cli.BoolFlag {
	requiredFlag := *ForceFlag
	requiredFlag.Usage = usage
	return &requiredFlag
}
