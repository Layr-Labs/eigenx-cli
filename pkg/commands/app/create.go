package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/config"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/Layr-Labs/eigenx-cli/pkg/template"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:      "create",
	Usage:     "Create new app project from template",
	ArgsUsage: "[name] [language]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.TemplateFlag,
		common.TemplateVersionFlag,
	}...),
	Action: createAction,
}

// Language configuration
var primaryLanguages = []string{"typescript", "golang", "rust", "python"}

var shortNames = map[string]string{
	"ts": "typescript",
	"go": "golang",
	"rs": "rust",
	"py": "python",
}

var languageFiles = map[string][]string{
	"typescript": {"package.json"},
	"rust":       {"Cargo.toml", "Dockerfile"},
	"golang":     {"go.mod"},
}

func createAction(cCtx *cli.Context) error {
	// Get project name
	name := cCtx.Args().First()
	if name == "" {
		var err error
		name, err = output.InputString("Enter project name:", "", "", validateProjectName)
		if err != nil {
			return fmt.Errorf("failed to get project name: %w", err)
		}
	}

	// Check if directory exists
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %s already exists", name)
	}

	// Get language - only needed for built-in templates
	var language string
	if cCtx.String(common.TemplateFlag.Name) == "" {
		language = cCtx.Args().Get(1)
		if language == "" {
			var err error
			language, err = output.SelectString("Select language:", primaryLanguages)
			if err != nil {
				return fmt.Errorf("failed to select language: %w", err)
			}
		} else {
			// Resolve short names to full names
			if fullName, exists := shortNames[language]; exists {
				language = fullName
			}

			// Validate language is supported
			supported := slices.Contains(primaryLanguages, language)
			if !supported {
				return fmt.Errorf("unsupported language: %s", language)
			}
		}
	}

	// Resolve template URL and subdirectory path
	repoURL, ref, subPath, err := resolveTemplateSource(cCtx.String(common.TemplateFlag.Name), cCtx.String(common.TemplateVersionFlag.Name), language)
	if err != nil {
		return err
	}

	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", name, err)
	}

	// Setup GitFetcher
	contextLogger := common.LoggerFromContext(cCtx)
	tracker := common.ProgressTrackerFromContext(cCtx.Context)

	fetcher := &template.GitFetcher{
		Client: template.NewGitClient(),
		Config: template.GitFetcherConfig{
			Verbose: cCtx.Bool("verbose"),
		},
		Logger: *logger.NewProgressLogger(contextLogger, tracker),
	}

	// Check if we should use local templates (for development)
	if os.Getenv("EIGENX_USE_LOCAL_TEMPLATES") == "true" {
		// First try EIGENX_TEMPLATES_PATH env var, then look for the eigenx-templates directory as a sibling directory
		eigenxTemplatesPath := os.Getenv("EIGENX_TEMPLATES_PATH")
		if eigenxTemplatesPath == "" {
			// Look for eigenx-templates as a sibling directory
			for _, path := range []string{"eigenx-templates", "../eigenx-templates"} {
				if _, err := os.Stat(filepath.Join(path, "templates/minimal")); err == nil {
					eigenxTemplatesPath = path
					break
				}
			}
			if eigenxTemplatesPath == "" {
				return fmt.Errorf("cannot find eigenx-templates directory. Set EIGENX_TEMPLATES_PATH or ensure eigenx-templates is a sibling directory")
			}
		}

		// Use local templates from the eigenx-templates repository
		localTemplatePath := filepath.Join(eigenxTemplatesPath, "templates/minimal", language)
		if _, err := os.Stat(localTemplatePath); os.IsNotExist(err) {
			return fmt.Errorf("local template not found at %s", localTemplatePath)
		}

		// Copy local template to project directory
		err = copyDir(localTemplatePath, name)
		if err != nil {
			os.RemoveAll(name)
			return fmt.Errorf("failed to copy local template: %w", err)
		}
		contextLogger.Info("Using local template from %s", localTemplatePath)
	} else {
		if subPath != "" {
			// Fetch only the subdirectory, if one is specified
			err = fetcher.FetchSubdirectory(context.Background(), repoURL, ref, subPath, name)
		} else {
			// Fetch the full repository
			err = fetcher.Fetch(context.Background(), repoURL, ref, name)
		}
		if err != nil {
			// Cleanup on failure
			os.RemoveAll(name)
			return fmt.Errorf("failed to create project from template: %w", err)
		}
	}

	// Post-process only internal templates
	if subPath != "" {
		if err := postProcessTemplate(name, language); err != nil {
			return fmt.Errorf("failed to post-process template: %w", err)
		}
	}

	fmt.Printf("Successfully created %s project: %s\n", language, name)
	return nil
}

