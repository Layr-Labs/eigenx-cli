package utils

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v1/AppController"
	dockercommand "github.com/docker/cli/cli/command"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// registryInfo holds information about an authenticated Docker registry
type registryInfo struct {
	URL      string
	Username string
	Type     string // "dockerhub", "ghcr", "gcr", "other"
}

// SelectRegistryInteractive provides interactive selection of registry for image reference
func SelectRegistryInteractive(registries []registryInfo, imageName string, tag string, promptMessage string, validator func(string) error) (string, error) {
	// If we have registries, offer them as choices
	if len(registries) == 1 {
		// Single registry - suggest it as default
		defaultRef := suggestImageReference(registries[0], imageName, tag)
		imageRef, err := output.InputString(
			"Enter image reference:",
			promptMessage,
			defaultRef,
			validator,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get image reference: %w", err)
		}
		return imageRef, nil
	}

	// Multiple registries - let user choose
	options := []string{}
	for _, reg := range registries {
		suggestion := suggestImageReference(reg, imageName, tag)
		options = append(options, suggestion)
	}
	options = append(options, "Enter custom image reference")

	choice, err := output.SelectString("Select image destination:", options)
	if err != nil {
		return "", fmt.Errorf("failed to select registry: %w", err)
	}

	if choice == "Enter custom image reference" {
		imageRef, err := output.InputString(
			"Enter image reference:",
			promptMessage,
			"",
			validator,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get image reference: %w", err)
		}
		return imageRef, nil
	}
	return choice, nil
}

// GetImageReferenceInteractive prompts for image reference if not provided
func GetImageReferenceInteractive(cCtx *cli.Context, argIndex int, buildFromDockerfile bool) (string, error) {
	// Check if provided as argument
	if cCtx.Args().Len() > argIndex {
		return cCtx.Args().Get(argIndex), nil
	}

	// Get available registries
	registries, _ := getAvailableRegistries()

	// Get default app name for suggestions
	appName := getDefaultAppName()

	// Interactive prompt
	if buildFromDockerfile {
		fmt.Println("\n📦 Build & Push Configuration")
		fmt.Println("Your Docker image will be built and pushed to a registry")
		fmt.Println("so that EigenX can pull and run it in the TEE.")
		fmt.Println()

		if len(registries) > 0 {
			displayDetectedRegistries(registries, appName)
			return SelectRegistryInteractive(registries, appName, "latest", "Where to push your built image", validateImageReference)
		}

		// No registries detected
		displayAuthenticationInstructions()
	} else {
		fmt.Println("\n🐳 Docker Image Selection")
		fmt.Println("Specify an existing Docker image from a registry to run in the TEE.")
		fmt.Println()
	}

	// Fallback to manual input
	displayRegistryExamples(appName)

	imageRef, err := output.InputString(
		"Enter Docker image reference:",
		"The image reference to use",
		"",
		validateImageReference,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get image reference: %w", err)
	}

	return imageRef, nil
}

// GetLayeredTargetImageInteractive prompts for target image when layering a published image
func GetLayeredTargetImageInteractive(cCtx *cli.Context, sourceImageRef string) (string, error) {
	// Display warning and explanation
	fmt.Println()
	fmt.Printf("Warning: The image '%s' is missing EigenX components.", sourceImageRef)
	fmt.Println()
	confirmed, err := output.ConfirmWithDefault("Would you like to layer the image with EigenX components?", true)
	if err != nil {
		return "", fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return "", fmt.Errorf("user cancelled")
	}

	// Show registry information and configuration
	fmt.Println("\n📦 Layer & Push Configuration")
	fmt.Println("Your image will be layered with EigenX components and pushed to a registry")
	fmt.Println("so that EigenX can pull and run it in the TEE.")
	fmt.Println()

	// Detect available registries
	registries, err := getAvailableRegistries()

	// Extract base image name and tag from source for suggestions
	baseImage, tag := extractImageNameAndTag(sourceImageRef)
	layeredTag := tag + "-eigenx"

	// Create validator that ensures target is different from source
	validator := func(ref string) error {
		if err := validateImageReference(ref); err != nil {
			return err
		}
		if ref == sourceImageRef {
			return fmt.Errorf("target must be different from source image '%s'", sourceImageRef)
		}
		return nil
	}

	if err == nil && len(registries) > 0 {
		displayDetectedRegistries(registries, baseImage)
		return SelectRegistryInteractive(registries, baseImage, layeredTag, "Where the EigenX-compatible version will be published", validator)
	}

	// No registries detected
	displayAuthenticationInstructions()

	// Fallback to manual input
	fmt.Println("Examples:")
	fmt.Printf("  • username/%s:%s (Docker Hub)\n", baseImage, layeredTag)
	fmt.Printf("  • ghcr.io/username/%s:%s (GitHub)\n", baseImage, layeredTag)
	fmt.Printf("  • gcr.io/project-id/%s:%s (Google)\n", baseImage, layeredTag)
	fmt.Println()

	// Use simple default suffix
	var defaultTarget string
	if strings.Contains(sourceImageRef, ":") {
		defaultTarget = sourceImageRef + "-eigenx"
	} else {
		defaultTarget = sourceImageRef + ":eigenx"
	}

	targetRef, err := output.InputString(
		"Enter target image reference:",
		"Where the EigenX-compatible version will be published",
		defaultTarget,
		validator,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get target image reference: %w", err)
	}

	return targetRef, nil
}

