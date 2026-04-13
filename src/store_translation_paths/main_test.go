package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Override exitFunc for testing.
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	code := m.Run()

	// Restore exitFunc after testing.
	exitFunc = os.Exit

	os.Exit(code)
}

type failingWriter struct{}

func (f failingWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write failed")
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
				"TRANSLATIONS_PATH": "\npath1\n\npath2\n",
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
			name: "Root translation path",
			env: map[string]string{
				"TRANSLATIONS_PATH": ".",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"."},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
		},
		{
			name: "Name pattern with ../",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "../**/*.json",
			},
			wantPanic: true,
		},
		{
			name: "Translation path with ../",
			env: map[string]string{
				"TRANSLATIONS_PATH": "../locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "TRANSLATIONS_PATH cleans to .. fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "a/../..",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "TRANSLATIONS_PATH ./path is OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "./path",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"path"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
		},
		{
			name: "TRANSLATIONS_PATH /path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "/path",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "NamePattern glob OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "**/*.yaml",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "**/*.yaml",
		},
		{
			name: "NamePattern nested glob OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "pkg/i18n",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "en/**/custom_*.json",
			},
			wantPaths:       []string{"pkg/i18n"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: filepath.Clean("en/**/custom_*.json"),
		},
		{
			name: "FILE_EXT normalization and dedupe",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          " JSON \n.yaml\njson\n YML \n .xml ",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json", "yaml", "yml", "xml"},
			wantNamePattern: "",
		},
		{
			name: "FILE_EXT empty after normalization fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".\n \n",
				"NAME_PATTERN":      "",
			},
			wantPanic: true,
		},
		{
			name: "Absolute NAME_PATTERN fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "/tmp/file.json",
			},
			wantPanic: true,
		},
		{
			name: "Whitespace NAME_PATTERN fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "   ",
			},
			wantPanic: true,
		},
		{
			name: "Leading dots and casing in FILE_EXT normalize correctly",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".json\n.JSON\n json ",
				"NAME_PATTERN":      "",
			},
			wantPaths:       []string{"locales"},
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
		exactOrder  bool
		writer      io.Writer
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
			exactOrder: true,
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
			name:       "Root path with flat naming",
			paths:      []string{"."},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(".", ".", "en.json"),
			},
		},
		{
			name:        "Root path with custom name pattern",
			paths:       []string{"."},
			flatNaming:  false,
			baseLang:    "en",
			fileExt:     []string{"json"},
			namePattern: "some_dir/**.yaml",
			expected: []string{
				filepath.Join(".", ".", "some_dir", "**.yaml"),
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
		{
			name:       "Nested naming with root path",
			paths:      []string{"."},
			flatNaming: false,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected: []string{
				filepath.Join(".", ".", "en", "**", "*.json"),
			},
		},
		{
			name:       "Duplicate paths and duplicate extensions are deduped",
			paths:      []string{"translations", "translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json", "json"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
			},
			exactOrder: true,
		},
		{
			name:        "Name pattern overrides extension expansion",
			paths:       []string{"translations"},
			flatNaming:  true,
			baseLang:    "en",
			fileExt:     []string{"json", "yaml", "xml"},
			namePattern: "custom_name.txt",
			expected: []string{
				filepath.Join(".", "translations", "custom_name.txt"),
			},
			exactOrder: true,
		},
		{
			name:       "Extensions are sorted deterministically",
			paths:      []string{"translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"yaml", "json"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name:       "Empty extensions are skipped",
			paths:      []string{"translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json", "", "   ", "yaml"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name:        "Writer error is returned",
			paths:       []string{"translations"},
			flatNaming:  true,
			baseLang:    "en",
			fileExt:     []string{"json"},
			shouldError: true,
			writer:      failingWriter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			writer := tt.writer
			if writer == nil {
				writer = &buf
			}

			err := storeTranslationPaths(tt.paths, tt.flatNaming, tt.baseLang, tt.fileExt, tt.namePattern, writer)

			if tt.shouldError {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			lines := normalizeLines(strings.Split(strings.TrimSpace(buf.String()), "\n"))
			expected := normalizeLines(tt.expected)

			if tt.exactOrder {
				if !reflect.DeepEqual(lines, expected) {
					t.Fatalf("unexpected lines.\nwant=%v\ngot=%v", expected, lines)
				}
				return
			}

			for _, expectedLine := range expected {
				if !slices.Contains(lines, expectedLine) {
					t.Errorf("missing expected line: %s", expectedLine)
				}
			}

			if len(lines) != len(expected) {
				t.Errorf("unexpected number of lines. expected %d, got %d", len(expected), len(lines))
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
