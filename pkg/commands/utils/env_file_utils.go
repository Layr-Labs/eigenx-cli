package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// defaultEnvFileMode is the default file permission for env files
	defaultEnvFileMode = 0644
)

// SetEnvFileVariable sets or updates a key-value pair in an environment file.
// This function is idempotent and handles both file creation and updates:
// - If the file doesn't exist, it creates it with the key-value pair
// - If the file exists, it updates or adds the key-value pair
// - If the key exists, it replaces the value
// - If the key doesn't exist, it appends it
// - Preserves comments and formatting in existing files
// Returns an error if the path is a directory.
func SetEnvFileVariable(filePath, key, value string) error {
	info, err := os.Stat(filePath)

	if err == nil {
		// File exists - check if it's a directory
		if info.IsDir() {
			return fmt.Errorf("path is a directory, not a file: %s", filePath)
		}
		// File exists and is a file, update it
		return updateExistingEnvFile(filePath, key, value)
	}

	if !os.IsNotExist(err) {
		// Some other error (permissions, etc.)
		return fmt.Errorf("failed to access env file %s: %w", filePath, err)
	}

	// File doesn't exist, create it
	return createNewEnvFile(filePath, key, value)
}

// createNewEnvFile creates a new environment file with a single key-value pair
func createNewEnvFile(filePath, key, value string) error {
	content := fmt.Sprintf("%s=%s\n", key, value)
	if err := os.WriteFile(filePath, []byte(content), defaultEnvFileMode); err != nil {
		return fmt.Errorf("failed to create env file %s: %w", filePath, err)
	}
	return nil
}

// updateExistingEnvFile updates or adds a key-value pair in an existing environment file
func updateExistingEnvFile(filePath, key, value string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open env file %s: %w", filePath, err)
	}
	defer file.Close()

	var lines []string
	var keyFound bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if this line contains our key (more precise matching)
		if isEnvKeyMatch(trimmed, key) {
			// Replace the first occurrence only
			if !keyFound {
				lines = append(lines, fmt.Sprintf("%s=%s", key, value))
				keyFound = true
			}
			// Skip duplicate keys if they exist
		} else {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	// If key wasn't found, append it
	if !keyFound {
		// Add a blank line if file isn't empty and doesn't end with blank line
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back to file
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filePath, []byte(content), defaultEnvFileMode); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

// isEnvKeyMatch checks if a line matches the given environment variable key
// Returns true only if the line is exactly "KEY=..." (not a comment, not a different key)
func isEnvKeyMatch(line, key string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Split on first '=' to get the key part
	parts := strings.SplitN(line, "=", 2)
	if len(parts) < 1 {
		return false
	}

	// Compare the key exactly
	return parts[0] == key
}