// GetAppIDInteractive gets app ID from args or interactive selection
func GetAppIDInteractive(cCtx *cli.Context, argIndex int, action string) (ethcommon.Address, error) {
	// First try to get from args
	appID, err := GetAppID(cCtx, argIndex)
	if err == nil {
		return appID, nil
	}

	// If not provided, show interactive selection
	fmt.Printf("\nSelect an app to %s:\n", action)

	// Get list of apps for the user
	client, appController, err := GetAppControllerBinding(cCtx)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to connect to contract: %w", err)
	}
	defer client.Close()

	developerAddr, err := GetDeveloperAddress(cCtx)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to get developer address: %w", err)
	}

	result, err := appController.GetAppsByDeveloper(&bind.CallOpts{Context: cCtx.Context}, developerAddr, big.NewInt(0), big.NewInt(50))
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to list apps: %w", err)
	}

	if len(result.Apps) == 0 {
		return ethcommon.Address{}, fmt.Errorf("no apps found for your address")
	}

	// Get environment config for context
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to get environment config: %w", err)
	}

	// Build apps list with status priority
	type appItem struct {
		addr    ethcommon.Address
		config  interface{}
		display string
		status  common.AppStatus
		index   int // Index from contract result (newer apps have higher indices)
	}

	// Get API statuses for all Started apps to identify which have exited
	exitedApps := getExitedApps(cCtx, result.Apps, result.AppConfigsMem)

	// Determine which apps are eligible for the action
	isEligible := func(status common.AppStatus, addr ethcommon.Address) bool {
		switch action {
		case "view":
			return true
		case "start":
			return status == common.ContractAppStatusStopped || exitedApps[addr.Hex()]
		case "stop":
			return status == common.ContractAppStatusStarted || exitedApps[addr.Hex()]
		default:
			return status != common.ContractAppStatusTerminated
		}
	}

	var appItems []appItem
	for i, appAddr := range result.Apps {
		config := result.AppConfigsMem[i]
		status := common.AppStatus(config.Status)

		if !isEligible(status, appAddr) {
			continue
		}

		statusStr := getStatusString(status)
		if exitedApps[appAddr.Hex()] {
			statusStr = "Exited"
		}

		appName := common.GetAppName(environmentConfig.Name, appAddr.Hex())
		displayName := appAddr.Hex()
		if appName != "" {
			displayName = fmt.Sprintf("%s (%s)", appName, appAddr.Hex())
		}

		appItems = append(appItems, appItem{
			addr:    appAddr,
			config:  config,
			display: fmt.Sprintf("%s - %s", displayName, statusStr),
			status:  status,
			index:   i,
		})
	}

	// Sort by status priority: Started > Exited > Stopped > Terminated
	// Within same status, show newest apps first (higher index = newer)
	sort.Slice(appItems, func(i, j int) bool {
		iIsExited := exitedApps[appItems[i].addr.Hex()]
		jIsExited := exitedApps[appItems[j].addr.Hex()]

		iPriority := getStatusPriority(appItems[i].status, iIsExited)
		jPriority := getStatusPriority(appItems[j].status, jIsExited)

		// First compare by status priority
		if iPriority != jPriority {
			return iPriority < jPriority
		}

		// If same status, sort by index descending (newer apps first)
		return appItems[i].index > appItems[j].index
	})

	// Build final options and activeApps lists
	var options []string
	var activeApps []ethcommon.Address
	for _, item := range appItems {
		options = append(options, item.display)
		activeApps = append(activeApps, item.addr)
	}

	if len(options) == 0 {
		switch action {
		case "start":
			return ethcommon.Address{}, fmt.Errorf("no startable apps found - only Stopped apps can be started")
		case "stop":
			return ethcommon.Address{}, fmt.Errorf("no running apps found - only Running apps can be stopped")
		default:
			return ethcommon.Address{}, fmt.Errorf("no active apps found")
		}
	}

	selected, err := output.SelectString("Select app:", options)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("failed to select app: %w", err)
	}

	// Find the selected app
	for i, option := range options {
		if option == selected {
			return activeApps[i], nil
		}
	}

	return ethcommon.Address{}, fmt.Errorf("failed to find selected app")
}

