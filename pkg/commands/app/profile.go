package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var ProfileCommand = &cli.Command{
	Name:      "profile",
	Usage:     "Manage public app profile",
	ArgsUsage: "<app-id|name>",
	Subcommands: []*cli.Command{
		{
			Name:      "set",
			Usage:     "Set public profile information for an app",
			ArgsUsage: "<app-id|name>",
			Flags: append(common.GlobalFlags, []cli.Flag{
				common.EnvironmentFlag,
				common.RpcUrlFlag,
				common.NameFlag,
				common.WebsiteFlag,
				common.DescriptionFlag,
				common.XURLFlag,
				common.ImageFlag,
			}...),
			Action: profileSetAction,
		},
	},
}

func profileSetAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Get app ID
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "set profile for")
	if err != nil {
		return err
	}

	logger.Info("Setting profile for app: %s", appID.Hex())

	// Collect profile fields using shared function
	profile, err := utils.GetAppProfileInteractive(cCtx, "", false)
	if err != nil {
		return err
	}

	// Upload profile via API
	logger.Info("Uploading app profile...")

	userApiClient, err := utils.NewUserApiClient(cCtx)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	response, err := userApiClient.UploadAppProfile(cCtx, appID.Hex(), profile.Name, profile.Website, profile.Description, profile.XURL, profile.ImagePath)
	if err != nil {
		return fmt.Errorf("failed to upload profile: %w", err)
	}

	// Display success message with returned data
	logger.Info("✓ Profile updated successfully for app '%s'", response.Name)

	// Show uploaded profile data
	fmt.Println("\nUploaded Profile:")
	fmt.Printf("  Name:        %s\n", response.Name)
	if response.Website != nil {
		fmt.Printf("  Website:     %s\n", *response.Website)
	}
	if response.Description != nil {
		fmt.Printf("  Description: %s\n", *response.Description)
	}
	if response.XURL != nil {
		fmt.Printf("  X URL:       %s\n", *response.XURL)
	}
	if response.ImageURL != nil {
		fmt.Printf("  Image URL:   %s\n", *response.ImageURL)
	}

	return nil
}
