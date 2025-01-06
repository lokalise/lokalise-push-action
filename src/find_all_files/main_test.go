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
		"locales/en/sub1",
		"locales/fr",
		"i18n/en/sub2",
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
		"locales/en/sub1/custom_abc.json": "{}",
		"i18n/en/sub2/custom_xyz.json":    "{}",
		"locales/fr/whatever.json":        "{}",
		"en.json":                         "{}",
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
		namePattern string
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
			name:        "Custom name pattern with wildcard",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileFormat:  "",
			namePattern: "**/*.json",
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
				filepath.Join(baseTestDir, "flat/translations/en-US.json"),
				filepath.Join(baseTestDir, "flat/translations/fr.json"),
			},
		},
		{
			name:        "Invalid name pattern",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileFormat:  "",
			namePattern: "[invalid pattern",
			expected:    nil,
			shouldError: true,
		},
		{
			name:       "Mixed flat and nested paths",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations"), filepath.Join(baseTestDir, "nested")},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name:        "Case sensitivity check (may vary by OS)",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileFormat:  "",
			namePattern: "**/*.JSON", // Intentionally capitalized
			expected:    []string{},  // Should be empty on case-sensitive systems
		},
		{
			name:       "Empty directory",
			paths:      []string{filepath.Join(baseTestDir, "empty")},
			flatNaming: false,
			baseLang:   "en",
			fileFormat: "json",
			expected:   []string{},
		},
		{
			name: "Multiple valid paths",
			paths: []string{
				filepath.Join(baseTestDir, "locales"),
				filepath.Join(baseTestDir, "i18n"),
			},
			flatNaming:  false,
			baseLang:    "",
			fileFormat:  "",
			namePattern: "en/**/custom_*.json",
			expected: []string{
				filepath.Join(baseTestDir, "locales/en/sub1/custom_abc.json"),
				filepath.Join(baseTestDir, "i18n/en/sub2/custom_xyz.json"),
			},
		},
		{
			name:        "Custom pattern with no matches",
			paths:       []string{filepath.Join(baseTestDir, "locales")},
			flatNaming:  false,
			baseLang:    "",
			fileFormat:  "",
			namePattern: "es/**/custom_*.json", // No matching files
			expected:    []string{},
		},
		{
			name:       "Root directory translations with flat naming",
			paths:      []string{filepath.Join(baseTestDir)},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(baseTestDir, "en.json"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := findAllTranslationFiles(tt.paths, tt.flatNaming, tt.baseLang, tt.fileFormat, tt.namePattern)

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
		os.Setenv("NAME_PATTERN", "custom_name.json")
		defer os.Clearenv()

		paths, baseLang, fileFormat, namePattern := validateEnvironment()

		if len(paths) != 2 || paths[0] != "path1" || paths[1] != "path2" {
			t.Errorf("Unexpected translations paths: %v", paths)
		}
		if baseLang != "en" {
			t.Errorf("Expected baseLang 'en', got '%s'", baseLang)
		}
		if fileFormat != "json" {
			t.Errorf("Expected fileFormat 'json', got '%s'", fileFormat)
		}
		if namePattern != "custom_name.json" {
			t.Errorf("Expected namePattern 'custom_name.json', got '%s'", namePattern)
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