// GetOrPromptAppName gets app name from flag or prompts interactively
func GetOrPromptAppName(cCtx *cli.Context, context string, imageRef string) (string, error) {
	// Check if provided via flag
	if name := cCtx.String(common.NameFlag.Name); name != "" {
		// Validate the provided name
		if err := common.ValidateAppName(name); err != nil {
			return "", fmt.Errorf("invalid app name: %w", err)
		}
		// Check if it's available
		if !IsAppNameAvailable(context, name) {
			fmt.Printf("Warning: App name '%s' is already taken.\n", name)
			return GetAvailableAppNameInteractive(context, imageRef)
		}
		return name, nil
	}

	// No flag provided, get interactively
	return GetAvailableAppNameInteractive(context, imageRef)
}

// GetAvailableAppNameInteractive interactively gets an available app name
func GetAvailableAppNameInteractive(context, imageRef string) (string, error) {
	// Start with a suggestion from the image
	baseName, err := extractAppNameFromImage(imageRef)
	if err != nil {
		return "", fmt.Errorf("failed to extract app name from image reference %s: %w", imageRef, err)
	}

	// Find the first available name based on the suggestion
	suggestedName := findAvailableName(context, baseName)

	for {
		fmt.Printf("\nApp name selection:\n")
		name, err := output.InputString(
			"Enter app name:",
			fmt.Sprintf("A friendly name to identify your app (suggested: %s)", suggestedName),
			suggestedName,
			common.ValidateAppName,
		)
		if err != nil {
			// If input fails, use the suggestion
			name = suggestedName
		}

		// Check if the name is available
		if IsAppNameAvailable(context, name) {
			return name, nil
		}

		// Name is taken, suggest alternatives and loop
		fmt.Printf("App name '%s' is already taken.\n", name)

		// Suggest alternatives based on their input
		suggestedName = findAvailableName(context, name)
		fmt.Printf("Suggested alternative: %s\n", suggestedName)
	}
}

// IsAppNameAvailable checks if an app name is available in the given context
func IsAppNameAvailable(context, name string) bool {
	apps, _ := common.ListApps(context)
	_, exists := apps[name]
	return !exists
}

// GetEnvFileInteractive prompts for env file path if not provided
func GetEnvFileInteractive(cCtx *cli.Context) (string, error) {
	// Check if provided via flag and exists
	if envFile := cCtx.String(common.EnvFlag.Name); envFile != "" {
		if _, err := os.Stat(envFile); err == nil {
			return envFile, nil
		}
		// Flag provided but file doesn't exist, continue to interactive prompt
	}

	// Check if default .env exists
	if _, err := os.Stat(".env"); err == nil {
		return ".env", nil
	}

	// Interactive prompt when env file doesn't exist
	fmt.Println("\nEnvironment file not found.")
	fmt.Println("Environment files contain variables like RPC_URL, etc.")

	options := []string{
		"Enter path to existing env file",
		"Continue without env file",
	}

	choice, err := output.SelectString("Choose an option:", options)
	if err != nil {
		return "", fmt.Errorf("failed to get environment file choice: %w", err)
	}

	switch choice {
	case "Enter path to existing env file":
		envFile, err := output.InputString(
			"Enter environment file path:",
			"Path to environment file (e.g., .env.prod, config/.env)",
			"",
			validateFilePath,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get environment file path: %w", err)
		}
		return envFile, nil
	case "Continue without env file":
		return "", nil
	default:
		return "", fmt.Errorf("unexpected choice: %s", choice)
	}
}

