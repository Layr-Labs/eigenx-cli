package utils

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	appcontrollerV2 "github.com/Layr-Labs/eigenx-contracts/pkg/bindings/v2/AppController"
	kmscrypto "github.com/Layr-Labs/eigenx-kms/pkg/crypto"
	kmstypes "github.com/Layr-Labs/eigenx-kms/pkg/types"
	"github.com/docker/docker/client"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-envparse"
	"github.com/urfave/cli/v2"
)

// ============================================================================
// Release and Environment Processing
// ============================================================================

// PrepareReleaseFromContext prepares a release with separated Dockerfile handling
// The dockerfile path and env file path are provided as parameters (already collected earlier)
// maxPushRetries controls how many times to retry on push permission errors (0 = no retries)
func PrepareReleaseFromContext(cCtx *cli.Context, environmentConfig *common.EnvironmentConfig, appID gethcommon.Address, dockerfilePath string, imageRef string, envFilePath string, logRedirect string, instanceType string, maxPushRetries int) (appcontrollerV2.IAppControllerRelease, string, error) {
	logger := common.LoggerFromContext(cCtx)

	// Create operation closures that capture context
	buildAndPush := func(ref string) (string, error) {
		return buildAndPushLayeredImage(cCtx, *environmentConfig, dockerfilePath, ref, logRedirect, envFilePath)
	}

	layerRemoteImage := func(ref string) (string, error) {
		return layerRemoteImageIfNeeded(cCtx, *environmentConfig, ref, logRedirect, envFilePath)
	}

	// Ensure image is compatible with EigenX (either build from Dockerfile or layer existing image)
	var err error
	if dockerfilePath != "" {
		// Build and push with retry logic for permission errors
		imageRef, err = retryImagePushOperation(cCtx, maxPushRetries, "build and push", buildAndPush, imageRef)
		if err != nil {
			return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to build and push layered image: %w", err)
		}

		// Wait for registry propagation
		logger.Info("Waiting %d seconds for registry propagation...", RegistryPropagationWaitSeconds)
		time.Sleep(RegistryPropagationWaitSeconds * time.Second)
	} else {
		// Layer remote image if needed, with retry logic for permission errors
		imageRef, err = retryImagePushOperation(cCtx, maxPushRetries, "layer published image", layerRemoteImage, imageRef)
		if err != nil {
			return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to ensure image compatibility: %w", err)
		}
	}

	digest, name, err := getImageDigestAndName(cCtx.Context, imageRef)
	if err != nil {
		return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to get image digest and name: %w", err)
	}

	fmt.Println()
	logger.Info("Name: %s", name)
	logger.Info("Image digest: %s", hex.EncodeToString(digest[:]))

	var publicEnv, privateEnv map[string]string
	if envFilePath == "" {
		logger.Info("Continuing without environment file")
		publicEnv, privateEnv = make(map[string]string), make(map[string]string)
	} else {
		publicEnv, privateEnv, err = parseAndValidateEnvFile(cCtx, envFilePath)
		if err != nil {
			return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to parse and validate env file: %w", err)
		}
	}

	// Inject instance type selection into public environment variables
	// This overrides any value in .env file if present
	publicEnv[common.EigenMachineTypeEnvVar] = instanceType
	logger.Info("Instance type: %s", instanceType)

	publicEnvBytes, err := json.Marshal(publicEnv)
	if err != nil {
		return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to marshal public env: %w", err)
	}
	privateEnvBytes, err := json.Marshal(privateEnv)
	if err != nil {
		return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to marshal private env: %w", err)
	}

	encryptionKey, _, err := getKMSKeysForEnvironment(environmentConfig.Name)
	if err != nil {
		return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to get encryption key: %w", err)
	}

	protectedHeaders := kmscrypto.GetAppProtectedHeaders(appID.Hex())
	encryptedEnvStr, err := kmscrypto.EncryptRSAOAEPAndAES256GCMWithPEM(encryptionKey, privateEnvBytes, protectedHeaders)
	if err != nil {
		return appcontrollerV2.IAppControllerRelease{}, imageRef, fmt.Errorf("failed to encrypt env: %w", err)
	}

	release := appcontrollerV2.IAppControllerRelease{
		RmsRelease: appcontrollerV2.IReleaseManagerTypesRelease{
			Artifacts: []appcontrollerV2.IReleaseManagerTypesArtifact{
				{
					Digest:   digest,
					Registry: name,
				},
			},
			UpgradeByTime: uint32(time.Now().Unix() + 3600),
		},
		PublicEnv:    publicEnvBytes,
		EncryptedEnv: []byte(encryptedEnvStr),
	}

	return release, imageRef, nil
}

