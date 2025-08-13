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
	"time"
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

func TestExecuteUploadTimeout_Integration(t *testing.T) {
	// Build the mock sleep binary
	mockBinary := "./fixtures/sleep/mock_sleep"
	if runtime.GOOS == "windows" {
		mockBinary += ".exe"
	}
	buildMockBinaryIfNeeded(t, "./fixtures/sleep/sleep.go", mockBinary)

	args := []string{"sleep"} // makes the fixture sleep 2s
	uploadTimeout := 1        // 1s timeout so it should trip

	err := executeUpload(mockBinary, args, uploadTimeout)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	// Be robust against optional stderr suffix; just check the prefix
	wantPrefix := fmt.Sprintf("command timed out after %ds", uploadTimeout)
	if !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("want error prefix %q, got %q", wantPrefix, err.Error())
	}
}

func TestExecuteUpload_RateLimitStderrDetected(t *testing.T) {
	// Build the 429-stderr binary
	bin := "./fixtures/exit_429/exit_429"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	buildMockBinaryIfNeeded(t, "./fixtures/exit_429/exit_429.go", bin)

	// No args, immediate exit with 429-ish stderr
	err := executeUpload(bin, nil, 3)
	if err == nil {
		t.Fatalf("expected non-nil error from 429 mock")
	}
	if !isRateLimitError(err) {
		t.Fatalf("expected isRateLimitError to be true; got err=%q", err.Error())
	}
}

func TestExecuteUpload_NonRateLimitError(t *testing.T) {
	// Build the non-429-stderr binary
	bin := "./fixtures/exit_err/exit_err"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	buildMockBinaryIfNeeded(t, "./fixtures/exit_err/exit_err.go", bin)

	err := executeUpload(bin, nil, 3)
	if err == nil {
		t.Fatalf("expected non-nil error from error mock")
	}
	if isRateLimitError(err) {
		t.Fatalf("expected isRateLimitError to be false; got err=%q", err.Error())
	}
}

func TestUploadFile_RetriesOnRateLimit_WithMock(t *testing.T) {
	cfg := UploadConfig{
		FilePath:      "testfile_retry.json",
		ProjectID:     "test_project",
		Token:         "test_token",
		LangISO:       "en",
		GitHubRefName: "main",
		MaxRetries:    3,
		SleepTime:     0,
		UploadTimeout: 120,
	}

	// temp file so validateFile passes
	f, err := os.Create(cfg.FilePath)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_ = f.Close()
	defer os.Remove(cfg.FilePath)

	call := 0
	mockExec := func(cmdPath string, args []string, uploadTimeout int) error {
		call++
		if call == 1 {
			return fmt.Errorf("API request error 429: boom")
		}
		return nil
	}

	done := make(chan struct{})
	go func() {
		uploadFile(cfg, mockExec)
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatalf("uploadFile did not complete in time (likely stuck)")
	}

	if call != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", call)
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

func TestExecuteUpload_WrapsExitErrorAndStderr(t *testing.T) {
	bin := "./fixtures/exit_err/exit_err"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	buildMockBinaryIfNeeded(t, "./fixtures/exit_err/exit_err.go", bin)

	err := executeUpload(bin, nil, 3)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "some permanent error happened") {
		t.Fatalf("stderr not included: %q", err.Error())
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected wrapped exec.ExitError")
	}
}

func TestUploadFile_MaxRetriesCallCount(t *testing.T) {
	cfg := UploadConfig{
		FilePath:      "retry_forever.json",
		ProjectID:     "p",
		Token:         "tok",
		LangISO:       "en",
		GitHubRefName: "main",
		MaxRetries:    4,
		SleepTime:     0, // keep fast
		UploadTimeout: 1,
	}
	f, _ := os.Create(cfg.FilePath)
	f.Close()
	defer os.Remove(cfg.FilePath)

	var calls int
	mockExec := func(cmdPath string, args []string, uploadTimeout int) error {
		calls++
		return fmt.Errorf("API request error 429: nope")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic after retries exhausted")
		}
		if calls != cfg.MaxRetries {
			t.Fatalf("expected %d attempts, got %d", cfg.MaxRetries, calls)
		}
	}()
	uploadFile(cfg, mockExec)
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

func TestValidate_DirectoryPath(t *testing.T) {
	dir := t.TempDir()
	cfg := UploadConfig{
		FilePath:      dir, // directory, not a file
		ProjectID:     "p",
		Token:         "tok",
		LangISO:       "en",
		GitHubRefName: "main",
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected error for directory path, got none")
		}
	}()
	validate(cfg)
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
