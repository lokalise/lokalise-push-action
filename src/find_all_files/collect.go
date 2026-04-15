package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// fileCollector accumulates unique file paths and normalizes them to forward slashes
// to keep output deterministic across operating systems.
type fileCollector struct {
	seen  map[string]struct{}
	files []string
}

func newFileCollector() *fileCollector {
	return &fileCollector{
		seen: make(map[string]struct{}),
	}
}

func (c *fileCollector) add(path string) {
	path = filepath.ToSlash(path)
	if _, ok := c.seen[path]; ok {
		return
	}
	c.seen[path] = struct{}{}
	c.files = append(c.files, path)
}

func (c *fileCollector) sorted() []string {
	out := append([]string(nil), c.files...)
	sort.Strings(out)
	return out
}

// collectFilesByPattern applies NAME_PATTERN relative to the given root.
// The pattern is evaluated against os.DirFS("."), so it must be repo-relative
// and must not start with "./".
func collectFilesByPattern(root, namePattern string, add func(string)) error {
	pattern := filepath.ToSlash(filepath.Join(root, namePattern))
	pattern = strings.TrimPrefix(pattern, "./")

	globOpts := []doublestar.GlobOption{
		doublestar.WithFilesOnly(),
		doublestar.WithFailOnIOErrors(),
	}

	matches, err := doublestar.Glob(os.DirFS("."), pattern, globOpts...)
	if err != nil {
		return fmt.Errorf("apply name pattern %q: %w", pattern, err)
	}

	for _, match := range matches {
		add(match)
	}

	return nil
}

// collectFlatFiles checks for exact flat-layout file names:
//
//	<root>/<baseLang>.<ext>
//
// Missing files are ignored. Unexpected stat errors are returned.
func collectFlatFiles(root, baseLang string, fileExts []string, add func(string)) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error reading directory %s: %w", root, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext == "" {
			continue
		}

		base := strings.TrimSuffix(name, ext)
		if base != baseLang {
			continue
		}

		if hasMatchingExtension(name, fileExts) {
			add(filepath.Join(root, name))
		}
	}

	return nil
}

// collectNestedFiles walks the nested layout directory:
//
//	<root>/<baseLang>/...
//
// Missing language directories are treated as "no files found", not as errors.
func collectNestedFiles(root, baseLang string, fileExts []string, add func(string)) error {
	targetDir := filepath.Join(root, baseLang)

	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error accessing directory %s: %w", targetDir, err)
	}

	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(targetDir, func(fp string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error walking through directory %s: %w", targetDir, walkErr)
		}
		if d.IsDir() {
			return nil
		}
		if hasMatchingExtension(d.Name(), fileExts) {
			add(fp)
		}
		return nil
	})
}

// hasMatchingExtension reports whether the file name ends with one of the allowed extensions.
// Comparison is case-insensitive.
func hasMatchingExtension(name string, fileExts []string) bool {
	for _, ext := range fileExts {
		if strings.EqualFold(filepath.Ext(name), "."+ext) {
			return true
		}
	}
	return false
}
