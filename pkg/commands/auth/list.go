package auth

import (
	"fmt"
	"sort"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

var ListCommand = &cli.Command{
	Name:   "list",
	Usage:  "List all stored private keys by deployment environment",
	Flags:  common.GlobalFlags,
	Action: listAction,
}

func listAction(cCtx *cli.Context) error {
	// Get all stored keys
	keys, err := ListStoredKeys()
	if err != nil {
		return fmt.Errorf("failed to list stored keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No keys stored in keyring")
		fmt.Println("")
		fmt.Println("To store a key, use:")
		fmt.Println("  eigenx auth login")
		return nil
	}

	// Sort keys by name for consistent output
	var sortedNames []string
	for name := range keys {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Display header
	fmt.Println("Stored private keys:")
	fmt.Println("")

	// Display each key
	for _, name := range sortedNames {
		address := keys[name]
		fmt.Printf("  %-12s %s\n", name, address)
	}

	// Show help text
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  eigenx auth login                    # Store key for active deployment environment")
	fmt.Println("  eigenx auth logout                   # Remove key for active deployment environment")
	fmt.Println("  eigenx --environment <env> <command> # Use different deployment environment")

	return nil
}
