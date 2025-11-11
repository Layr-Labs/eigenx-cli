package utils

import (
	"fmt"
	"html"
	"image"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	MaxImageSize         = 4 * 1024 * 1024 // 4MB
	MaxAppNameLength     = 100
	MaxDescriptionLength = 1000
	BytesPerMB           = 1024 * 1024
)

var (
	ValidImageExtensions = []string{".jpg", ".jpeg", ".png"}
	ValidXHosts          = []string{"twitter.com", "www.twitter.com", "x.com", "www.x.com"}
)

// AppProfile represents the profile information for an app
type AppProfile struct {
	Name        string `json:"name"`
	Website     string `json:"website"`
	Description string `json:"description"`
	XURL        string `json:"xURL"`
	ImageURL    string `json:"imageURL"`
}

// ImageInfo contains validated image metadata
type ImageInfo struct {
	Width  int
	Height int
	SizeKB float64
	Format string
}

// IsSquare checks if image has approximately square aspect ratio
func (img *ImageInfo) IsSquare() bool {
	if img.Width == 0 || img.Height == 0 {
		return false
	}
	aspectRatio := float64(img.Width) / float64(img.Height)
	return aspectRatio >= 0.8 && aspectRatio <= 1.25
}

// AspectRatio returns the width/height ratio
func (img *ImageInfo) AspectRatio() float64 {
	if img.Height == 0 {
		return 0
	}
	return float64(img.Width) / float64(img.Height)
}

// ValidateURL validates that a string is a valid URL
func ValidateURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}

	return nil
}

// ValidateXURL validates that a URL is a valid X (Twitter) URL
func ValidateXURL(rawURL string) error {
	if err := ValidateURL(rawURL); err != nil {
		return err
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	host := strings.ToLower(parsedURL.Host)

	// Accept twitter.com and x.com domains
	if !slices.Contains(ValidXHosts, host) {
		return fmt.Errorf("URL must be a valid X/Twitter URL (x.com or twitter.com)")
	}

	// Ensure it has a path (username/profile)
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		return fmt.Errorf("x URL must include a username or profile path")
	}

	return nil
}

// ValidateAndGetImageInfo validates and extracts image information in one pass
func ValidateAndGetImageInfo(filePath string) (*ImageInfo, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("image path cannot be empty")
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("image file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to access image file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Check file size
	if info.Size() > MaxImageSize {
		sizeMB := float64(info.Size()) / float64(BytesPerMB)
		return nil, fmt.Errorf("image file size (%.2f MB) exceeds maximum allowed size of %d MB",
			sizeMB, MaxImageSize/BytesPerMB)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !slices.Contains(ValidImageExtensions, ext) {
		return nil, fmt.Errorf("image must be JPG or PNG format (found: %s)", ext)
	}

	// Open and decode image (validates format and gets dimensions)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("invalid or corrupted image file: %w", err)
	}

	return &ImageInfo{
		Width:  cfg.Width,
		Height: cfg.Height,
		SizeKB: float64(info.Size()) / 1024,
		Format: format,
	}, nil
}

// ValidateAppName validates an app name
func ValidateAppName(name string) error {
	if err := validateNotEmpty(name, "name"); err != nil {
		return err
	}

	if len(name) > MaxAppNameLength {
		return fmt.Errorf("name cannot exceed %d characters", MaxAppNameLength)
	}

	return nil
}

// ValidateAppDescription validates an app description
func ValidateAppDescription(description string) error {
	if err := validateNotEmpty(description, "description"); err != nil {
		return err
	}

	if len(description) > MaxDescriptionLength {
		return fmt.Errorf("description cannot exceed %d characters", MaxDescriptionLength)
	}

	return nil
}

// SanitizeString sanitizes a string by trimming whitespace and escaping HTML
func SanitizeString(s string) string {
	return html.EscapeString(strings.TrimSpace(s))
}

// SanitizeURL sanitizes a URL by trimming whitespace and validating
func SanitizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Add https:// if no scheme is present
	if !hasScheme(rawURL) {
		rawURL = "https://" + rawURL
	}

	if err := ValidateURL(rawURL); err != nil {
		return "", err
	}

	return rawURL, nil
}

// SanitizeXURL sanitizes an X URL
func SanitizeXURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Handle username-only input (e.g., "@username" or "username")
	if !strings.Contains(rawURL, "://") && !strings.Contains(rawURL, ".") {
		// Remove @ if present
		username := strings.TrimPrefix(rawURL, "@")
		rawURL = fmt.Sprintf("https://x.com/%s", username)
	} else if !hasScheme(rawURL) {
		// Add https:// if URL-like but missing scheme
		rawURL = "https://" + rawURL
	}

	// Normalize twitter.com to x.com
	rawURL = strings.Replace(rawURL, "twitter.com", "x.com", 1)
	rawURL = strings.Replace(rawURL, "www.x.com", "x.com", 1)

	if err := ValidateXURL(rawURL); err != nil {
		return "", err
	}

	return rawURL, nil
}

// validateNotEmpty checks if a string is empty after trimming
func validateNotEmpty(s, fieldName string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// hasScheme checks if a URL has an http or https scheme
func hasScheme(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}
