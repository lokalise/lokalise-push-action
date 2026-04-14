package main

import (
	"bytes"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

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
			name:       "Duplicate extensions are deduped and sorted deterministically",
			paths:      []string{"translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"yaml", "json", "yaml", "json"},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name:       "No paths produces no output",
			paths:      []string{},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{"json"},
			expected:   []string{},
			exactOrder: true,
		},
		{
			name:       "No file extensions and no name pattern produce no output",
			paths:      []string{"translations"},
			flatNaming: true,
			baseLang:   "en",
			fileExt:    []string{},
			expected:   []string{},
			exactOrder: true,
		},
		{
			name:        "Name pattern override works with empty extensions",
			paths:       []string{"translations"},
			flatNaming:  true,
			baseLang:    "en",
			fileExt:     []string{},
			namePattern: "**/*.yaml",
			expected: []string{
				filepath.Join(".", "translations", "**", "*.yaml"),
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