// GetDockerfileInteractive prompts to build from Dockerfile if it exists
func GetDockerfileInteractive(cCtx *cli.Context) (string, error) {
	// Check if provided via flag
	if dockerfilePath := cCtx.String(common.FileFlag.Name); dockerfilePath != "" {
		return dockerfilePath, nil
	}

	// Check if default Dockerfile exists
	if _, err := os.Stat("Dockerfile"); err != nil {
		// No Dockerfile found, return empty string (deploy existing image)
		return "", nil
	}

	// Interactive prompt when Dockerfile exists
	fmt.Println("\nFound Dockerfile in current directory.")

	options := []string{
		"Build and deploy from Dockerfile",
		"Deploy existing image from registry",
	}

	choice, err := output.SelectString("Choose deployment method:", options)
	if err != nil {
		return "", fmt.Errorf("failed to get deployment method choice: %w", err)
	}

	switch choice {
	case "Build and deploy from Dockerfile":
		return "Dockerfile", nil
	case "Deploy existing image from registry":
		return "", nil
	default:
		return "", fmt.Errorf("unexpected choice: %s", choice)
	}
}

// getAvailableRegistries returns a list of registries the user has authenticated to
func getAvailableRegistries() ([]registryInfo, error) {
	dockerCli, err := dockercommand.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker cli: %w", err)
	}

	allCreds, err := dockerCli.ConfigFile().GetAllCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	var registries []registryInfo
	for registry, auth := range allCreds {
		if auth.Username == "" {
			continue
		}

		info := registryInfo{
			URL:      registry,
			Username: auth.Username,
		}

		// Determine registry type
		switch {
		case strings.Contains(registry, "index.docker.io"):
			info.Type = "dockerhub"
		case strings.Contains(registry, "ghcr.io"):
			info.Type = "ghcr"
		case strings.Contains(registry, "gcr.io"):
			info.Type = "gcr"
		default:
			info.Type = "other"
		}

		// Skip access-token and refresh-token entries for Docker Hub
		if info.Type == "dockerhub" && (strings.Contains(registry, "access-token") || strings.Contains(registry, "refresh-token")) {
			continue
		}

		registries = append(registries, info)
	}

	// Sort registries with Docker Hub first
	sort.Slice(registries, func(i, j int) bool {
		if registries[i].Type == "dockerhub" {
			return true
		}
		if registries[j].Type == "dockerhub" {
			return false
		}
		return registries[i].Type < registries[j].Type
	})

	return registries, nil
}

// suggestImageReference generates an image reference suggestion based on registry and context
func suggestImageReference(registry registryInfo, imageName string, tag string) string {
	// Clean up image name for use in image reference
	imageName = strings.ToLower(imageName)
	imageName = strings.ReplaceAll(imageName, "_", "-")

	// Default to latest if no tag provided
	if tag == "" {
		tag = "latest"
	}

	switch registry.Type {
	case "dockerhub":
		return fmt.Sprintf("%s/%s:%s", registry.Username, imageName, tag)
	case "ghcr":
		return fmt.Sprintf("ghcr.io/%s/%s:%s", registry.Username, imageName, tag)
	case "gcr":
		// For GCR, username is typically the project ID
		return fmt.Sprintf("gcr.io/%s/%s:%s", registry.Username, imageName, tag)
	default:
		// For other registries, try to construct a reasonable default
		host := registry.URL
		if after, ok := strings.CutPrefix(host, "https://"); ok {
			host = after
		}
		if after, ok := strings.CutPrefix(host, "http://"); ok {
			host = after
		}
		host = strings.TrimSuffix(host, "/")
		return fmt.Sprintf("%s/%s/%s:%s", host, registry.Username, imageName, tag)
	}
}

// getDefaultAppName returns a default app name based on current directory
func getDefaultAppName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "myapp"
	}
	return filepath.Base(cwd)
}

// extractImageNameAndTag extracts the base image name and tag from an image reference
func extractImageNameAndTag(imageRef string) (imageName string, tag string) {
	// Remove registry prefix if present
	parts := strings.Split(imageRef, "/")
	if len(parts) > 1 {
		imageName = parts[len(parts)-1]
	} else {
		imageName = imageRef
	}

	// Split image and tag
	if strings.Contains(imageName, ":") {
		splitParts := strings.SplitN(imageName, ":", 2)
		return splitParts[0], splitParts[1]
	}
	return imageName, "latest"
}

