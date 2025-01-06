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

func TestStoreTranslationPaths(t *testing.T) {
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
			name:       "Nested naming with valid paths",
			paths:      []string{"translations", "more_translations"},
			flatNaming: false,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(".", "translations", "en", "**", "*.json"),
				filepath.Join(".", "more_translations", "en", "**", "*.json"),
			},
		},
		{
			name:       "Flat naming with paths containing spaces",
			paths:      []string{"translations", "more translations", "nested dir/with spaces"},
			flatNaming: true,
			baseLang:   "en-US",
			fileFormat: "yml",
			expected: []string{
				filepath.Join(".", "translations", "en-US.yml"),
				filepath.Join(".", "more translations", "en-US.yml"),
				filepath.Join(".", "nested dir", "with spaces", "en-US.yml"),
			},
		},
		{
			name:       "Nested naming with paths containing spaces",
			paths:      []string{"translations", "more translations", "nested dir/with spaces"},
			flatNaming: false,
			baseLang:   "en-GB",
			fileFormat: "po",
			expected: []string{
				filepath.Join(".", "translations", "en-GB", "**", "*.po"),
				filepath.Join(".", "more translations", "en-GB", "**", "*.po"),
				filepath.Join(".", "nested dir", "with spaces", "en-GB", "**", "*.po"),
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
		{
			name:       "Paths with special characters",
			paths:      []string{"special!@#$%^&()[]{};'", "unicode/路径"},
			flatNaming: true,
			baseLang:   "ja",
			fileFormat: "txt",
			expected: []string{
				filepath.Join(".", "special!@#$%^&()[]{};'", "ja.txt"),
				filepath.Join(".", "unicode", "路径", "ja.txt"),
			},
		},
		{
			name:       "Empty paths list",
			paths:      []string{},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected:   []string{},
		},
		{
			name:       "Paths with empty strings",
			paths:      []string{"", "translations", ""},
			flatNaming: true,
			baseLang:   "en",
			fileFormat: "json",
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
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
			err := storeTranslationPaths(tt.paths, tt.flatNaming, tt.baseLang, tt.fileFormat, &buf)

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
