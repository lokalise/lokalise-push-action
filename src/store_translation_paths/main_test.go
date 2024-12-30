package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

func TestValidateEnvironment(t *testing.T) {
	t.Run("Valid environment variables", func(t *testing.T) {
		os.Setenv("TRANSLATIONS_PATH", "\npath1\n\npath2\n")
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

func TestStoreTranslationPaths(t *testing.T) {
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
			name:       "Flat naming with valid paths",
			paths:      []string{"translations", "more_translations"},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "more_translations", "en.json"),
			},
		},
		{
			name:        "Custom naming pattern",
			paths:       []string{"translations", "more_translations"},
			flatNaming:  true,
			baseLang:    "en",
			fileFormat:  "json",
			namePattern: "custom_name.json",
			expected: []string{
				filepath.Join(".", "translations", "custom_name.json"),
				filepath.Join(".", "more_translations", "custom_name.json"),
			},
		},
		{
			name:        "Nested naming with custom pattern",
			paths:       []string{"translations"},
			flatNaming:  false,
			baseLang:    "en",
			fileFormat:  "json",
			namePattern: "**.yaml",
			expected: []string{
				filepath.Join(".", "translations", "**.yaml"),
			},
		},
		{
			name:       "Flat naming with nested paths",
			paths:      []string{"dir1/dir2/dir3", "another/nested/dir"},
			flatNaming: true,
			baseLang:   "fr",
			fileFormat: "xml",
			expected: []string{
				filepath.Join(".", "dir1", "dir2", "dir3", "fr.xml"),
				filepath.Join(".", "another", "nested", "dir", "fr.xml"),
			},
		},
		{
			name:       "Nested naming with nested paths",
			paths:      []string{"dir1/dir2/dir3", "another/nested/dir"},
			flatNaming: false,
			baseLang:   "de",
			fileFormat: "properties",
			expected: []string{
				filepath.Join(".", "dir1", "dir2", "dir3", "de", "**", "*.properties"),
				filepath.Join(".", "another", "nested", "dir", "de", "**", "*.properties"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use a buffer instead of mocking os.Create
			var buf bytes.Buffer

			// Call the function with the buffer as writer
			err := storeTranslationPaths(tt.paths, tt.flatNaming, tt.baseLang, tt.fileFormat, tt.namePattern, &buf)

			if tt.shouldError {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			data := buf.String()

			lines := strings.Split(strings.TrimSpace(data), "\n")
			lines = normalizeLines(lines)

			expected := normalizeLines(tt.expected)

			for _, expectedLine := range expected {
				found := false
				for _, line := range lines {
					if line == expectedLine {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected line: %s", expectedLine)
				}
			}

			if len(lines) != len(expected) {
				t.Errorf("unexpected number of lines. Expected %d, got %d", len(expected), len(lines))
			}
		})
	}
}

func normalizeLines(lines []string) []string {
	var normalized []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized = append(normalized, filepath.ToSlash(line))
	}
	return normalized
}
