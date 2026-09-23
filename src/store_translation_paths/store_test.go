package main

import (
	"bytes"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestStoreTranslationPaths(t *testing.T) {
	tests := []struct {
		name        string
		cfg         envConfig
		expected    []string
		shouldError bool
		exactOrder  bool
		writer      io.Writer
	}{
		{
			name: "Flat naming with valid paths",
			cfg: envConfig{
				Paths:      []string{"translations", "more_translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "more_translations", "en.json"),
			},
		},
		{
			name: "Flat naming with valid path and multiple exts",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json", "yaml"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name: "Duplicate extensions are deduped and sorted deterministically",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"yaml", "json", "yaml", "json"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name: "No paths produces no output",
			cfg: envConfig{
				Paths:      []string{},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json"},
			},
			expected:   []string{},
			exactOrder: true,
		},
		{
			name: "No file extensions and no name pattern produce no output",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{},
			},
			expected:   []string{},
			exactOrder: true,
		},
		{
			name: "Name pattern override works with empty extensions",
			cfg: envConfig{
				Paths:       []string{"translations"},
				FlatNaming:  true,
				BaseLang:    "en",
				FileExts:    []string{},
				NamePattern: "**/*.yaml",
			},
			expected: []string{
				filepath.Join(".", "translations", "**", "*.yaml"),
			},
			exactOrder: true,
		},
		{
			name: "Custom naming pattern",
			cfg: envConfig{
				Paths:       []string{"translations", "more_translations"},
				FlatNaming:  true,
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "custom_name.json",
			},
			expected: []string{
				filepath.Join(".", "translations", "custom_name.json"),
				filepath.Join(".", "more_translations", "custom_name.json"),
			},
		},
		{
			name: "Nested naming with custom pattern",
			cfg: envConfig{
				Paths:       []string{"translations", "translations"},
				FlatNaming:  false,
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "**.yaml",
			},
			expected: []string{
				filepath.Join(".", "translations", "**.yaml"),
			},
		},
		{
			name: "Flat naming with nested paths",
			cfg: envConfig{
				Paths:      []string{"dir1/dir2/dir3", "another/nested/dir"},
				FlatNaming: true,
				BaseLang:   "fr",
				FileExts:   []string{"xml"},
			},
			expected: []string{
				filepath.Join(".", "dir1", "dir2", "dir3", "fr.xml"),
				filepath.Join(".", "another", "nested", "dir", "fr.xml"),
			},
		},
		{
			name: "Nested naming with nested paths",
			cfg: envConfig{
				Paths:      []string{"dir1/dir2/dir3", "another/nested/dir"},
				FlatNaming: false,
				BaseLang:   "de",
				FileExts:   []string{"properties"},
			},
			expected: []string{
				filepath.Join(".", "dir1", "dir2", "dir3", "de", "**", "*.properties"),
				filepath.Join(".", "another", "nested", "dir", "de", "**", "*.properties"),
			},
		},
		{
			name: "Root path with flat naming",
			cfg: envConfig{
				Paths:      []string{"."},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json"},
			},
			expected: []string{
				filepath.Join(".", ".", "en.json"),
			},
		},
		{
			name: "Root path with custom name pattern",
			cfg: envConfig{
				Paths:       []string{"."},
				FlatNaming:  false,
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "some_dir/**.yaml",
			},
			expected: []string{
				filepath.Join(".", ".", "some_dir", "**.yaml"),
			},
		},
		{
			name: "Complex custom name pattern",
			cfg: envConfig{
				Paths:       []string{"translations"},
				FlatNaming:  false,
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "en/**/custom_*.json",
			},
			expected: []string{
				filepath.Join(".", "translations", "en", "**", "custom_*.json"),
			},
		},
		{
			name: "Nested naming with root path",
			cfg: envConfig{
				Paths:      []string{"."},
				FlatNaming: false,
				BaseLang:   "en",
				FileExts:   []string{"json"},
			},
			expected: []string{
				filepath.Join(".", ".", "en", "**", "*.json"),
			},
		},
		{
			name: "Duplicate paths and duplicate extensions are deduped",
			cfg: envConfig{
				Paths:      []string{"translations", "translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json", "json"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
			},
			exactOrder: true,
		},
		{
			name: "Name pattern overrides extension expansion",
			cfg: envConfig{
				Paths:       []string{"translations"},
				FlatNaming:  true,
				BaseLang:    "en",
				FileExts:    []string{"json", "yaml", "xml"},
				NamePattern: "custom_name.txt",
			},
			expected: []string{
				filepath.Join(".", "translations", "custom_name.txt"),
			},
			exactOrder: true,
		},
		{
			name: "Extensions are sorted deterministically",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"yaml", "json"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name: "Empty extensions are skipped",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json", "", "   ", "yaml"},
			},
			expected: []string{
				filepath.Join(".", "translations", "en.json"),
				filepath.Join(".", "translations", "en.yaml"),
			},
			exactOrder: true,
		},
		{
			name: "Writer error is returned",
			cfg: envConfig{
				Paths:      []string{"translations"},
				FlatNaming: true,
				BaseLang:   "en",
				FileExts:   []string{"json"},
			},
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

			err := storeTranslationPaths(tt.cfg, writer)

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
				if !slices.Equal(lines, expected) {
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

func TestStoreExcludedPaths(t *testing.T) {
	tests := []struct {
		name string
		cfg  envConfig
		want string
	}{
		{
			name: "Stores exclude patterns for every root",
			cfg: envConfig{
				Paths: []string{
					"src/AuditService/Resources",
					"src/WebHost/Resources",
				},
				ExcludePatterns: []string{
					"*.fr-FR.resx",
					"*.de-DE.resx",
				},
			},
			want: "" +
				"src/AuditService/Resources/*.de-DE.resx\n" +
				"src/AuditService/Resources/*.fr-FR.resx\n" +
				"src/WebHost/Resources/*.de-DE.resx\n" +
				"src/WebHost/Resources/*.fr-FR.resx\n",
		},
		{
			name: "Empty exclude patterns produce no output",
			cfg: envConfig{
				Paths:           []string{"translations"},
				ExcludePatterns: nil,
			},
			want: "",
		},
		{
			name: "Empty patterns are skipped",
			cfg: envConfig{
				Paths: []string{"translations"},
				ExcludePatterns: []string{
					"",
					"*.de-DE.resx",
				},
			},
			want: "translations/*.de-DE.resx\n",
		},
		{
			name: "Nested glob is preserved",
			cfg: envConfig{
				Paths: []string{"translations"},
				ExcludePatterns: []string{
					"generated/**/*.json",
				},
			},
			want: "translations/generated/**/*.json\n",
		},
		{
			name: "Duplicate pathspecs are written once",
			cfg: envConfig{
				Paths: []string{
					"translations",
					"translations",
				},
				ExcludePatterns: []string{
					"*.de-DE.resx",
					"*.de-DE.resx",
				},
			},
			want: "translations/*.de-DE.resx\n",
		},
		{
			name: "Pattern output is sorted within each root",
			cfg: envConfig{
				Paths: []string{"translations"},
				ExcludePatterns: []string{
					"*.sv-SE.resx",
					"*.de-DE.resx",
					"*.fr-FR.resx",
				},
			},
			want: "" +
				"translations/*.de-DE.resx\n" +
				"translations/*.fr-FR.resx\n" +
				"translations/*.sv-SE.resx\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder

			err := storeExcludedPaths(tt.cfg, &buf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := buf.String(); got != tt.want {
				t.Fatalf("output mismatch.\nwant:\n%q\ngot:\n%q", tt.want, got)
			}
		})
	}
}

func TestStoreExcludedPathsReturnsWriteError(t *testing.T) {
	cfg := envConfig{
		Paths: []string{"translations"},
		ExcludePatterns: []string{
			"*.de-DE.resx",
		},
	}

	err := storeExcludedPaths(cfg, failingWriter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf(
			"expected error containing %q, got %q",
			"write failed",
			err.Error(),
		)
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
