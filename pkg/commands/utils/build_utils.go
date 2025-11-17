package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	project "github.com/Layr-Labs/eigenx-cli"
	"github.com/Layr-Labs/eigenx-cli/internal/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	dockercommand "github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// ============================================================================
// Error Types
// ============================================================================

// PushPermissionError indicates a push failed due to authentication/permission issues
type PushPermissionError struct {
	ImageRef string
	Err      error
}

func (e *PushPermissionError) Error() string {
	return fmt.Sprintf("permission denied pushing to %s: %v", e.ImageRef, e.Err)
}

func (e *PushPermissionError) Unwrap() error {
	return e.Err
}

// IsPushPermissionError checks if an error is a push permission error
func IsPushPermissionError(err error) bool {
	var pushErr *PushPermissionError
	return errors.As(err, &pushErr)
}

// ============================================================================
// Template Processing Utilities
// ============================================================================

func processTemplate(templatePath string, data any) ([]byte, error) {
	tmpl, err := template.ParseFS(project.TemplatesFS, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.Bytes(), nil
}

func setupLayeredBuildDirectory(environmentConfig common.EnvironmentConfig, layeredDockerfileContent []byte, scriptContent []byte, includeTLS bool) (string, error) {
	tempDir, err := common.CreateTempDir(LayeredBuildDirPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write layered Dockerfile
	layeredDockerfilePath := filepath.Join(tempDir, LayeredDockerfileName)
	err = os.WriteFile(layeredDockerfilePath, layeredDockerfileContent, 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write layered dockerfile: %w", err)
	}

	// Write wrapper script
	scriptPath := filepath.Join(tempDir, EnvSourceScriptName)
	err = os.WriteFile(scriptPath, scriptContent, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// Copy KMS keys
	_, signingKey, err := getKMSKeysForEnvironment(environmentConfig.Name)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to get keys for environment %s: %w", environmentConfig.Name, err)
	}

	signingKeyPath := filepath.Join(tempDir, KMSSigningKeyName)
	err = os.WriteFile(signingKeyPath, signingKey, 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write signing key: %w", err)
	}

	// Copy kms-client binary
	kmsClientPath := filepath.Join(tempDir, KMSClientBinaryName)
	err = os.WriteFile(kmsClientPath, project.RawKmsClient, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write kms-client binary: %w", err)
	}

	// Only include TLS components if requested
	if includeTLS {
		// Copy tls-keygen binary
		tlsKeygenPath := filepath.Join(tempDir, TlsKeygenBinaryName)
		err = os.WriteFile(tlsKeygenPath, project.RawTlsKeygenBinary, 0755)
		if err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to write tls-keygen binary: %w", err)
		}

		// Handle Caddyfile - check if user has one in current directory
		userCaddyfileContent, err := os.ReadFile(CaddyfileName)
		if err == nil {
			// User has a Caddyfile, use it
			caddyfilePath := filepath.Join(tempDir, CaddyfileName)
			err = os.WriteFile(caddyfilePath, userCaddyfileContent, 0644)
			if err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to write user Caddyfile: %w", err)
			}
		} else if os.IsNotExist(err) {
			// No user Caddyfile but TLS is requested
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("TLS is enabled (DOMAIN is set) but Caddyfile not found. Run 'eigenx app configure tls' to set up TLS configuration")
		} else {
			// Unexpected error reading Caddyfile
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to check for user Caddyfile: %w", err)
		}
	}

	return tempDir, nil
}

func extractImageConfig(dockerClient *client.Client, ctx context.Context, imageTag string) ([]string, string, error) {
	inspectResp, err := dockerClient.ImageInspect(ctx, imageTag)
	if err != nil {
		return nil, "", fmt.Errorf("failed to inspect base image: %w", err)
	}

	originalCmd := inspectResp.Config.Cmd
	if len(originalCmd) == 0 && len(inspectResp.Config.Entrypoint) > 0 {
		originalCmd = inspectResp.Config.Entrypoint
	}

	return originalCmd, inspectResp.Config.User, nil
}

// extractDigestFromRepoDigest extracts the sha256 digest from a Docker repo digest string
// Format: "repo@sha256:xxxxx" -> returns [32]byte digest
func extractDigestFromRepoDigest(repoDigest string) *[32]byte {
	const prefix = "@sha256:"
	idx := strings.LastIndex(repoDigest, prefix)
	if idx == -1 {
		return nil
	}

	hexDigest := repoDigest[idx+len(prefix):]
	digest, err := hexStringToBytes32(hexDigest)
	if err != nil {
		return nil
	}

	return &digest
}

