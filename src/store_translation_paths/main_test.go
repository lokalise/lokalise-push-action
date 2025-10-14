package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

func TestValidateEnvironment(t *testing.T) {
	t.Run("Valid environment variables", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "\npath1\n\npath2\n")
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
			t.Fatalf("fileExt mismatch. want=%v got=%v", want, fileExt)
		}
		if namePattern != "custom_name.json" {
			t.Errorf("Expected namePattern 'custom_name.json', got '%s'", namePattern)
		}
	})

	t.Run("FILE_EXT has precedence over FILE_FORMAT", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "\npath1\n\npath2\n")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_FORMAT", "json_structured")
		t.Setenv("FILE_EXT", "json\nyaml")
		t.Setenv("NAME_PATTERN", "custom_name.json")

		_, _, fileExt, _ := validateEnvironment()

		want := []string{"json", "yaml"}
		if !reflect.DeepEqual(fileExt, want) {
			t.Fatalf("fileExt mismatch. want=%v got=%v", want, fileExt)
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

	t.Run("Root translation path", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", ".")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")

		paths, baseLang, exts, pattern := validateEnvironment()

		if len(paths) != 1 || paths[0] != "." {
			t.Fatalf("expected paths=[\".\"], got %v", paths)
		}
		if baseLang != "en" {
			t.Fatalf("expected baseLang=en, got %s", baseLang)
		}
		if !reflect.DeepEqual(exts, []string{"json"}) {
			t.Fatalf("expected exts=[json], got %v", exts)
		}
		if pattern != "" {
			t.Fatalf("expected empty namePattern, got %q", pattern)
		}
	})

	t.Run("Name pattern with ../", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "locales")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		t.Setenv("NAME_PATTERN", "../**/*.json")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for NAME_PATTERN escaping repo")
			}
		}()

		validateEnvironment()
	})

	t.Run("Translation path with ../", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "../locales")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for TRANSLATIONS_PATH escaping repo")
			}
		}()

		validateEnvironment()
	})

	t.Run("TRANSLATIONS_PATH cleans to .. fails", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic when path cleans to '..'")
			}
		}()
		t.Setenv("TRANSLATIONS_PATH", "a/../..")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		validateEnvironment()
	})

	t.Run("TRANSLATIONS_PATH './path' is OK (relative)", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "./path")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")

		paths, _, _, _ := validateEnvironment()
		if len(paths) != 1 || filepath.ToSlash(paths[0]) != "path" {
			t.Fatalf("expected cleaned relative path 'path', got %v", paths)
		}
	})

	t.Run("TRANSLATIONS_PATH '/path' fails (absolute)", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for absolute TRANSLATIONS_PATH")
			}
		}()
		t.Setenv("TRANSLATIONS_PATH", "/path")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		validateEnvironment()
	})

	t.Run("NamePattern glob OK", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "translations")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		t.Setenv("NAME_PATTERN", "**/*.yaml")

		_, _, _, pattern := validateEnvironment()
		if got := filepath.ToSlash(pattern); got != "**/*.yaml" {
			t.Fatalf("expected namePattern '**/*.yaml', got %q", got)
		}
	})

	t.Run("NamePattern nested glob OK", func(t *testing.T) {
		t.Setenv("TRANSLATIONS_PATH", "pkg/i18n")
		t.Setenv("BASE_LANG", "en")
		t.Setenv("FILE_EXT", "json")
		t.Setenv("NAME_PATTERN", "en/**/custom_*.json")

		_, _, _, pattern := validateEnvironment()
		if got := filepath.ToSlash(pattern); got != "en/**/custom_*.json" {
			t.Fatalf("expected pattern 'en/**/custom_*.json', got %q", got)
		}
	})
}

func TestStoreTranslationPaths(t *testing.T) {
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
			name:       "Flat naming with valid paths",
			paths:      []string{"translations", "more_translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "more_translations", "en.json"),
			},
		},
		{
			name:       "Flat naming with valid path and multiple exts",
			paths:      []string{"translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json", "yaml"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
		},
		{
			name:        "Custom naming pattern",
			paths:       []string{"translations", "more_translations"},
			flatNaming:  true,
			baseLang:    "en",
			fileExt:     []string{"json"},
			namePattern: "custom_name.json",
			expected: []string{
				filepath.Join(".", "translations", "custom_name.json"),
				filepath.Join(".", "more_translations", "custom_name.json"),
			},
		},
		{
			name:        "Nested naming with custom pattern",
			paths:       []string{"translations", "translations"},
			flatNaming:  false,
			baseLang:    "en",
			fileExt:     []string{"json"},
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
			fileExt:    []string{"xml"},
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
			fileExt:    []string{"properties"},
			expected: []string{
				filepath.Join(".", "dir1", "dir2", "dir3", "de", "**", "*.properties"),
				filepath.Join(".", "another", "nested", "dir", "de", "**", "*.properties"),
			},
		},

		{
			name:       "Root path (.) with flat naming",
			paths:      []string{"."},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(".", ".", "en.json"), // normalizes to ././en.json, effectively ./en.json
			},
		},
		{
			name:        "Root path (.) with custom name pattern",
			paths:       []string{"."},
			flatNaming:  false,
			baseLang:    "en",
			fileExt:     []string{"json"},
			namePattern: "some_dir/**.yaml",
			expected: []string{
				filepath.Join(".", ".", "some_dir", "**.yaml"), // e.g. ././some_dir/**.yaml
			},
		},
		{
			name:        "Complex custom name pattern",
			paths:       []string{"translations"},
			flatNaming:  false,
			baseLang:    "en",
			fileExt:     []string{"json"},
			namePattern: "en/**/custom_*.json",
			expected: []string{
				filepath.Join(".", "translations", "en", "**", "custom_*.json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use a buffer instead of mocking os.Create
			var buf bytes.Buffer

			err := storeTranslationPaths(tt.paths, tt.flatNaming, tt.baseLang, tt.fileExt, tt.namePattern, &buf)

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
