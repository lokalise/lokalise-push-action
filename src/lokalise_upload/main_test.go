package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Override exitFunc for testing
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	// Run tests
	code := m.Run()

	// Restore exitFunc after testing (optional)
	exitFunc = os.Exit

	os.Exit(code)
}

func TestUploadFile(t *testing.T) {
	tests := []struct {
		name         string
		config       UploadConfig
		mockExecutor func(args []string) error
		shouldError  bool
	}{
		{
			name: "Successful upload",
			config: UploadConfig{
				FilePath:      "testfile_success.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				MaxRetries:    3,
				SleepTime:     1,
			},
			mockExecutor: func(args []string) error {
				return nil // Simulate success
			},
			shouldError: false,
		},
		{
			name: "Rate-limited and retries succeed",
			config: UploadConfig{
				FilePath:      "testfile_retry.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				MaxRetries:    3,
				SleepTime:     1,
			},
			mockExecutor: func() func(args []string) error {
				callCount := 0
				return func(args []string) error {
					callCount++
					if callCount == 1 {
						return errors.New("API request error 429")
					}
					return nil
				}
			}(),
			shouldError: false,
		},
		{
			name: "Permanent error",
			config: UploadConfig{
				FilePath:      "testfile_error.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				MaxRetries:    3,
				SleepTime:     1,
			},
			mockExecutor: func(args []string) error {
				return errors.New("Permanent error")
			},
			shouldError: true,
		},
		{
			name: "Max retries exceeded",
			config: UploadConfig{
				FilePath:      "testfile_max_retries.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				MaxRetries:    2,
				SleepTime:     1,
			},
			mockExecutor: func(args []string) error {
				return errors.New("API request error 429")
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable

		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			if tt.config.FilePath != "" {
				f, err := os.Create(tt.config.FilePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				f.Close()
				defer os.Remove(tt.config.FilePath)
			}

			// Capture panic to test error handling
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldError {
						t.Errorf("Unexpected error in test '%s': %v", tt.name, r)
					}
				} else if tt.shouldError {
					t.Errorf("Expected an error in test '%s' but did not get one", tt.name)
				}
			}()

			// Run the uploadFile function with the mock executor
			uploadFile(tt.config, tt.mockExecutor)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      UploadConfig
		shouldError bool
	}{
		{
			name: "Valid configuration",
			config: UploadConfig{
				FilePath:      "valid_file.json",
				ProjectID:     "valid_project_id",
				Token:         "valid_token",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			shouldError: false,
		},
		{
			name: "Missing FilePath",
			config: UploadConfig{
				FilePath:      "",
				ProjectID:     "valid_project_id",
				Token:         "valid_token",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			shouldError: true,
		},
		{
			name: "Non-existent FilePath",
			config: UploadConfig{
				FilePath:      "non_existent_file.json",
				ProjectID:     "valid_project_id",
				Token:         "valid_token",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			shouldError: true,
		},
		{
			name: "Missing ProjectID",
			config: UploadConfig{
				FilePath:      "valid_file.json",
				ProjectID:     "",
				Token:         "valid_token",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			shouldError: true,
		},
		{
			name: "Missing Token",
			config: UploadConfig{
				FilePath:      "valid_file.json",
				ProjectID:     "valid_project_id",
				Token:         "",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			shouldError: true,
		},
		{
			name: "Missing LangISO",
			config: UploadConfig{
				FilePath:      "valid_file.json",
				ProjectID:     "valid_project_id",
				Token:         "valid_token",
				LangISO:       "",
				GitHubRefName: "main",
			},
			shouldError: true,
		},
		{
			name: "Missing GitHubRefName",
			config: UploadConfig{
				FilePath:      "valid_file.json",
				ProjectID:     "valid_project_id",
				Token:         "valid_token",
				LangISO:       "en",
				GitHubRefName: "",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a temporary file if needed
			if tt.config.FilePath != "" && tt.config.FilePath != "non_existent_file.json" {
				f, err := os.Create(tt.config.FilePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				f.Close()
				defer os.Remove(tt.config.FilePath)
			}

			// Capture panic to test error handling
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldError {
						t.Errorf("Unexpected error in test '%s': %v", tt.name, r)
					}
				} else if tt.shouldError {
					t.Errorf("Expected an error in test '%s' but did not get one", tt.name)
				}
			}()

			// Call the validate function
			validate(tt.config)
		})
	}
}

func TestConstructArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   UploadConfig
		expected []string
	}{
		{
			name: "Basic configuration without additional params",
			config: UploadConfig{
				FilePath:      "testfile.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			expected: []string{
				"--token=test_token",
				"--project-id=test_project",
				"file", "upload",
				"--file=testfile.json",
				"--lang-iso=en",
				"--replace-modified",
				"--include-path",
				"--distinguish-by-file",
				"--poll",
				"--poll-timeout=120s",
				"--tag-inserted-keys",
				"--tag-skipped-keys=true",
				"--tag-updated-keys",
				"--tags", "main",
			},
		},
		{
			name: "Configuration with additional params",
			config: UploadConfig{
				FilePath:         "testfile.json",
				ProjectID:        "test_project",
				Token:            "test_token",
				LangISO:          "en",
				GitHubRefName:    "main",
				AdditionalParams: "--custom-param=value1 --another-param=value2",
			},
			expected: []string{
				"--token=test_token",
				"--project-id=test_project",
				"file", "upload",
				"--file=testfile.json",
				"--lang-iso=en",
				"--replace-modified",
				"--include-path",
				"--distinguish-by-file",
				"--poll",
				"--poll-timeout=120s",
				"--tag-inserted-keys",
				"--tag-skipped-keys=true",
				"--tag-updated-keys",
				"--tags", "main",
				"--custom-param=value1", "--another-param=value2",
			},
		},
		{
			name: "Empty configuration",
			config: UploadConfig{
				FilePath:      "",
				ProjectID:     "",
				Token:         "",
				LangISO:       "",
				GitHubRefName: "",
			},
			expected: []string{
				"--token=",
				"--project-id=",
				"file", "upload",
				"--file=",
				"--lang-iso=",
				"--replace-modified",
				"--include-path",
				"--distinguish-by-file",
				"--poll",
				"--poll-timeout=120s",
				"--tag-inserted-keys",
				"--tag-skipped-keys=true",
				"--tag-updated-keys",
				"--tags", "",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := constructArgs(tt.config)

			// Normalize argument spacing for comparison
			actualNormalized := normalizeArgs(actual)
			expectedNormalized := normalizeArgs(tt.expected)

			if !reflect.DeepEqual(actualNormalized, expectedNormalized) {
				t.Errorf("Arguments do not match for test '%s'.\nExpected: %v\nActual:   %v",
					tt.name, expectedNormalized, actualNormalized)
			}
		})
	}
}

// normalizeArgs trims whitespace for consistent comparison of arguments.
func normalizeArgs(args []string) []string {
	normalized := make([]string, len(args))
	for i, arg := range args {
		normalized[i] = strings.TrimSpace(arg)
	}
	return normalized
}
