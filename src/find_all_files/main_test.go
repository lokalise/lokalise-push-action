package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

var baseTestDir string // Shared base directory for all tests

func TestMain(m *testing.M) {
	// Create shared directory structure
	baseTestDir = "test_fs"
	err := setupTestFileStructure(baseTestDir)
	if err != nil {
		panic(err)
	}

	// Override exitFunc for testing
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.RemoveAll(baseTestDir)
	// Restore exitFunc after testing (optional)
	exitFunc = os.Exit

	os.Exit(code)
}

func setupTestFileStructure(baseDir string) error {
	// Create directories
	dirs := []string{
		"flat/translations",
		"nested/en",
		"nested/es",
		"empty",
		"special chars dir",
		"multiple/dir1/en",
		"multiple/dir2/en",
		"multiple/dir3/es",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
			return err
		}
	}

	// Create files
	files := map[string]string{
		"flat/translations/en.json":       "{}",
		"flat/translations/en-US.json":    "{}",
		"flat/translations/fr.json":       "{}",
		"nested/en/file1.json":            "{}",
		"nested/en/file2.json":            "{}",
		"nested/es/file1.json":            "{}",
		"special chars dir/en-US.json":    "{}",
		"flat/translations/unrelated.txt": "skip",
		"multiple/dir1/en/file1.json":     "{}",
		"multiple/dir2/en/file2.json":     "{}",
		"multiple/dir3/es/file3.json":     "{}",
	}

	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		err := os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestFindAllTranslationFiles(t *testing.T) {
	tests := []struct {
		name        string
		paths       []string
		flatNaming  bool
		baseLang    string
		fileFormat  string
		expected    []string
		shouldError bool
	}{
		{
			name:       "Flat naming with valid files",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name:       "Nested naming with valid files",
			paths:      []string{filepath.Join(baseTestDir, "nested")},
			flatNaming: false,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "nested/en/file1.json"),
				filepath.Join(baseTestDir, "nested/en/file2.json"),
			},
		},
		{
			name:       "Nested naming with multiple baseLangs",
			paths:      []string{filepath.Join(baseTestDir, "nested")},
			flatNaming: false,
			baseLang:   "es",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "nested/es/file1.json"),
			},
		},
		{
			name: "Multiple paths",
			paths: []string{
				filepath.Join(baseTestDir, "flat/translations"),
				filepath.Join(baseTestDir, "nested"),
				filepath.Join(baseTestDir, "special chars dir"),
			},
			flatNaming: true,
			baseLang:   "en-US",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en-US.json"),
				filepath.Join(baseTestDir, "special chars dir/en-US.json"),
			},
		},
		{
			name: "Multiple paths with nested naming",
			paths: []string{
				filepath.Join(baseTestDir, "nested"),
				filepath.Join(baseTestDir, "multiple/dir1"),
				filepath.Join(baseTestDir, "multiple/dir2"),
			},
			flatNaming: false,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "nested/en/file1.json"),
				filepath.Join(baseTestDir, "nested/en/file2.json"),
				filepath.Join(baseTestDir, "multiple/dir1/en/file1.json"),
				filepath.Join(baseTestDir, "multiple/dir2/en/file2.json"),
			},
		},
		{
			name:       "Non-existent directory",
			paths:      []string{filepath.Join(baseTestDir, "nonexistent")},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected:   []string{},
		},
		{
			name:       "Empty paths array",
			paths:      []string{},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected:   []string{},
		},
		{
			name:       "Invalid file format",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "invalid",
			expected:   []string{},
		},
		{
			name:       "Flat naming with different baseLang",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "es",
			fileFormat: "json",
			expected:   []string{}, // No files should be found
		},
		{
			name:       "Nested naming with different file format",
			paths:      []string{filepath.Join(baseTestDir, "nested")},
			flatNaming: false,
			baseLang:   "en",
			fileFormat: "txt",
			expected:   []string{}, // No files should be found
		},
		{
			name:       "Special characters in baseLang and fileFormat",
			paths:      []string{filepath.Join(baseTestDir, "special chars dir")},
			flatNaming: true,
			baseLang:   "en-US",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "special chars dir/en-US.json"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := findAllTranslationFiles(tt.paths, tt.flatNaming, tt.baseLang, tt.fileFormat)

			if tt.shouldError {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			actualNormalized := normalizePaths(actual)
			expectedNormalized := normalizePaths(tt.expected)

			slices.Sort(actualNormalized)
			slices.Sort(expectedNormalized)

			if !reflect.DeepEqual(actualNormalized, expectedNormalized) {
				t.Errorf("expected files %v, got %v", expectedNormalized, actualNormalized)
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	t.Run("Valid environment variables", func(t *testing.T) {
		os.Setenv("TRANSLATIONS_PATH", "\npath1\npath2\n\n")
		os.Setenv("BASE_LANG", "en")
		os.Setenv("FILE_FORMAT", "json")
		defer os.Clearenv()

		paths, baseLang, fileFormat := validateEnvironment()

		if len(paths) != 2 || paths[0] != "path1" || paths[1] != "path2" {
			t.Errorf("Unexpected translations paths: %v", paths)
		}
		if baseLang != "en" {
			t.Errorf("Expected baseLang 'en', got '%s'", baseLang)
		}
		if fileFormat != "json" {
			t.Errorf("Expected fileFormat 'json', got '%s'", fileFormat)
		}
	})

	t.Run("Missing environment variables", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for missing environment variables")
			}
		}()

		os.Clearenv()
		validateEnvironment()
	})
}

func TestProcessAllFiles(t *testing.T) {
	t.Run("Files found", func(t *testing.T) {
		mockWrite := func(key, value string) bool {
			if key == "ALL_FILES" && value == "file1,file2" {
				return true
			}
			if key == "has_files" && value == "true" {
				return true
			}
			t.Errorf("Unexpected key-value pair: %s = %s", key, value)
			return false
		}

		processAllFiles([]string{"file1", "file2"}, mockWrite)
	})

	t.Run("No files found", func(t *testing.T) {
		mockWrite := func(key, value string) bool {
			if key == "has_files" && value == "false" {
				return true
			}
			t.Errorf("Unexpected key-value pair: %s = %s", key, value)
			return false
		}

		processAllFiles([]string{}, mockWrite)
	})

	t.Run("WriteOutput fails", func(t *testing.T) {
		mockWrite := func(key, value string) bool {
			return false // Simulate failure
		}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic but got none")
			}
		}()

		processAllFiles([]string{"file1", "file2"}, mockWrite)
	})
}

func normalizePaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = filepath.ToSlash(filepath.Clean(p))
	}
	return normalized
}
