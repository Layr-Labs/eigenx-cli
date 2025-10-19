package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetEnvFileVariable(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error // Setup test environment
		filePath      string
		key           string
		value         string
		wantErr       bool
		expectedValue string
		errorContains string
	}{
		{
			name: "creates new file when doesn't exist",
			setup: func(dir string) error {
				return nil // No setup needed
			},
			filePath:      "test.env",
			key:           "TEST_KEY",
			value:         "test_value",
			wantErr:       false,
			expectedValue: "TEST_KEY=test_value\n",
		},
		{
			name: "updates existing key",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.env"), []byte("TEST_KEY=old_value\nOTHER=value\n"), 0644)
			},
			filePath:      "test.env",
			key:           "TEST_KEY",
			value:         "new_value",
			wantErr:       false,
			expectedValue: "TEST_KEY=new_value",
		},
		{
			name: "adds new key to existing file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.env"), []byte("EXISTING=value\n"), 0644)
			},
			filePath:      "test.env",
			key:           "NEW_KEY",
			value:         "new_value",
			wantErr:       false,
			expectedValue: "NEW_KEY=new_value",
		},
		{
			name: "preserves comments",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.env"), []byte("# Comment\nKEY=value\n"), 0644)
			},
			filePath:      "test.env",
			key:           "KEY",
			value:         "updated",
			wantErr:       false,
			expectedValue: "# Comment",
		},
		{
			name: "replaces existing key value",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.env"), []byte("TEST_KEY=old_value\n"), 0644)
			},
			filePath:      "test.env",
			key:           "TEST_KEY",
			value:         "new_value",
			wantErr:       false,
			expectedValue: "TEST_KEY=new_value\n",
		},
		{
			name: "errors on directory path",
			setup: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, "testdir"), 0755)
			},
			filePath:      "testdir",
			key:           "KEY",
			value:         "value",
			wantErr:       true,
			errorContains: "path is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "eigenx-test-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test environment
			if tt.setup != nil {
				if err := tt.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			// Get full path
			fullPath := filepath.Join(tmpDir, tt.filePath)

			// Call SetEnvFileVariable
			err = SetEnvFileVariable(fullPath, tt.key, tt.value)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("SetEnvFileVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("SetEnvFileVariable() error = %v, want error containing %q", err, tt.errorContains)
				}
			}

			if !tt.wantErr {
				// Read file and check contents
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}

				if !strings.Contains(string(content), tt.expectedValue) {
					t.Errorf("SetEnvFileVariable() content = %q, want to contain %q", string(content), tt.expectedValue)
				}
			}
		})
	}
}

func TestIsEnvKeyMatch(t *testing.T) {
	tests := []struct {
		name string
		line string
		key  string
		want bool
	}{
		{
			name: "exact match",
			line: "TEST_KEY=value",
			key:  "TEST_KEY",
			want: true,
		},
		{
			name: "no match - different key",
			line: "OTHER_KEY=value",
			key:  "TEST_KEY",
			want: false,
		},
		{
			name: "no match - comment",
			line: "# TEST_KEY=value",
			key:  "TEST_KEY",
			want: false,
		},
		{
			name: "no match - empty line",
			line: "",
			key:  "TEST_KEY",
			want: false,
		},
		{
			name: "no match - prefix only",
			line: "TEST_KEY_PUBLIC=value",
			key:  "TEST_KEY",
			want: false,
		},
		{
			name: "match with spaces in value",
			line: "KEY=value with spaces",
			key:  "KEY",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEnvKeyMatch(tt.line, tt.key); got != tt.want {
				t.Errorf("isEnvKeyMatch(%q, %q) = %v, want %v", tt.line, tt.key, got, tt.want)
			}
		})
	}
}