// retryImagePushOperation wraps an image push operation with retry logic for permission errors
func retryImagePushOperation(
	cCtx *cli.Context,
	maxRetries int,
	operationName string,
	operation func(imageRef string) (string, error),
	initialImageRef string,
) (string, error) {
	logger := common.LoggerFromContext(cCtx)
	imageRef := initialImageRef
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		imageRef, err = operation(imageRef)

		// If no error or not a permission error, break out of retry loop
		if err == nil || !IsPushPermissionError(err) {
			break
		}

		// Permission error detected - offer to retry with different registry
		if attempt < maxRetries {
			fmt.Println()
			logger.Warn("Push failed during %s due to permission issues with the registry.", operationName)
			logger.Info("This may be because:")
			logger.Info("  • Your authentication token has expired or is invalid")
			logger.Info("  • Your token lacks the required permissions (e.g., 'write:packages' for GitHub)")
			logger.Info("  • The registry requires additional authentication steps")
			fmt.Println()

			// Ask if they want to try a different registry
			retry, retryErr := output.Confirm("Would you like to try a different registry?")
			if retryErr != nil || !retry {
				break
			}

			// Get a new image reference
			logger.Info("\nPlease select a different registry or re-authenticate and try again.")
			newImageRef, imgErr := GetImageReferenceInteractive(cCtx, 0, true)
			if imgErr != nil {
				logger.Warn("Failed to get new image reference: %v", imgErr)
				break
			}

			// Update imageRef for retry
			imageRef = newImageRef
			logger.Info("Retrying with new registry: %s\n", imageRef)
		}
	}

	return imageRef, err
}

func layerRemoteImageIfNeeded(cCtx *cli.Context, environmentConfig common.EnvironmentConfig, imageRef, logRedirect, envFilePath string) (string, error) {
	// Check if the provided image is missing image layering, which is required for EigenX
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}
	defer dockerClient.Close()

	alreadyLayered, err := checkIfImageAlreadyLayeredForEigenX(dockerClient, cCtx.Context, imageRef)
	if err != nil {
		return "", fmt.Errorf("failed to check if image needs layering: %w", err)
	}

	if !alreadyLayered {
		logger := common.LoggerFromContext(cCtx)

		// Prompt for target image to avoid overwriting the source
		targetImageRef, err := GetLayeredTargetImageInteractive(cCtx, imageRef)
		if err != nil {
			return "", fmt.Errorf("failed to get target image reference: %w", err)
		}

		logger.Info("Adding EigenX components to create %s from %s...", targetImageRef, imageRef)
		layeredImageRef, err := layerLocalImage(cCtx, dockerClient, environmentConfig, imageRef, targetImageRef, logRedirect, envFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to layer published image: %w", err)
		}
		imageRef = layeredImageRef

		// Wait for registry propagation
		logger.Info("Waiting %d seconds for registry propagation...", RegistryPropagationWaitSeconds)
		time.Sleep(RegistryPropagationWaitSeconds * time.Second)
	}

	return imageRef, nil
}

// ============================================================================
// Image Registry Operations
// ============================================================================