func checkIfImageAlreadyLayeredForEigenX(dockerClient *client.Client, ctx context.Context, imageRef string) (bool, error) {
	// First get the remote image digest to ensure we're working with the latest
	// This also validates that the image exists and supports linux/amd64 platform
	remoteDigest, _, err := getImageDigestAndName(ctx, imageRef)
	if err != nil {
		return false, err
	}

	// Try to inspect the image locally
	var inspectResp image.InspectResponse
	needToPull := true

	inspectResp, err = dockerClient.ImageInspect(ctx, imageRef)
	if err == nil {
		// Image exists locally - check if it matches the remote digest
		// Docker stores digests in RepoDigests field as "repo@sha256:xxx"
		for _, repoDigest := range inspectResp.RepoDigests {
			if localDigest := extractDigestFromRepoDigest(repoDigest); localDigest != nil {
				if *localDigest == remoteDigest {
					// Local image matches registry version, no need to pull
					needToPull = false
					break
				}
			}
		}
	}

	if needToPull {
		// Pull the image with the required platform
		resp, pullErr := dockerClient.ImagePull(ctx, imageRef, image.PullOptions{
			Platform: DockerPlatform, // linux/amd64
		})
		if pullErr != nil {
			return false, fmt.Errorf("failed to pull image %s for platform %s: %w", imageRef, DockerPlatform, pullErr)
		}
		defer resp.Close()

		// Must read the response to ensure the pull completes
		if _, err := io.Copy(io.Discard, resp); err != nil {
			return false, fmt.Errorf("failed to complete image pull for %s: %w", imageRef, err)
		}

		// Try inspection again after pulling
		inspectResp, err = dockerClient.ImageInspect(ctx, imageRef)
		if err != nil {
			return false, fmt.Errorf("failed to inspect image %s after pulling: %w", imageRef, err)
		}
	}

	// Check if image already has EigenX layering by looking for our entrypoint
	_, alreadyLayered := inspectResp.Config.Labels["eigenx_cli_version"]
	return alreadyLayered, nil
}

// ============================================================================
// Image Building and Pushing
// ============================================================================

func buildAndPushLayeredImage(cCtx *cli.Context, environmentConfig common.EnvironmentConfig, dockerfilePath, targetImageRef, logRedirect, envFilePath string) (string, error) {
	logger := common.LoggerFromContext(cCtx)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}
	defer dockerClient.Close()

	// Build base image from user's Dockerfile
	baseImageTag := fmt.Sprintf("%s%s", TempImagePrefix, strings.ToLower(dockerfilePath))
	logger.Info("Building base image from %s...", dockerfilePath)

	err = buildDockerImage(".", dockerfilePath, baseImageTag)
	if err != nil {
		return "", fmt.Errorf("failed to build base image: %w", err)
	}

	return layerLocalImage(cCtx, dockerClient, environmentConfig, baseImageTag, targetImageRef, logRedirect, envFilePath)
}

