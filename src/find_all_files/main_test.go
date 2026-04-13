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

var baseTestDir string // Shared base directory for all tests.

func TestMain(m *testing.M) {
	// Create shared directory structure.
	baseTestDir = "test_fs"
	err := setupTestFileStructure(baseTestDir)
	if err != nil {
		panic(err)
	}

	// Override exitFunc for testing.
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	code := m.Run()

	// Cleanup.
	err = os.RemoveAll(baseTestDir)
	if err != nil {
		log.Printf("Failed to remove %s: %v", baseTestDir, err)
	}

	// Restore exitFunc after testing.
	exitFunc = os.Exit
	os.Exit(code)
}

func setupTestFileStructure(baseDir string) error {
	dirs := []string{
		"flat/translations",
		"nested/en",
		"nested/en/deeper",
		"nested/es",
		"empty",
		"special chars dir",
		"multiple/dir1/en",
		"multiple/dir2/en",
		"multiple/dir3/es",
		"locales/en/sub1",
		"locales/fr",
		"i18n/en/sub2",
		"dup/locales/en/sub",
		"pattern-only/sub",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"flat/translations/en.json":       "{}",
		"flat/translations/en.yaml":       "{}",
		"flat/translations/en-US.json":    "{}",
		"flat/translations/fr.json":       "{}",
		"flat/translations/unrelated.txt": "skip",

		"nested/en/file1.json":        "{}",
		"nested/en/file2.json":        "{}",
		"nested/en/file3.YAML":        "{}",
		"nested/en/deeper/file4.json": "{}",
		"nested/es/file1.json":        "{}",

		"special chars dir/en-US.json": "{}",

		"multiple/dir1/en/file1.json": "{}",
		"multiple/dir2/en/file2.json": "{}",
		"multiple/dir3/es/file3.json": "{}",

		"locales/en/sub1/custom_abc.json": "{}",
		"locales/fr/whatever.json":        "{}",
		"i18n/en/sub2/custom_xyz.json":    "{}",

		"dup/locales/en/sub/shared.json": "{}",

		"pattern-only/sub/custom_name.json": "{}",

		"en.json": "{}",
	}

	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
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

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		env             map[string]string
		wantPaths       []string
		wantBaseLang    string
		wantFileExt     []string
		wantNamePattern string
		wantPanic       bool
	}{
		{
			name: "Valid environment variables",
			env: map[string]string{
				"TRANSLATIONS_PATH": "\npath1\npath2\n\n",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "custom_name.json",
			},
			wantPaths:       []string{"path1", "path2"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "custom_name.json",
		},
		{
			name: "Missing environment variables",
			env: map[string]string{
				"TRANSLATIONS_PATH": "",
				"BASE_LANG":         "",
				"FILE_EXT":          "",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "Roots are cleaned and remain relative",
			env: map[string]string{
				"TRANSLATIONS_PATH": ".\n./locales\nlocales/../locales/en/..",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{".", "locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
		},
		{
			name: "Absolute translations path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "/etc/locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "Parent escape translations path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "../locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "Name pattern glob variants are allowed",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "en/**/custom_*.json",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "en/**/custom_*.json",
		},
		{
			name: "Absolute name pattern fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "/tmp/**/*.json",
			},
			wantPanic: true,
		},
		{
			name: "File extensions are normalized and deduplicated",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          " JSON \n.yaml\njson\n YML \n .xml ",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json", "yaml", "yml", "xml"},
			wantNamePattern: "",
		},
		{
			name: "Empty file extensions after normalization fail",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".\n \n",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "Leading dots and casing in file extensions normalize correctly",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".json\n.JSON\n json ",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"TRANSLATIONS_PATH", "BASE_LANG", "FILE_EXT", "NAME_PATTERN"} {
				t.Setenv(key, tt.env[key])
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic")
					}
				}()
			}

			paths, baseLang, fileExt, namePattern := validateEnvironment()

			if tt.wantPanic {
				return
			}

			if !reflect.DeepEqual(paths, tt.wantPaths) {
				t.Fatalf("paths mismatch. want=%v got=%v", tt.wantPaths, paths)
			}
			if baseLang != tt.wantBaseLang {
				t.Fatalf("baseLang mismatch. want=%q got=%q", tt.wantBaseLang, baseLang)
			}
			if !reflect.DeepEqual(fileExt, tt.wantFileExt) {
				t.Fatalf("fileExt mismatch. want=%v got=%v", tt.wantFileExt, fileExt)
			}
			if filepath.ToSlash(namePattern) != filepath.ToSlash(tt.wantNamePattern) {
				t.Fatalf("namePattern mismatch. want=%q got=%q", tt.wantNamePattern, namePattern)
			}
		})
	}
}

