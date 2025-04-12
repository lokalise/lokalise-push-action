package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime"
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

func TestExecuteUploadTimeout(t *testing.T) {
	// Mock to replace the real call to lokalise2
	mockBinary := "./fixtures/mock_sleep"
	if runtime.GOOS == "windows" {
		mockBinary += ".exe"
	}

	// Build the mock binary if needed
	buildMockBinaryIfNeeded(t, "./fixtures/sleep.go", mockBinary)

	args := []string{"sleep"} // Argument to trigger sleep in the mock process
	uploadTimeout := 1        // Timeout in seconds, smaller than sleep duration

	fmt.Println("Executing upload with timeout...")

	// Call the actual executeUpload function
	err := executeUpload(mockBinary, args, uploadTimeout)

	fmt.Println("Execution completed.")

	// Debugging: Display the exact error received
	if err != nil {
		fmt.Printf("Error returned: %v\n", err)
	}

	// Validate the result
	if err == nil {
		t.Errorf("Expected timeout error, but got nil")
	} else if err.Error() != "command timed out" {
		t.Errorf("Expected 'command timed out' error, but got: %v", err)
	}
}

func TestUploadFile(t *testing.T) {
	tests := []struct {
		name         string
		config       UploadConfig
		mockExecutor func(cmdPath string, args []string, uploadTimeout int) error
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
				UploadTimeout: 120,
			},
			mockExecutor: func(cmdPath string, args []string, uploadTimeout int) error {
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
				UploadTimeout: 120,
			},
			mockExecutor: func() func(cmdPath string, args []string, uploadTimeout int) error {
				callCount := 0
				return func(cmdPath string, args []string, uploadTimeout int) error {
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
				UploadTimeout: 120,
			},
			mockExecutor: func(cmdPath string, args []string, uploadTimeout int) error {
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
				UploadTimeout: 120,
			},
			mockExecutor: func(cmdPath string, args []string, uploadTimeout int) error {
				return errors.New("API request error 429")
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a temporary file for testing
			if tt.config.FilePath != "" {
				f, err := os.Create(tt.config.FilePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				err = f.Close()
				if err != nil {
					log.Printf("Failed to close %s: %v", tt.config.FilePath, err)
				}
				defer func() {
					if err := os.Remove(tt.config.FilePath); err != nil {
						log.Printf("Failed to remove %s: %v", tt.config.FilePath, err)
					}
				}()
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
				PollTimeout:   120,
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
				PollTimeout:   120,
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
				PollTimeout:   120,
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
				PollTimeout:   120,
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
				PollTimeout:   120,
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
				PollTimeout:   120,
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
				PollTimeout:   120,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable

		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file if needed
			if tt.config.FilePath != "" && tt.config.FilePath != "non_existent_file.json" {
				f, err := os.Create(tt.config.FilePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				err = f.Close()
				if err != nil {
					log.Printf("Failed to close %s: %v", tt.config.FilePath, err)
				}
				defer func() {
					if err := os.Remove(tt.config.FilePath); err != nil {
						log.Printf("Failed to remove %s: %v", tt.config.FilePath, err)
					}
				}()
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
				PollTimeout:   120,
				SkipTagging:   false,
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
				"--tag-skipped-keys",
				"--tag-updated-keys",
				"--tags", "main",
			},
		},
		{
			name: "Configuration with SkipTagging enabled",
			config: UploadConfig{
				FilePath:      "testfile.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				PollTimeout:   120,
				SkipTagging:   true,
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
			},
		},
		{
			name: "Configuration with SkipPolling enabled",
			config: UploadConfig{
				FilePath:      "testfile.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				SkipPolling:   true,
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
				"--tag-inserted-keys",
				"--tag-skipped-keys",
				"--tag-updated-keys",
				"--tags", "main",
			},
		},
		{
			name: "Configuration with SkipDefaultFlags enabled",
			config: UploadConfig{
				FilePath:         "testfile.json",
				ProjectID:        "test_project",
				Token:            "test_token",
				LangISO:          "en",
				GitHubRefName:    "main",
				SkipDefaultFlags: true,
				SkipTagging:      true,
				PollTimeout:      120,
			},
			expected: []string{
				"--token=test_token",
				"--project-id=test_project",
				"file", "upload",
				"--file=testfile.json",
				"--lang-iso=en",
				"--poll",
				"--poll-timeout=120s",
			},
		},
		{
			name: "Configuration with multiple additional params",
			config: UploadConfig{
				FilePath:      "testfile.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				AdditionalParams: `
--convert-placeholders
--custom-flag=true
--another-flag=false
--quoted="some value"
--json={"key": "value with space"}
`,
				PollTimeout: 120,
				SkipTagging: false,
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
				"--tag-skipped-keys",
				"--tag-updated-keys",
				"--tags", "main",
				"--convert-placeholders",
				"--custom-flag=true",
				"--another-flag=false",
				`--quoted="some value"`,
				`--json={"key": "value with space"}`,
			},
		},
		{
			name: "Configuration with extra spaces in additional params",
			config: UploadConfig{
				FilePath:      "testfile.json",
				ProjectID:     "test_project",
				Token:         "test_token",
				LangISO:       "en",
				GitHubRefName: "main",
				AdditionalParams: `
--flag1=value1
--flag2=value2
--spaced="this  has   multiple spaces"
`,
				PollTimeout: 120,
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
				"--tag-skipped-keys",
				"--tag-updated-keys",
				"--tags", "main",
				"--flag1=value1",
				"--flag2=value2",
				`--spaced="this  has   multiple spaces"`,
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
				PollTimeout:   0,
				SkipTagging:   true,
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
				"--poll-timeout=0s",
			},
		},
		{
			name: "Configuration with multiple additional params (YAML style)",
			config: UploadConfig{
				FilePath:      "locales/en.json",
				ProjectID:     "proj_abc123",
				Token:         "super_secret",
				LangISO:       "en",
				GitHubRefName: "release",
				PollTimeout:   180,
				AdditionalParams: `
--directory-prefix=%LANG_ISO%
--indentation=4sp
--json-unescaped-slashes=true
--export-empty-as=skip
--export-sort=a_z
--replace-breaks=false
--language-mapping=[{"original_language_iso":"en_US","custom_language_iso":"en-US"},{"original_language_iso":"fr_CA","custom_language_iso":"fr-CA"}]
`,
			},
			expected: []string{
				"--token=super_secret",
				"--project-id=proj_abc123",
				"file", "upload",
				"--file=locales/en.json",
				"--lang-iso=en",
				"--replace-modified",
				"--include-path",
				"--distinguish-by-file",
				"--poll",
				"--poll-timeout=180s",
				"--tag-inserted-keys",
				"--tag-skipped-keys",
				"--tag-updated-keys",
				"--tags", "release",
				"--directory-prefix=%LANG_ISO%",
				"--indentation=4sp",
				"--json-unescaped-slashes=true",
				"--export-empty-as=skip",
				"--export-sort=a_z",
				"--replace-breaks=false",
				// Note that in reality the upload does not have language mappings
				`--language-mapping=[{"original_language_iso":"en_US","custom_language_iso":"en-US"},{"original_language_iso":"fr_CA","custom_language_iso":"fr-CA"}]`,
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

// buildMockBinaryIfNeeded compiles the binary only if it doesn’t exist or is outdated.
func buildMockBinaryIfNeeded(t *testing.T, sourcePath, outputPath string) {
	// Check if the binary already exists and is up-to-date
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	binaryInfo, err := os.Stat(outputPath)
	if err == nil && binaryInfo.ModTime().After(sourceInfo.ModTime()) {
		// Binary exists and is newer than the source, no need to rebuild
		return
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", outputPath, sourcePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build mock binary: %v", err)
	}
}