// Platform represents a container platform
type Platform struct {
	OS   string
	Arch string
}

// String returns the platform in standard format
func (p Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// IsLinuxAMD64 checks if platform matches required linux/amd64
func (p Platform) IsLinuxAMD64() bool {
	return p.OS == LinuxOS && p.Arch == AMD64Arch
}

// imageDigestResult holds the result of image digest extraction
type imageDigestResult struct {
	digest    [32]byte
	name      string
	platforms []Platform
}

// extractDigestFromMultiPlatform extracts digest from multi-platform image index
func extractDigestFromMultiPlatform(idx v1.ImageIndex, ref name.Reference) (*imageDigestResult, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get image manifest: %w", err)
	}

	var platforms []Platform
	for _, m := range manifest.Manifests {
		if m.Platform != nil {
			platform := Platform{OS: m.Platform.OS, Arch: m.Platform.Architecture}
			platforms = append(platforms, platform)

			if platform.IsLinuxAMD64() {
				digest, err := hexStringToBytes32(m.Digest.Hex)
				if err != nil {
					return nil, fmt.Errorf("failed to decode digest %s: %w", m.Digest.Hex, err)
				}
				return &imageDigestResult{
					digest:    digest,
					name:      ref.Context().Name(),
					platforms: platforms,
				}, nil
			}
		}
	}

	return &imageDigestResult{platforms: platforms}, nil
}

// extractDigestFromSinglePlatform extracts digest from single-platform image
func extractDigestFromSinglePlatform(img v1.Image, ref name.Reference) (*imageDigestResult, error) {
	config, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	platform := Platform{OS: config.OS, Arch: config.Architecture}
	platforms := []Platform{platform}

	if platform.IsLinuxAMD64() {
		digestHash, err := img.Digest()
		if err != nil {
			return nil, fmt.Errorf("failed to get image digest: %w", err)
		}
		digest, err := hexStringToBytes32(digestHash.Hex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode digest %s: %w", digestHash.Hex, err)
		}
		return &imageDigestResult{
			digest:    digest,
			name:      ref.Context().Name(),
			platforms: platforms,
		}, nil
	}

	return &imageDigestResult{platforms: platforms}, nil
}

// createPlatformErrorMessage creates a detailed error message for platform mismatch
func createPlatformErrorMessage(imageRef string, platforms []Platform) error {
	platformStrs := make([]string, len(platforms))
	for i, p := range platforms {
		platformStrs[i] = p.String()
	}

	errorMsg := fmt.Sprintf(`EigenX requires linux/amd64 images for TEE deployment.

Image: %s
Found platform(s): %s
Required platform: linux/amd64

To fix this issue:
1. Manual fix:
   a. Rebuild your image with the correct platform:
      docker build --platform linux/amd64 -t %s .
   b. Push the rebuilt image to your remote registry:
      docker push %s

2. Or use eigenx to build with the correct platform automatically:
   eigenx app deploy --dockerfile /path/to/Dockerfile

   (Or run 'eigenx app deploy' from the directory containing your Dockerfile)

The --platform linux/amd64 flag ensures your image works in EigenX's TEE environment.`,
		imageRef,
		strings.Join(platformStrs, ", "),
		imageRef,
		imageRef)

	return fmt.Errorf("%s", errorMsg)
}

