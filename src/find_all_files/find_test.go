package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFindAllTranslationFiles(t *testing.T) {
	tests := []struct {
		name            string
		paths           []string
		flatNaming      bool
		baseLang        string
		fileExt         []string
		namePattern     string
		excludePatterns []string
		expected        []string
		shouldError     bool
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
		{
			name: "Exclude patterns can exclude all matched files",
			paths: []string{
				filepath.Join(baseTestDir, "resx/Resources"),
			},
			namePattern:     "*.resx",
			excludePatterns: []string{"*.resx"},
			expected:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := findAllTranslationFiles(
				tt.paths,
				tt.flatNaming,
				tt.baseLang,
				tt.fileExt,
				tt.namePattern,
				tt.excludePatterns,
			)

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

			if !slices.Equal(actualNormalized, expectedNormalized) {
				t.Fatalf("expected files %v, got %v", expectedNormalized, actualNormalized)
			}
		})
	}

	t.Run("excludes localized RESX files", func(t *testing.T) {
		root := t.TempDir()

		for _, name := range []string{
			"Account.resx",
			"Account.de-DE.resx",
			"Account.fr-FR.resx",
			"General.resx",
			"General.de-DE.resx",
			"General.fr-FR.resx",
		} {
			writeTestFile(t, filepath.Join(root, name))
		}

		actual, err := findAllTranslationFiles(
			[]string{root},
			false,
			"en",
			[]string{"resx"},
			"*.resx",
			[]string{"*.de-DE.resx", "*.fr-FR.resx"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			filepath.Join(root, "Account.resx"),
			filepath.Join(root, "General.resx"),
		}

		if !slices.Equal(
			normalizePaths(actual),
			normalizePaths(expected),
		) {
			t.Fatalf("expected files %v, got %v", expected, actual)
		}
	})

	t.Run("exclude patterns are relative to translation root", func(t *testing.T) {
		root := t.TempDir()

		for _, name := range []string{
			"Account.resx",
			"Account.de-DE.resx",
			"nested/Account.resx",
			"nested/Account.de-DE.resx",
		} {
			writeTestFile(t, filepath.Join(root, name))
		}

		actual, err := findAllTranslationFiles(
			[]string{root},
			false,
			"en",
			[]string{"resx"},
			"**/*.resx",
			[]string{"nested/*.de-DE.resx"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			filepath.Join(root, "Account.resx"),
			filepath.Join(root, "Account.de-DE.resx"),
			filepath.Join(root, "nested/Account.resx"),
		}

		actual = normalizePaths(actual)
		expected = normalizePaths(expected)
		slices.Sort(actual)
		slices.Sort(expected)

		if !slices.Equal(actual, expected) {
			t.Fatalf("expected files %v, got %v", expected, actual)
		}
	})

	t.Run("exclude patterns apply to nested layout", func(t *testing.T) {
		root := t.TempDir()

		writeTestFile(t, filepath.Join(root, "en", "common.json"))
		writeTestFile(t, filepath.Join(root, "en", "ignored.json"))

		actual, err := findAllTranslationFiles(
			[]string{root},
			false,
			"en",
			[]string{"json"},
			"",
			[]string{"en/ignored.json"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			filepath.Join(root, "en", "common.json"),
		}

		if !slices.Equal(
			normalizePaths(actual),
			normalizePaths(expected),
		) {
			t.Fatalf("expected files %v, got %v", expected, actual)
		}
	})
}

func TestFindAllTranslationFiles_ReturnsSortedOutput(t *testing.T) {
	t.Parallel()

	paths := []string{filepath.Join(baseTestDir, "flat/translations")}

	got, err := findAllTranslationFiles(paths, true, "en", []string{"yaml", "json"}, "", []string{""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got = normalizePaths(got)
	want := normalizePaths([]string{
		filepath.Join(baseTestDir, "flat/translations/en.json"),
		filepath.Join(baseTestDir, "flat/translations/en.yaml"),
	})

	slices.Sort(want)

	if !slices.Equal(got, want) {
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

func writeTestFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}
