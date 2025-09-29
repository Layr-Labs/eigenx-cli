package common

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetLogger_ReturnsLoggerAndTracker(t *testing.T) {
	log, tracker := GetLogger(false)

	logType := reflect.TypeOf(log).String()
	trackerType := reflect.TypeOf(tracker).String()

	if !isValidLogger(logType) {
		t.Errorf("unexpected logger type: %s", logType)
	}
	if !isValidTracker(trackerType) {
		t.Errorf("unexpected tracker type: %s", trackerType)
	}
}

func isValidLogger(typ string) bool {
	return typ == "*logger.Logger" || typ == "*logger.ZapLogger"
}

func isValidTracker(typ string) bool {
	return typ == "*progress.TTYProgressTracker" || typ == "*progress.LogProgressTracker"
}

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid simple name",
			input:   "myapp",
			wantErr: false,
		},
		{
			name:    "Valid name with numbers",
			input:   "app123",
			wantErr: false,
		},
		{
			name:    "Valid name with hyphens",
			input:   "my-app",
			wantErr: false,
		},
		{
			name:    "Valid name with underscores",
			input:   "my_app",
			wantErr: false,
		},
		{
			name:    "Valid complex name",
			input:   "my-app_123",
			wantErr: false,
		},
		{
			name:    "Valid minimum length",
			input:   "ab",
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
			errMsg:  "app name cannot be empty",
		},
		{
			name:    "Too short - single character",
			input:   "a",
			wantErr: true,
			errMsg:  "app name must be at least 2 characters long",
		},
		{
			name:    "Too long - over 255 characters",
			input:   "a" + strings.Repeat("b", 255), // 256 characters
			wantErr: true,
			errMsg:  "app name must be 255 characters or less",
		},
		{
			name:    "Invalid - uppercase letters",
			input:   "MyApp",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Invalid - spaces",
			input:   "my app",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Invalid - dots",
			input:   "my.app",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Invalid - special characters",
			input:   "my@app",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Invalid - forward slash",
			input:   "my/app",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Invalid - colon",
			input:   "my:app",
			wantErr: true,
			errMsg:  "app name can only contain lowercase letters, numbers, hyphens (-), and underscores (_)",
		},
		{
			name:    "Valid long name - exactly 255 characters",
			input:   strings.Repeat("a", 255),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppName(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAppName() expected error for input '%s', but got none", tt.input)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidateAppName() error = '%s', expected '%s'", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAppName() unexpected error for input '%s': %v", tt.input, err)
				}
			}
		})
	}
}
