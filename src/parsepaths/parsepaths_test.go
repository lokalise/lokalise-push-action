package parsepaths

import (
	"reflect"
	"testing"
)

func TestParsePaths(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		expected []string
	}{
		{
			name:     "Single path, no newline",
			envVar:   "/path/to/dir",
			expected: []string{"/path/to/dir"},
		},
		{
			name:     "Multiple paths, Unix newlines",
			envVar:   "/path/to/dir\n/another/path\n/yet/another/path",
			expected: []string{"/path/to/dir", "/another/path", "/yet/another/path"},
		},
		{
			name:     "Multiple paths, Windows newlines",
			envVar:   "/path/to/dir\r\n/another/path\r\n/yet/another/path",
			expected: []string{"/path/to/dir", "/another/path", "/yet/another/path"},
		},
		{
			name:     "Empty lines and whitespace",
			envVar:   "/path/to/dir\n\n \n/another/path\n   \n/yet/another/path\n",
			expected: []string{"/path/to/dir", "/another/path", "/yet/another/path"},
		},
		{
			name:     "Empty lines and whitespace plus whitespaces inside",
			envVar:   "/path/to/dir\n\n \n/another/path\n   \n/yet/another/path\n/path/to my/dir",
			expected: []string{"/path/to/dir", "/another/path", "/yet/another/path", "/path/to my/dir"},
		},
		{
			name:     "Only empty lines and whitespace",
			envVar:   "\n\n \n  \n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePaths(tt.envVar)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParsePaths() = %v, want %v", result, tt.expected)
			}
		})
	}
}
