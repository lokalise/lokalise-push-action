package parsepaths

import (
	"strings"
)

// parsePaths takes a newline-separated string and returns a slice of cleaned paths.
func ParsePaths(envVar string) []string {
	paths := strings.FieldsFunc(envVar, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
	}
	return paths
}