// validateProjectName validates that a project name is valid
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.Contains(name, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}
	return nil
}

// resolveTemplateSource determines the repository URL, ref, and subdirectory path for a template
func resolveTemplateSource(templateFlag, templateVersionFlag, language string) (string, string, string, error) {
	if templateFlag != "" {
		// Custom template URL provided via --template flag
		ref := templateVersionFlag
		if ref == "" {
			ref = "main"
		}
		return templateFlag, ref, "", nil
	}

	// Use template configuration system for defaults
	config, err := template.LoadConfig()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load template config: %w", err)
	}

	// Get template URL and version from config for "tee" framework
	templateURL, version, err := template.GetTemplateURLs(config, "tee", language)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get template URLs: %w", err)
	}

	// Override version if --template-version flag provided
	ref := version
	if templateVersionFlag != "" {
		ref = templateVersionFlag
	}

	// For templates from config, assume they follow our subdirectory structure
	subPath := fmt.Sprintf("templates/minimal/%s", language)
	return templateURL, ref, subPath, nil
}

// postProcessTemplate updates template files with project-specific values
func postProcessTemplate(projectDir, language string) error {
	projectName := filepath.Base(projectDir)
	templateName := fmt.Sprintf("eigenx-tee-%s-app", language)

	// Copy .gitignore from config directory
	if err := copyGitignore(projectDir); err != nil {
		return fmt.Errorf("failed to copy .gitignore: %w", err)
	}

	// Copy shared template files (.env.example)
	if err := copySharedTemplateFiles(projectDir); err != nil {
		return fmt.Errorf("failed to copy shared template files: %w", err)
	}

	// Update README.md title for all languages
	if err := updateProjectFile(projectDir, "README.md", templateName, projectName); err != nil {
		return err
	}

	// Update language-specific project files
	if filenames, exists := languageFiles[language]; exists {
		for _, filename := range filenames {
			if err := updateProjectFile(projectDir, filename, templateName, projectName); err != nil {
				return err
			}
		}
	}

	return nil
}

// copySharedTemplateFiles copies shared template files to the project directory
func copySharedTemplateFiles(projectDir string) error {
	// Write .env.example from embedded string
	envPath := filepath.Join(projectDir, ".env.example")
	if err := os.WriteFile(envPath, []byte(config.EnvExample), 0644); err != nil {
		return fmt.Errorf("failed to write .env.example: %w", err)
	}

	// Write or append README.md from embedded string
	readmePath := filepath.Join(projectDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		// README.md exists, append the content
		file, err := os.OpenFile(readmePath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open README.md for appending: %w", err)
		}
		defer file.Close()

		// Add newline before appending
		if _, err := file.WriteString("\n" + config.ReadMe); err != nil {
			return fmt.Errorf("failed to append to README.md: %w", err)
		}
	} else {
		// README.md doesn't exist, create it
		if err := os.WriteFile(readmePath, []byte(config.ReadMe), 0644); err != nil {
			return fmt.Errorf("failed to write README.md: %w", err)
		}
	}

	return nil
}

// copyGitignore copies the .gitignore from embedded config to the project directory if it doesn't exist
func copyGitignore(projectDir string) error {
	destPath := filepath.Join(projectDir, ".gitignore")

	// Check if .gitignore already exists
	if _, err := os.Stat(destPath); err == nil {
		return nil // File already exists, skip copying
	}

	// Use embedded config .gitignore
	err := os.WriteFile(destPath, []byte(config.GitIgnore), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// updateProjectFile updates a project file by replacing a specific string
func updateProjectFile(projectDir, filename, oldString, newString string) error {
	filePath := filepath.Join(projectDir, filename)

	// Read current file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Replace the specified string
	newContent := strings.ReplaceAll(string(content), oldString, newString)

	// Write back to file
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to update %s: %w", filename, err)
	}

	return nil
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
