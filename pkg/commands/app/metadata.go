package app

import (
	"fmt"

	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

var MetadataCommand = &cli.Command{
	Name:      "metadata",
	Usage:     "Manage app metadata",
	ArgsUsage: "<app-id|name>",
	Subcommands: []*cli.Command{
		{
			Name:      "set",
			Usage:     "Set metadata for an app",
			ArgsUsage: "<app-id|name>",
			Flags: append(common.GlobalFlags, []cli.Flag{
				common.EnvironmentFlag,
				common.RpcUrlFlag,
				&cli.StringFlag{
					Name:     "name",
					Usage:    "App display name (required)",
					Required: false,
				},
				&cli.StringFlag{
					Name:  "website",
					Usage: "App website URL (optional)",
				},
				&cli.StringFlag{
					Name:  "description",
					Usage: "App description (optional)",
				},
				&cli.StringFlag{
					Name:  "x-url",
					Usage: "X (Twitter) profile URL (optional)",
				},
				&cli.StringFlag{
					Name:  "image",
					Usage: "Path to app icon/logo image (JPG/PNG, max 4MB, square recommended, optional)",
				},
			}...),
			Action: metadataSetAction,
		},
	},
}

func metadataSetAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Get app ID
	appID, err := utils.GetAppIDInteractive(cCtx, 0, "set metadata for")
	if err != nil {
		return err
	}

	logger.Info("Setting metadata for app: %s", appID.Hex())

	// Collect metadata fields
	name, err := utils.GetAppNameInteractive(cCtx)
	if err != nil {
		return err
	}

	website, err := utils.GetAppWebsiteInteractive(cCtx)
	if err != nil {
		return err
	}

	description, err := utils.GetAppDescriptionInteractive(cCtx)
	if err != nil {
		return err
	}

	xURL, err := utils.GetAppXURLInteractive(cCtx)
	if err != nil {
		return err
	}

	imagePath, err := utils.GetAppImageInteractive(cCtx)
	if err != nil {
		return err
	}

	// Display metadata for confirmation
	fmt.Println(formatMetadataForDisplay(name, website, description, xURL, imagePath))

	confirmed, err := output.Confirm("Upload this metadata?")
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("operation cancelled")
	}

	// Upload profile via API
	logger.Info("Uploading app metadata...")

	userApiClient, err := utils.NewUserApiClient(cCtx)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	response, err := userApiClient.UploadAppProfile(cCtx, appID.Hex(), name, website, description, xURL, imagePath)
	if err != nil {
		return fmt.Errorf("failed to upload profile: %w", err)
	}

	// Display success message with returned data
	logger.Info("✓ Metadata updated successfully for app '%s'", response.Name)

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

// formatMetadataForDisplay formats metadata for display to the user
func formatMetadataForDisplay(name string, website, description, xURL *string, imagePath string) string {
	output := "\nApp Metadata:\n"
	output += fmt.Sprintf("  Name:        %s\n", name)

	if website != nil && *website != "" {
		output += fmt.Sprintf("  Website:     %s\n", *website)
	} else {
		output += "  Website:     (not provided)\n"
	}

	if description != nil && *description != "" {
		output += fmt.Sprintf("  Description: %s\n", *description)
	} else {
		output += "  Description: (not provided)\n"
	}

	if xURL != nil && *xURL != "" {
		output += fmt.Sprintf("  X URL:       %s\n", *xURL)
	} else {
		output += "  X URL:       (not provided)\n"
	}

	if imagePath != "" {
		output += fmt.Sprintf("  Image:       %s\n", imagePath)
	} else {
		output += "  Image:       (not provided)\n"
	}

	return output
}
