package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/eigenx-cli/config"
	"github.com/Layr-Labs/eigenx-cli/pkg/commands/utils"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/output"
	"github.com/Layr-Labs/eigenx-cli/pkg/template"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:      "create",
	Usage:     "Create new app project from template",
	ArgsUsage: "[name] [language] [template-name]",
	Flags: append(common.GlobalFlags, []cli.Flag{
		common.TemplateRepoFlag,
		common.TemplateVersionFlag,
	}...),
	Action: createAction,
}

var (
	primaryLanguages = []string{"typescript", "golang", "rust", "python"}

	shortNames = map[string]string{
		"ts": "typescript",
		"go": "golang",
		"rs": "rust",
		"py": "python",
	}
)

type projectConfig struct {
	name          string
	language      string
	templateName  string
	templateEntry *template.TemplateEntry
	repoURL       string
	ref           string
	subPath       string
}

func createAction(cCtx *cli.Context) error {
	cfg, err := gatherProjectConfig(cCtx)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(cfg.name); err == nil {
		return fmt.Errorf("directory %s already exists", cfg.name)
	}

	// Create project directory
	if err := os.MkdirAll(cfg.name, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", cfg.name, err)
	}

	if err := populateProjectFromTemplate(cCtx, cfg); err != nil {
		os.RemoveAll(cfg.name)
		return err
	}

	if cfg.subPath != "" {
		if err := postProcessTemplate(cfg.name, cfg.language, cfg.templateEntry); err != nil {
			return fmt.Errorf("failed to post-process template: %w", err)
		}
	}

	fmt.Printf("Successfully created %s project: %s\n", cfg.language, cfg.name)
	return nil
}

func gatherProjectConfig(cCtx *cli.Context) (*projectConfig, error) {
	cfg := &projectConfig{}

	// Get project name
	name := cCtx.Args().First()
	if name == "" {
		var err error
		name, err = output.InputString("Enter project name:", "", "", validateProjectName)
		if err != nil {
			return nil, fmt.Errorf("failed to get project name: %w", err)
		}
	}
	cfg.name = name

	// Handle custom template repo
	customTemplateRepo := cCtx.String(common.TemplateRepoFlag.Name)
	if customTemplateRepo != "" {
		cfg.repoURL = customTemplateRepo
		cfg.ref = cCtx.String(common.TemplateVersionFlag.Name)
		if cfg.ref == "" {
			cfg.ref = "main"
		}
		return cfg, nil
	}

	// Handle built-in templates
	language := cCtx.Args().Get(1)
	if language == "" {
		var err error
		language, err = output.SelectString("Select language:", primaryLanguages)
		if err != nil {
			return nil, fmt.Errorf("failed to select language: %w", err)
		}
	} else {
		// Resolve short names to full names
		if fullName, exists := shortNames[language]; exists {
			language = fullName
		}
	}
	cfg.language = language

	// Get template name
	templateName := cCtx.Args().Get(2)
	if templateName == "" {
		var err error
		templateName, err = utils.SelectTemplateInteractive(language)
		if err != nil {
			return nil, fmt.Errorf("failed to select template: %w", err)
		}
	}
	cfg.templateName = templateName

	// Resolve template details from catalog
	catalog, err := template.FetchTemplateCatalog()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template catalog: %w", err)
	}

	matchedTemplate, err := catalog.GetTemplate(templateName, language)
	if err != nil {
		return nil, err
	}

	cfg.templateEntry = matchedTemplate
	cfg.repoURL = template.DefaultTemplateRepo
	cfg.ref = template.DefaultTemplateVersion
	if versionFlag := cCtx.String(common.TemplateVersionFlag.Name); versionFlag != "" {
		cfg.ref = versionFlag
	}
	cfg.subPath = matchedTemplate.Path

	return cfg, nil
}

