package main

import (
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
		{
			name:     "matches extension in multi-dot filename",
			filename: "archive.tar.gz",
			fileExts: []string{"gz"},
			want:     true,
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