func getImageDigestAndName(ctx context.Context, imageRef string) ([32]byte, string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return [32]byte{}, "", fmt.Errorf("failed to parse image reference %s: %w", imageRef, err)
	}

	desc, err := remote.Get(ref, remote.WithContext(ctx))
	if err != nil {
		return [32]byte{}, "", fmt.Errorf("failed to get image %s: %w", imageRef, err)
	}

	var result *imageDigestResult

	if desc.MediaType.IsIndex() {
		idx, err := desc.ImageIndex()
		if err != nil {
			return [32]byte{}, "", fmt.Errorf("failed to get image index %s: %w", imageRef, err)
		}

		result, err = extractDigestFromMultiPlatform(idx, ref)
		if err != nil {
			return [32]byte{}, "", fmt.Errorf("failed to process multi-platform image %s: %w", imageRef, err)
		}
	} else {
		img, err := desc.Image()
		if err != nil {
			return [32]byte{}, "", fmt.Errorf("failed to get image %s: %w", imageRef, err)
		}

		result, err = extractDigestFromSinglePlatform(img, ref)
		if err != nil {
			return [32]byte{}, "", fmt.Errorf("failed to process single-platform image %s: %w", imageRef, err)
		}
	}

	// If we found a compatible platform, return success
	if result.name != "" {
		return result.digest, result.name, nil
	}

	// No compatible platform found, return helpful error
	return [32]byte{}, "", createPlatformErrorMessage(imageRef, result.platforms)
}

func hexStringToBytes32(hexStr string) ([32]byte, error) {
	var result [32]byte

	// Remove "sha256:" prefix if present
	hexStr = strings.TrimPrefix(hexStr, SHA256Prefix)

	// Decode hex string
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, fmt.Errorf("failed to decode hex string: %w", err)
	}

	// Ensure we have exactly 32 bytes
	if len(bytes) != 32 {
		return result, fmt.Errorf("digest must be exactly 32 bytes, got %d", len(bytes))
	}

	copy(result[:], bytes)
	return result, nil
}

// ============================================================================
// Environment and Configuration
// ============================================================================

func parseAndValidateEnvFile(cCtx *cli.Context, envFilePath string) (kmstypes.Env, kmstypes.Env, error) {
	logger := common.LoggerFromContext(cCtx)

	file, err := os.Open(envFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open env file %s: %w", envFilePath, err)
	}
	defer file.Close()

	publicEnv := kmstypes.Env{}
	privateEnv := kmstypes.Env{}
	mnemonicFiltered := false

	envVars, err := envparse.Parse(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse env file %s: %w", envFilePath, err)
	}

	for varName, value := range envVars {
		// Filter out mnemonic variables
		if strings.ToUpper(varName) == common.MnemonicEnvVar {
			mnemonicFiltered = true
			continue
		}

		if strings.HasSuffix(varName, "_PUBLIC") {
			publicEnv[varName] = value
		} else {
			privateEnv[varName] = value
		}
	}

	logger.Info("Your container will deploy with the following environment variables:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "\n")

	// Print filtered mnemonic variables
	if mnemonicFiltered {
		fmt.Fprintf(w, "\033[3;36mMnemonic environment variable removed to be overridden by protocol provided mnemonic\033[0m\n")
		fmt.Fprintf(w, "\n")
	}

	// Print public variables
	if len(publicEnv) != 0 {
		fmt.Fprintf(w, "PUBLIC VARIABLE\tVALUE\n")
		fmt.Fprintf(w, "---------------\t-----\n")

		for k, v := range publicEnv {
			fmt.Fprintf(w, "%s\t%s\n", k, v)
		}
	} else {
		fmt.Fprintf(w, "No public variables found\n")
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "-----------------------------------------\n")
	fmt.Fprintf(w, "\n")

	// Print private variables
	if len(privateEnv) != 0 {
		fmt.Fprintf(w, "PRIVATE VARIABLE\tVALUE\n")
		fmt.Fprintf(w, "----------------\t-----\n")

		for k, v := range privateEnv {
			fmt.Fprintf(w, "%s\t%s\n", k, v)
		}
	} else {
		fmt.Fprintf(w, "No private variables found\n")
	}
	fmt.Fprintf(w, "\n")

	w.Flush()

	confirmed, err := output.ConfirmWithDefault("Is this categorization correct? Public variables will be in plaintext onchain. Private variables will be encrypted onchain.", false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		return nil, nil, fmt.Errorf("user rejected variable categorization")
	}

	return publicEnv, privateEnv, nil
}