// displayAuthenticationInstructions shows instructions for authenticating to registries
func displayAuthenticationInstructions() {
	fmt.Println("ℹ️  No authenticated Docker registries detected.")
	fmt.Println("   Make sure you're logged in to a registry:")
	fmt.Println("   • Docker Hub: docker login")
	fmt.Println("   • GitHub: docker login ghcr.io")
	fmt.Println("   • Google: docker login gcr.io")
	fmt.Println()
}

// displayRegistryExamples shows example image reference formats
func displayRegistryExamples(appName string) {
	if appName == "" {
		appName = "myapp"
	}
	fmt.Println("Examples:")
	fmt.Printf("  • username/%s:latest (Docker Hub)\n", appName)
	fmt.Printf("  • ghcr.io/username/%s:v1.0 (GitHub)\n", appName)
	fmt.Printf("  • gcr.io/project/%s:latest (Google)\n", appName)
}

// displayDetectedRegistries shows detected registries with examples
func displayDetectedRegistries(registries []registryInfo, appName string) {
	fmt.Println("Detected authenticated registries:")
	for _, reg := range registries {
		suggestion := suggestImageReference(reg, appName, "latest")
		switch reg.Type {
		case "dockerhub":
			fmt.Printf("  • Docker Hub (username: %s)\n", reg.Username)
			fmt.Printf("    Example: %s\n", suggestion)
		case "ghcr":
			fmt.Printf("  • GitHub Container Registry (username: %s)\n", reg.Username)
			fmt.Printf("    Example: %s\n", suggestion)
		case "gcr":
			fmt.Printf("  • Google Container Registry (project: %s)\n", reg.Username)
			fmt.Printf("    Example: %s\n", suggestion)
		default:
			fmt.Printf("  • %s (username: %s)\n", reg.URL, reg.Username)
			fmt.Printf("    Example: %s\n", suggestion)
		}
	}
	fmt.Println()
}

// findAvailableName finds an available name by appending numbers if needed
func findAvailableName(context, baseName string) string {
	apps, _ := common.ListApps(context)

	// Check if base name is available
	if _, exists := apps[baseName]; !exists {
		return baseName
	}

	// Try with incrementing numbers
	for i := 2; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", baseName, i)
		if _, exists := apps[candidate]; !exists {
			return candidate
		}
	}

	// Fallback to timestamp if somehow we have 100+ duplicates
	return fmt.Sprintf("%s-%d", baseName, time.Now().Unix())
}

// extractAppNameFromImage extracts the app name from an image reference
// Examples:
// - "my-app:latest" -> "my-app"
// - "docker.io/user/my-service:v1.2" -> "my-service"
// - "registry.com/org/project:tag" -> "project"
func extractAppNameFromImage(imageRef string) (string, error) {
	if strings.TrimSpace(imageRef) == "" {
		return "", fmt.Errorf("image reference cannot be empty")
	}

	// Remove tag (everything after last ":")
	if lastColon := strings.LastIndex(imageRef, ":"); lastColon != -1 {
		imageRef = imageRef[:lastColon]
	}

	// Get the last component after splitting by "/"
	parts := strings.Split(imageRef, "/")
	if len(parts) > 0 {
		appName := strings.TrimSpace(parts[len(parts)-1])
		if appName == "" {
			return "", fmt.Errorf("could not extract app name from image reference: %s", imageRef)
		}
		return appName, nil
	}

	return "", fmt.Errorf("invalid image reference format: %s", imageRef)
}

// validateImageReference validates that an image reference is not empty
func validateImageReference(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("image reference cannot be empty")
	}
	return nil
}

// validateFilePath validates that a file exists
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	return nil
}

// getStatusPriority returns a priority for sorting (lower = higher priority)
func getStatusPriority(status common.AppStatus, isExited bool) int {
	switch status {
	case common.ContractAppStatusStarted:
		if !isExited {
			return 1 // Running apps come first
		}
		return 2 // Exited apps come second
	case common.ContractAppStatusStopped:
		return 3
	case common.ContractAppStatusTerminated:
		return 4
	default:
		return 5
	}
}