func layerLocalImage(cCtx *cli.Context, dockerClient *client.Client, environmentConfig common.EnvironmentConfig, sourceImageRef, targetImageRef, logRedirect, envFilePath string) (string, error) {
	logger := common.LoggerFromContext(cCtx)

	// Extract original command and user from source image
	originalCmd, originalUser, err := extractImageConfig(dockerClient, cCtx.Context, sourceImageRef)
	if err != nil {
		return "", fmt.Errorf("failed to extract image config: %w", err)
	}

	// Check if user has DOMAIN configured in env file
	includeTLS := false
	if _, err := os.Stat(envFilePath); err == nil {
		// Parse env file using godotenv
		envMap, err := godotenv.Read(envFilePath)
		if err == nil {
			if domain, exists := envMap["DOMAIN"]; exists && domain != "" && domain != "localhost" {
				includeTLS = true
				logger.Debug("Found DOMAIN=%s in %s, including TLS components", domain, envFilePath)
			}
		}
	}
	logger.Debug("Adding EigenX components to %s (TLS disabled for published images)", sourceImageRef)

	// Generate template content
	originalCmdStr, err := formatCmdForDockerfile(originalCmd)
	if err != nil {
		return "", fmt.Errorf("failed to format original command: %w", err)
	}

	layeredDockerfileContent, err := processTemplate(LayeredDockerfilePath, LayeredDockerfileTemplateData{
		BaseImage:        sourceImageRef,
		OriginalCmd:      originalCmdStr,
		OriginalUser:     originalUser,
		LogRedirect:      logRedirect,
		IncludeTLS:       includeTLS,
		EigenXCLIVersion: version.GetVersion(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to process dockerfile template: %w", err)
	}

	scriptContent, err := processTemplate(EnvSourceScriptTemplatePath, EnvSourceScriptTemplateData{
		KMSServerURL: environmentConfig.KMSServerURL,
		UserAPIURL:   environmentConfig.UserApiServerURL,
	})
	if err != nil {
		return "", fmt.Errorf("failed to process script template: %w", err)
	}

	// Setup build directory with all required files
	tempDir, err := setupLayeredBuildDirectory(environmentConfig, layeredDockerfileContent, scriptContent, includeTLS)
	if err != nil {
		return "", fmt.Errorf("failed to setup build directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Build layered image
	logger.Info("Building updated image with EigenX components for %s...", sourceImageRef)
	layeredDockerfilePath := filepath.Join(tempDir, LayeredDockerfileName)
	err = buildDockerImage(tempDir, layeredDockerfilePath, targetImageRef)
	if err != nil {
		return "", fmt.Errorf("failed to build layered image: %w", err)
	}

	// Push to registry
	logger.Info("Publishing updated image to %s...", targetImageRef)
	err = pushDockerImage(dockerClient, targetImageRef)
	if err != nil {
		return "", fmt.Errorf("failed to push layered image: %w", err)
	}

	logger.Info("Successfully published updated image: %s", targetImageRef)
	return targetImageRef, nil
}

// ============================================================================
// Docker Operations
// ============================================================================

func buildDockerImage(buildContext, dockerfilePath, tag string) error {
	cmd := exec.Command("docker", "buildx", "build",
		"--platform", DockerPlatform,
		"-t", tag,
		"-f", dockerfilePath,
		"--load",
		"--progress=plain",
		buildContext,
	)

	// Inherit stdout and stderr for real-time output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("buildx command failed: %w", err)
	}

	return nil
}

func pushDockerImage(dockerClient *client.Client, imageRef string) error {
	ctx := context.Background()

	// Use empty auth config - Docker client will use system auth
	dockerCli, err := dockercommand.NewDockerCli()
	if err != nil {
		return fmt.Errorf("failed to create docker cli: %w", err)
	}
	encodedAuth, err := dockercommand.RetrieveAuthTokenFromImage(dockerCli.ConfigFile(), imageRef)
	if err != nil {
		return fmt.Errorf("failed to retrieve auth token: %w", err)
	}

	resp, err := dockerClient.ImagePush(ctx, imageRef, image.PushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		// Check if this is a permission error
		if isPermissionError(err.Error()) {
			return &PushPermissionError{
				ImageRef: imageRef,
				Err:      err,
			}
		}
		return err
	}
	defer resp.Close()

	// Parse and display push output
	err = parseBuildOutput(resp)
	if err != nil {
		// Check if the error from parsing output is a permission error
		if isPermissionError(err.Error()) {
			return &PushPermissionError{
				ImageRef: imageRef,
				Err:      err,
			}
		}
	}
	return err
}

func parseBuildOutput(output io.Reader) error {
	scanner := bufio.NewScanner(output)
	layerStatus := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// If not JSON, print as-is
			fmt.Println(line)
			continue
		}

		// Extract and print stream content (build steps)
		if stream, ok := msg["stream"].(string); ok {
			if stream != "" {
				fmt.Print(stream)
			}
		}

		// Extract and print status content (layer pushes) - deduplicate
		if status, ok := msg["status"].(string); ok {
			if id, hasID := msg["id"].(string); hasID {
				// Only print if status changed for this layer
				if layerStatus[id] != status {
					layerStatus[id] = status
					fmt.Printf("%s: %s\n", id, status)
				}
			} else {
				fmt.Println(status)
			}
		}

		// Handle errors
		if errorMsg, ok := msg["error"].(string); ok {
			return fmt.Errorf("build error: %s", errorMsg)
		}
	}

	return scanner.Err()
}

// isPermissionError checks if an error message indicates a permission/auth issue
func isPermissionError(errMsg string) bool {
	errLower := strings.ToLower(errMsg)
	permissionKeywords := []string{
		"denied",
		"unauthorized",
		"forbidden",
		"insufficient_scope",
		"authentication required",
		"access forbidden",
		"permission denied",
		"requested access to the resource is denied",
	}

	for _, keyword := range permissionKeywords {
		if strings.Contains(errLower, keyword) {
			return true
		}
	}
	return false
}

func formatCmdForDockerfile(cmd []string) (string, error) {
	if len(cmd) == 0 {
		return `[""]`, nil
	}

	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Sprintf(`["%s"]`, cmd[0]), err
	}
	return string(jsonBytes), nil
}
