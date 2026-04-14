package main

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

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
			name:        "Custom pattern works with empty file extensions",
			paths:       []string{filepath.Join(baseTestDir, "pattern-only")},
			flatNaming:  true,
			baseLang:    "zz",
			fileExt:     nil,
			namePattern: "**/custom_name.json",
			expected: []string{
				filepath.Join(baseTestDir, "pattern-only/sub/custom_name.json"),
			},
		},
		{
			name:       "Flat naming missing files is not an error",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "de",
			fileExt:    []string{"json"},
			expected:   []string{},
		},
		{
			name:       "Nested naming finds files recursively",
			paths:      []string{filepath.Join(baseTestDir, "nested")},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "nested/en/file1.json"),
				filepath.Join(baseTestDir, "nested/en/file2.json"),
				filepath.Join(baseTestDir, "nested/en/deeper/file4.json"),
			},
		},
		{
			name:       "Nested naming matches extensions case-insensitively",
			paths:      []string{filepath.Join(baseTestDir, "nested")},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"yaml"},
			expected: []string{
				filepath.Join(baseTestDir, "nested/en/file3.YAML"),
			},
		},
		{
			name:       "Nested naming missing language directory is not an error",
			paths:      []string{filepath.Join(baseTestDir, "empty")},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected:   []string{},
		},
		{
			name:       "Mixed flat roots only return matching flat files",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations"), filepath.Join(baseTestDir, "nested")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
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
			name:        "Custom pattern overrides other inputs",
			paths:       []string{filepath.Join(baseTestDir, "pattern-only")},
			flatNaming:  true,
			baseLang:    "zz",
			fileExt:     []string{"xml"},
			namePattern: "**/custom_name.json",
			expected: []string{
				filepath.Join(baseTestDir, "pattern-only/sub/custom_name.json"),
			},
		},
		{
			name:        "Invalid name pattern",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileExt:     []string{""},
			namePattern: "[invalid pattern",
			shouldError: true,
		},
		{
			name:        "Case-sensitive pattern with no matches",
			paths:       []string{filepath.Join(baseTestDir, "flat/translations")},
			flatNaming:  false,
			baseLang:    "",
			fileExt:     []string{""},
			namePattern: "**/*.JSON",
			expected:    []string{},
		},
		{
			name: "Multiple valid roots with custom pattern",
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
			namePattern: "es/**/custom_*.json",
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
		{
			name:       "Duplicate roots and duplicate extensions are deduped",
			paths:      []string{filepath.Join(baseTestDir, "flat/translations"), filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json", "json"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name:       "Empty root entries are skipped",
			paths:      []string{"", filepath.Join(baseTestDir, "flat/translations")},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "flat/translations/en.json"),
			},
		},
		{
			name: "Nested naming across multiple roots",
			paths: []string{
				filepath.Join(baseTestDir, "multiple/dir1"),
				filepath.Join(baseTestDir, "multiple/dir2"),
				filepath.Join(baseTestDir, "multiple/dir3"),
			},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(baseTestDir, "multiple/dir1/en/file1.json"),
				filepath.Join(baseTestDir, "multiple/dir2/en/file2.json"),
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
				t.Fatalf("expected files %v, got %v", expectedNormalized, actualNormalized)
			}
		})
	}
}

func TestFindAllTranslationFiles_ReturnsSortedOutput(t *testing.T) {
	paths := []string{filepath.Join(baseTestDir, "flat/translations")}

	got, err := findAllTranslationFiles(paths, true, "en", []string{"yaml", "json"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got = normalizePaths(got)
	want := normalizePaths([]string{
		filepath.Join(baseTestDir, "flat/translations/en.json"),
		filepath.Join(baseTestDir, "flat/translations/en.yaml"),
	})

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sorted files %v, got %v", want, got)
	}
}

func normalizePaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = filepath.ToSlash(filepath.Clean(p))
	}
	return normalized
}