// getStatusString returns a human-readable status string
func getStatusString(status common.AppStatus) string {
	switch status {
	case common.ContractAppStatusStarted:
		return "Running"
	case common.ContractAppStatusStopped:
		return "Stopped"
	case common.ContractAppStatusTerminated:
		return "Terminated"
	default:
		return "Unknown"
	}
}

// getExitedApps fetches API statuses to identify started apps that have exited
func getExitedApps(cCtx *cli.Context, apps []ethcommon.Address, appConfigs []AppController.IAppControllerAppConfig) map[string]bool {
	exitedApps := make(map[string]bool)

	var startedApps []ethcommon.Address
	for i, appAddr := range apps {
		if common.AppStatus(appConfigs[i].Status) == common.ContractAppStatusStarted {
			startedApps = append(startedApps, appAddr)
		}
	}

	if len(startedApps) > 0 {
		if userApiClient, err := NewUserApiClient(cCtx); err == nil {
			if statuses, err := userApiClient.GetStatuses(cCtx, startedApps); err == nil && statuses != nil {
				for i, appAddr := range startedApps {
					if i < len(statuses.Apps) && strings.EqualFold(statuses.Apps[i].Status, common.AppStatusExited) {
						exitedApps[appAddr.Hex()] = true
					}
				}
			}
		}
	}

	return exitedApps
}

// GetLogSettingsInteractive gets log redirection and visibility settings from flags or interactive prompt
func GetLogSettingsInteractive(cCtx *cli.Context) (logRedirect string, publicLogs bool, err error) {
	// Check if flag is provided
	if logVisibilityFlag := cCtx.String("log-visibility"); logVisibilityFlag != "" {
		switch logVisibilityFlag {
		case "public":
			return "always", true, nil
		case "private":
			return "always", false, nil
		case "off":
			return "", false, nil
		default:
			return "", false, fmt.Errorf("invalid --log-visibility value: %s (must be public, private, or off)", logVisibilityFlag)
		}
	}

	// Interactive prompt with three options
	options := []string{
		"Yes, but only viewable by me",
		"Yes, publicly viewable by anyone",
		"No, disable logs entirely",
	}

	choice, err := output.SelectString("Do you want to view your app's logs?", options)
	if err != nil {
		return "", false, fmt.Errorf("failed to get log viewing choice: %w", err)
	}

	switch choice {
	case "Yes, but only viewable by me":
		return "always", false, nil
	case "Yes, publicly viewable by anyone":
		return "always", true, nil
	case "No, disable logs entirely":
		return "", false, nil
	default:
		return "", false, fmt.Errorf("unexpected choice: %s", choice)
	}
}

// GetEnvironmentInteractive gets environment from args or interactive selection
func GetEnvironmentInteractive(cCtx *cli.Context, argIndex int) (string, error) {
	// Check if provided as argument
	if cCtx.Args().Len() > argIndex {
		return cCtx.Args().Get(argIndex), nil
	}

	// Interactive prompt
	fmt.Println("Select deployment environment:")

	// Build environment options with descriptions (reuse existing pattern from hooks.go)
	var options []string
	var envNames []string
	for name, config := range common.EnvironmentConfigs {
		description := GetEnvironmentDescription(name, config.Name, false)
		options = append(options, fmt.Sprintf("%s - %s", name, description))
		envNames = append(envNames, name)
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no deployment environments available")
	}

	selected, err := output.SelectString("Select environment:", options)
	if err != nil {
		return "", fmt.Errorf("failed to select environment: %w", err)
	}

	// Find the selected environment name
	for i, option := range options {
		if option == selected {
			return envNames[i], nil
		}
	}

	return "", fmt.Errorf("failed to find selected environment")
}

// ConfirmMainnetEnvironment shows a confirmation prompt for mainnet environments
func ConfirmMainnetEnvironment(env string) error {
	if !common.IsMainnetEnvironment(env) {
		return nil // Not mainnet, no confirmation needed
	}

	fmt.Println()
	fmt.Println("⚠️  WARNING: You selected", strings.ToUpper(env))
	fmt.Println("⚠️  This environment uses real funds")
	fmt.Println()

	confirmed, err := output.Confirm("Are you sure you want to use mainnet?")
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("mainnet selection cancelled")
	}

	return nil
}
