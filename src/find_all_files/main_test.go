package main

import (
	"fmt"
	"log"
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
	err = os.RemoveAll(baseTestDir)
	if err != nil {
		log.Printf("Failed to remove %s: %v", baseTestDir, err)
	}
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
		"flat/translations/en.yaml":       "{}",
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
		fileExt     []string
		namePattern string
		expected    []string
		shouldError bool
	}{
		{
			name:       "Flat naming with valid files",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name:       "Flat naming with valid files and multiple exts",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json", "yaml"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
				filepath.Join(baseTestDir, "flat/translations/en.yaml"),
			},
		},
		{
			name:        "Custom name pattern with wildcard",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations"), filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileExt:     []string{""},
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
			fileExt:     []string{""},
			namePattern: "[invalid pattern",
			expected:    nil,
			shouldError: true,
		},
		{
			name:       "Mixed flat and nested paths",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations"), filepath.Join(baseTestDir, "nested")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name:        "Case sensitivity check (may vary by OS)",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileExt:     []string{""},
			namePattern: "**/*.JSON", // Intentionally capitalized
			expected:    []string{},  // Should be empty on case-sensitive systems
		},
		{
			name:       "Empty directory",
			paths:      []string{filepath.Join(baseTestDir, "empty")},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"json"},
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
			fileExt:     []string{""},
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
			fileExt:     []string{""},
			namePattern: "es/**/custom_*.json", // No matching files
			expected:    []string{},
		},
		{
			name:       "Root directory translations with flat naming",
			paths:      []string{filepath.Join(baseTestDir)},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "en.json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := findAllTranslationFiles(tt.paths, tt.flatNaming, tt.baseLang, tt.fileExt, tt.namePattern)

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
		t.Setenv("TRANSLATIONS_PATH", "\npath1\npath2\n\n")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_FORMAT", "json")
		t.Setenv("NAME_PATTERN", "custom_name.json")

		paths, baseLang, fileExt, namePattern := validateEnvironment()

		if len(paths) != 2 || paths[0] != "path1" || paths[1] != "path2" {
			t.Errorf("Unexpected translations paths: %v", paths)
		}
		if baseLang != "en" {
			t.Errorf("Expected baseLang 'en', got '%s'", baseLang)
		}
		want := []string{"json"}
		if !reflect.DeepEqual(fileExt, want) {
			t.Errorf("fileExt mismatch. want=%v got=%v", want, fileExt)
		}
		if namePattern != "custom_name.json" {
			t.Errorf("Expected namePattern 'custom_name.json', got '%s'", namePattern)
		}
	})

	t.Run("FILE_EXT has precedence over FILE_FORMAT", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "\npath1\npath2\n\n")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_FORMAT", "json_structured")
		t.Setenv("FILE_EXT", "json\nyaml")
		t.Setenv("NAME_PATTERN", "custom_name.json")

		_, _, fileExt, _ := validateEnvironment()

		want := []string{"json", "yaml"}
		if !reflect.DeepEqual(fileExt, want) {
			t.Errorf("fileExt mismatch. want=%v got=%v", want, fileExt)
		}
	})

	t.Run("Missing environment variables", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "")
		t.Setenv("BASE_LANG", "")
		t.Setenv("FILE_FORMAT", "")
		t.Setenv("FILE_EXT", "")
		t.Setenv("NAME_PATTERN", "")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for missing environment variables")
			}
		}()

		validateEnvironment()
	})

	t.Run("validate: roots cleaned and relative; dot root OK", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", ".\n./locales\nlocales/../locales/en/..")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		paths, base, exts, pat := validateEnvironment()

		if base != "en" || pat != "" {
			t.Fatalf("unexpected base/pattern: %q / %q", base, pat)
		}
		want := []string{".", "locales", "locales"} // clean collapses
		for i, p := range paths {
			if filepath.ToSlash(p) != filepath.ToSlash(want[i]) {
				t.Fatalf("paths[%d]=%q, want %q", i, p, want[i])
			}
		}
		if !reflect.DeepEqual(exts, []string{"json"}) {
			t.Fatalf("exts: %v", exts)
		}
	})

	t.Run("validate: absolute and parent escape fail", func(t *testing.T) {
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for absolute root")
				}
			}()
			t.Setenv("TRANSLATIONS_PATH", "/etc/locales")
			validateEnvironment()
		}()

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for parent escape root")
				}
			}()
			t.Setenv("TRANSLATIONS_PATH", "../locales")
			validateEnvironment()
		}()
	})

	t.Run("validate: namePattern glob variants OK, but absolute fails", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "translations")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")

		// ok patterns
		for _, np := range []string{"**/*.yaml", "en/**/custom_*.json", "dir/**/*.po"} {
			t.Setenv("NAME_PATTERN", np)
			_, _, _, pat := validateEnvironment()
			if got := filepath.ToSlash(pat); got != np {
				t.Fatalf("pattern got %q, want %q", got, np)
			}
		}

		// absolute pattern -> fail
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for absolute NAME_PATTERN")
				}
			}()
			t.Setenv("NAME_PATTERN", "/tmp/**/*.json")
			validateEnvironment()
		}()
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