func TestProcessAllFiles(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		failOnKey      string
		wantWrites     map[string]string
		wantWriteOrder []string
		wantPanic      bool
	}{
		{
			name:  "Files found",
			input: []string{"file1", "file2"},
			wantWrites: map[string]string{
				"ALL_FILES": "file1,file2",
				"has_files": "true",
			},
			wantWriteOrder: []string{"ALL_FILES", "has_files"},
		},
		{
			name:  "No files found",
			input: []string{},
			wantWrites: map[string]string{
				"has_files": "false",
			},
			wantWriteOrder: []string{"has_files"},
		},
		{
			name:      "WriteOutput fails on ALL_FILES",
			input:     []string{"file1", "file2"},
			failOnKey: "ALL_FILES",
			wantPanic: true,
		},
		{
			name:      "WriteOutput fails on has_files true",
			input:     []string{"file1", "file2"},
			failOnKey: "has_files",
			wantPanic: true,
		},
		{
			name:      "WriteOutput fails on has_files false",
			input:     []string{},
			failOnKey: "has_files",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes := make(map[string]string)
			var order []string

			mockWrite := func(key, value string) bool {
				order = append(order, key)
				if tt.failOnKey == key {
					return false
				}
				writes[key] = value
				return true
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic but got none")
					}
				}()
			}

			processAllFiles(tt.input, mockWrite)

			if tt.wantPanic {
				return
			}

			if !reflect.DeepEqual(writes, tt.wantWrites) {
				t.Fatalf("writes mismatch. want=%v got=%v", tt.wantWrites, writes)
			}
			if !reflect.DeepEqual(order, tt.wantWriteOrder) {
				t.Fatalf("write order mismatch. want=%v got=%v", tt.wantWriteOrder, order)
			}
		})
	}
}

func TestEnsureRepoRelative(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple relative",
			input: "foo/bar.json",
			want:  filepath.Clean("foo/bar.json"),
		},
		{
			name:  "trimmed relative",
			input: "  foo/bar.json  ",
			want:  filepath.Clean("foo/bar.json"),
		},
		{
			name:  "dot relative",
			input: "./foo/bar.json",
			want:  filepath.Clean("./foo/bar.json"),
		},
		{
			name:    "absolute path",
			input:   "/foo/bar.json",
			wantErr: true,
		},
		{
			name:    "parent escape",
			input:   "../foo/bar.json",
			wantErr: true,
		},
		{
			name:    "cleans to parent escape",
			input:   "a/../..",
			wantErr: true,
		},
		{
			name:    "empty after trim",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureRepoRelative(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected result. want=%q got=%q", tt.want, got)
			}
		})
	}
}

func TestHasMatchingExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		fileExts []string
		want     bool
	}{
		{
			name:     "matches lowercase extension",
			filename: "file.json",
			fileExts: []string{"json"},
			want:     true,
		},
		{
			name:     "matches uppercase extension case-insensitively",
			filename: "file.JSON",
			fileExts: []string{"json"},
			want:     true,
		},
		{
			name:     "matches one of multiple extensions",
			filename: "file.yaml",
			fileExts: []string{"json", "yaml"},
			want:     true,
		},
		{
			name:     "no extension does not match",
			filename: "file",
			fileExts: []string{"json"},
			want:     false,
		},
		{
			name:     "different extension does not match",
			filename: "file.txt",
			fileExts: []string{"json"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasMatchingExtension(tt.filename, tt.fileExts)
			if got != tt.want {
				t.Fatalf("hasMatchingExtension(%q, %v) = %v, want %v", tt.filename, tt.fileExts, got, tt.want)
			}
		})
	}
}

func normalizePaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = filepath.ToSlash(filepath.Clean(p))
	}
	return normalized
}