func populateProjectFromTemplate(cCtx *cli.Context, cfg *projectConfig) error {
	// Handle local templates for development
	if os.Getenv(template.EnvVarUseLocalTemplates) == "true" {
		eigenxTemplatesPath := os.Getenv(template.EnvVarTemplatesPath)
		if eigenxTemplatesPath == "" {
			// Look for eigenx-templates as a sibling directory
			for _, path := range []string{"eigenx-templates", "../eigenx-templates"} {
				if _, err := os.Stat(filepath.Join(path, "templates")); err == nil {
					eigenxTemplatesPath = path
					break
				}
			}
			if eigenxTemplatesPath == "" {
				return fmt.Errorf("cannot find eigenx-templates directory. Set %s or ensure eigenx-templates is a sibling directory", template.EnvVarTemplatesPath)
			}
		}

		localTemplatePath := filepath.Join(eigenxTemplatesPath, cfg.subPath)
		if _, err := os.Stat(localTemplatePath); os.IsNotExist(err) {
			return fmt.Errorf("local template not found at %s", localTemplatePath)
		}

		if err := copyDir(localTemplatePath, cfg.name); err != nil {
			return fmt.Errorf("failed to copy local template: %w", err)
		}

		contextLogger := common.LoggerFromContext(cCtx)
		contextLogger.Info("Using local template from %s", localTemplatePath)
		return nil
	}

	// Fetch from remote repository
	contextLogger := common.LoggerFromContext(cCtx)
	tracker := common.ProgressTrackerFromContext(cCtx.Context)

	fetcher := &template.GitFetcher{
		Client: template.NewGitClient(),
		Config: template.GitFetcherConfig{
			Verbose: cCtx.Bool("verbose"),
		},
		Logger: *logger.NewProgressLogger(contextLogger, tracker),
	}

	var err error
	if cfg.subPath != "" {
		err = fetcher.FetchSubdirectory(context.Background(), cfg.repoURL, cfg.ref, cfg.subPath, cfg.name)
	} else {
		err = fetcher.Fetch(context.Background(), cfg.repoURL, cfg.ref, cfg.name)
	}

	if err != nil {
		return fmt.Errorf("failed to create project from template: %w", err)
	}

	return nil
}

func postProcessTemplate(projectDir, language string, templateEntry *template.TemplateEntry) error {
	projectName := filepath.Base(projectDir)
	templateName := fmt.Sprintf("eigenx-tee-%s-app", language)

	if err := copyGitignore(projectDir); err != nil {
		return fmt.Errorf("failed to copy .gitignore: %w", err)
	}

	if err := copySharedTemplateFiles(projectDir); err != nil {
		return fmt.Errorf("failed to copy shared template files: %w", err)
	}

	// Get files to update from template metadata, fallback to just README.md
	filesToUpdate := templateEntry.PostProcess.ReplaceNameIn
	if len(filesToUpdate) == 0 {
		filesToUpdate = []string{"README.md"}
	}

	// Update all files specified in template metadata
	for _, filename := range filesToUpdate {
		if err := updateProjectFile(projectDir, filename, templateName, projectName); err != nil {
			return err
		}
	}

	return nil
}

func copySharedTemplateFiles(projectDir string) error {
	// Write .env.example
	envPath := filepath.Join(projectDir, ".env.example")
	if err := os.WriteFile(envPath, []byte(config.EnvExample), 0644); err != nil {
		return fmt.Errorf("failed to write .env.example: %w", err)
	}

	// Write or append README.md
	readmePath := filepath.Join(projectDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		// README.md exists, append the content
		file, err := os.OpenFile(readmePath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open README.md for appending: %w", err)
		}
		defer file.Close()

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

func copyGitignore(projectDir string) error {
	destPath := filepath.Join(projectDir, ".gitignore")

	// Skip if .gitignore already exists
	if _, err := os.Stat(destPath); err == nil {
		return nil
	}

	if err := os.WriteFile(destPath, []byte(config.GitIgnore), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

func updateProjectFile(projectDir, filename, oldString, newString string) error {
	filePath := filepath.Join(projectDir, filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	newContent := strings.ReplaceAll(string(content), oldString, newString)

	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update %s: %w", filename, err)
	}

	return nil
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.Contains(name, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

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
