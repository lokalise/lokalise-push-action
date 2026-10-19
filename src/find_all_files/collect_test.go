package main

import (
	"path/filepath"
	"slices"
	"testing"
)

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
			name:     "does not match uppercase extension case-insensitively",
			filename: "file.JSON",
			fileExts: []string{"json"},
			want:     false,
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
		{
			name:     "matches extension in multi-dot filename",
			filename: "archive.tar.gz",
			fileExts: []string{"gz"},
			want:     true,
		},
		{
			name:     "matches compound extension",
			filename: "archive.tar.gz",
			fileExts: []string{"tar.gz"},
			want:     true,
		},
		{
			name:     "does not match different compound extension",
			filename: "archive.tar.gz",
			fileExts: []string{"zip.gz"},
			want:     false,
		},
		{
			name:     "does not match non-last extension in multi-dot filename",
			filename: "archive.tar.gz",
			fileExts: []string{"tar"},
			want:     false,
		},
		{
			name:     "extension list without dot still matches",
			filename: "file.json",
			fileExts: []string{"json"},
			want:     true,
		},
		{
			name:     "empty extension list does not match anything",
			filename: "file.json",
			fileExts: []string{},
			want:     false,
		},
		{
			name:     "extension list with dot does not match (invalid input)",
			filename: "file.json",
			fileExts: []string{".json"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hasMatchingExtension(tt.filename, tt.fileExts)
			if got != tt.want {
				t.Fatalf("hasMatchingExtension(%q, %v) = %v, want %v", tt.filename, tt.fileExts, got, tt.want)
			}
		})
	}
}

func TestFileCollector(t *testing.T) {
	t.Parallel()

	collector := newFileCollector()

	collector.add(filepath.Join("z", "file.json"))
	collector.add(filepath.Join("a", "file.json"))
	collector.add(filepath.Join("z", "file.json"))

	got := collector.sorted()
	want := []string{
		"a/file.json",
		"z/file.json",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("sorted() = %v, want %v", got, want)
	}
}

func TestCollectFilesByPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		root        string
		namePattern string
		want        []string
		wantErr     bool
	}{
		{
			name:        "matches file relative to root",
			root:        filepath.Join(baseTestDir, "pattern-only"),
			namePattern: "sub/custom_*.json",
			want: []string{
				filepath.Join(baseTestDir, "pattern-only", "sub", "custom_name.json"),
			},
		},
		{
			name:        "rejects pattern escaping root",
			root:        filepath.Join(baseTestDir, "pattern-only"),
			namePattern: "../en.json",
			wantErr:     true,
		},
		{
			name:        "matches recursively",
			root:        filepath.Join(baseTestDir, "locales"),
			namePattern: "**/custom_*.json",
			want: []string{
				filepath.Join(baseTestDir, "locales", "en", "sub1", "custom_abc.json"),
			},
		},
		{
			name:        "accepts leading dot slash",
			root:        filepath.Join(baseTestDir, "pattern-only"),
			namePattern: "./sub/custom_*.json",
			want: []string{
				filepath.Join(baseTestDir, "pattern-only", "sub", "custom_name.json"),
			},
		},
		{
			name:        "returns no files when pattern has no matches",
			root:        filepath.Join(baseTestDir, "pattern-only"),
			namePattern: "**/*.yaml",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []string

			err := collectFilesByPattern(
				tt.root,
				tt.namePattern,
				func(path string) {
					got = append(got, path)
				},
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("collectFilesByPattern() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("collectFilesByPattern() error = %v", err)
			}

			assertPathsEqual(t, got, tt.want)
		})
	}
}

func TestCollectFlatFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		root     string
		baseLang string
		fileExts []string
		want     []string
	}{
		{
			name:     "collects matching base language files",
			root:     filepath.Join(baseTestDir, "flat", "translations"),
			baseLang: "en",
			fileExts: []string{"json", "yaml"},
			want: []string{
				filepath.Join(baseTestDir, "flat", "translations", "en.json"),
				filepath.Join(baseTestDir, "flat", "translations", "en.yaml"),
			},
		},
		{
			name:     "does not match similar language name",
			root:     filepath.Join(baseTestDir, "flat", "translations"),
			baseLang: "en-US",
			fileExts: []string{"json"},
			want: []string{
				filepath.Join(baseTestDir, "flat", "translations", "en-US.json"),
			},
		},
		{
			name:     "ignores unsupported extensions",
			root:     filepath.Join(baseTestDir, "flat", "translations"),
			baseLang: "en",
			fileExts: []string{"txt"},
			want:     nil,
		},
		{
			name:     "missing directory is ignored",
			root:     filepath.Join(baseTestDir, "does-not-exist"),
			baseLang: "en",
			fileExts: []string{"json"},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []string

			err := collectFlatFiles(
				tt.root,
				tt.baseLang,
				tt.fileExts,
				func(path string) {
					got = append(got, path)
				},
			)
			if err != nil {
				t.Fatalf("collectFlatFiles() error = %v", err)
			}

			assertPathsEqual(t, got, tt.want)
		})
	}
}

func TestCollectNestedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		root     string
		baseLang string
		fileExts []string
		want     []string
	}{
		{
			name:     "collects matching files recursively",
			root:     filepath.Join(baseTestDir, "nested"),
			baseLang: "en",
			fileExts: []string{"json", "yaml"},
			want: []string{
				filepath.Join(baseTestDir, "nested", "en", "file1.json"),
				filepath.Join(baseTestDir, "nested", "en", "file2.json"),
				filepath.Join(baseTestDir, "nested", "en", "deeper", "file4.json"),
			},
		},
		{
			name:     "does not collect files from another language",
			root:     filepath.Join(baseTestDir, "nested"),
			baseLang: "es",
			fileExts: []string{"json"},
			want: []string{
				filepath.Join(baseTestDir, "nested", "es", "file1.json"),
			},
		},
		{
			name:     "filters by extension",
			root:     filepath.Join(baseTestDir, "nested"),
			baseLang: "en",
			fileExts: []string{"yaml"},
			want:     nil,
		},
		{
			name:     "matches uppercase extension",
			root:     filepath.Join(baseTestDir, "nested"),
			baseLang: "en",
			fileExts: []string{"YAML"},
			want: []string{
				filepath.Join(baseTestDir, "nested", "en", "file3.YAML"),
			},
		},
		{
			name:     "missing language directory is ignored",
			root:     filepath.Join(baseTestDir, "nested"),
			baseLang: "de",
			fileExts: []string{"json"},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []string

			err := collectNestedFiles(
				tt.root,
				tt.baseLang,
				tt.fileExts,
				func(path string) {
					got = append(got, path)
				},
			)
			if err != nil {
				t.Fatalf("collectNestedFiles() error = %v", err)
			}

			assertPathsEqual(t, got, tt.want)
		})
	}
}

func assertPathsEqual(t *testing.T, got, want []string) {
	t.Helper()

	got = normalizedSortedPaths(got)
	want = normalizedSortedPaths(want)

	if !slices.Equal(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func normalizedSortedPaths(paths []string) []string {
	result := make([]string, len(paths))

	for i, path := range paths {
		result[i] = filepath.ToSlash(path)
	}

	slices.Sort(result)

	return result
}
