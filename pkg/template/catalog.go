package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// Default repository URL for templates
	DefaultTemplateRepo = "https://github.com/Layr-Labs/eigenx-templates"

	// Default version/branch for templates
	DefaultTemplateVersion = "main"

	// Default catalog URL in the eigenx-templates repository
	DefaultCatalogURL = "https://raw.githubusercontent.com/Layr-Labs/eigenx-templates/main/templates.json"

	// Cache duration for the catalog (15 minutes)
	CatalogCacheDuration = 15 * time.Minute
)

// TemplateEntry represents a single template in the catalog
type TemplateEntry struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

// TemplateCatalog represents the structure of templates.json
// Organized by language first, then by category (e.g., "typescript" -> "minimal")
type TemplateCatalog struct {
	Languages map[string]map[string]TemplateEntry `json:"-"`
	raw       map[string]interface{}
}

// GetTemplate finds a template by language and category
func (tc *TemplateCatalog) GetTemplate(category, language string) (*TemplateEntry, error) {
	templates, exists := tc.Languages[language]
	if !exists {
		return nil, fmt.Errorf("language %q not found in catalog", language)
	}

	template, exists := templates[category]
	if !exists {
		return nil, fmt.Errorf("category %q not found for language %q", category, language)
	}

	return &template, nil
}

// GetCategoryDescriptions returns a map of category names to their descriptions for a given language
func (tc *TemplateCatalog) GetCategoryDescriptions(language string) map[string]string {
	templates, exists := tc.Languages[language]
	if !exists {
		return nil
	}

	descriptions := make(map[string]string)
	for category, template := range templates {
		descriptions[category] = template.Description
	}
	return descriptions
}

// GetSupportedLanguages returns a list of all unique languages in the catalog
func (tc *TemplateCatalog) GetSupportedLanguages() []string {
	var languages []string
	for lang := range tc.Languages {
		languages = append(languages, lang)
	}
	return languages
}

// catalogCache holds the cached catalog and its expiration time
type catalogCache struct {
	catalog   *TemplateCatalog
	expiresAt time.Time
	mu        sync.RWMutex
}

var cache = &catalogCache{}

// FetchTemplateCatalog fetches and parses the template catalog from the remote URL
// It uses a 15-minute in-memory cache to avoid excessive network requests
// If EIGENX_USE_LOCAL_TEMPLATES is set, it looks for a local templates.json file
func FetchTemplateCatalog() (*TemplateCatalog, error) {
	// Check if using local templates
	if os.Getenv("EIGENX_USE_LOCAL_TEMPLATES") == "true" {
		return fetchLocalCatalog()
	}

	// Check cache first
	cache.mu.RLock()
	if cache.catalog != nil && time.Now().Before(cache.expiresAt) {
		defer cache.mu.RUnlock()
		return cache.catalog, nil
	}
	cache.mu.RUnlock()

	// Fetch from remote
	catalog, err := fetchRemoteCatalog(DefaultCatalogURL)
	if err != nil {
		return nil, err
	}

	// Update cache
	cache.mu.Lock()
	cache.catalog = catalog
	cache.expiresAt = time.Now().Add(CatalogCacheDuration)
	cache.mu.Unlock()

	return catalog, nil
}

// fetchRemoteCatalog fetches the catalog from a remote URL
func fetchRemoteCatalog(url string) (*TemplateCatalog, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch template catalog: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read template catalog: %w", err)
	}

	var catalog TemplateCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse template catalog: %w", err)
	}

	return &catalog, nil
}

// fetchLocalCatalog looks for a local templates.json file
func fetchLocalCatalog() (*TemplateCatalog, error) {
	// Look for EIGENX_TEMPLATES_PATH first
	templatesPath := os.Getenv("EIGENX_TEMPLATES_PATH")
	if templatesPath == "" {
		// Look for eigenx-templates directory as a sibling
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try sibling directory
		templatesPath = filepath.Join(filepath.Dir(cwd), "eigenx-templates")
		if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("local templates directory not found: %s", templatesPath)
		}
	}

	catalogPath := filepath.Join(templatesPath, "templates.json")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read local template catalog: %w", err)
	}

	var catalog TemplateCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse local template catalog: %w", err)
	}

	return &catalog, nil
}
