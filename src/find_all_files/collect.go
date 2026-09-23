package main

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// fileCollector accumulates unique file paths and normalizes them to forward slashes
// to keep output deterministic across operating systems.
type fileCollector struct {
	seen map[string]struct{}
}

func newFileCollector() *fileCollector {
	return &fileCollector{
		seen: make(map[string]struct{}),
	}
}

func (c *fileCollector) add(path string) {
	c.seen[filepath.ToSlash(path)] = struct{}{}
}

func (c *fileCollector) sorted() []string {
	return slices.Sorted(maps.Keys(c.seen))
}

// collectFilesByPattern applies NAME_PATTERN relative to the given root.
//
// The pattern is evaluated against os.DirFS(root). Leading "./" and
// platform-specific separators are normalized before matching.
func collectFilesByPattern(
	root string,
	namePattern string,
	add func(string),
) error {
	pattern := filepath.ToSlash(namePattern)
	pattern = strings.TrimPrefix(pattern, "./")

	err := doublestar.GlobWalk(
		os.DirFS(root),
		pattern,
		func(match string, _ fs.DirEntry) error {
			add(filepath.Join(root, filepath.FromSlash(match)))
			return nil
		},
		doublestar.WithFilesOnly(),
		doublestar.WithFailOnIOErrors(),
	)
	if err != nil {
		return fmt.Errorf(
			"apply name pattern %q in %q: %w",
			pattern,
			root,
			err,
		)
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
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error reading directory %q: %w", root, err)
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
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error accessing directory %q: %w", targetDir, err)
	}

	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(targetDir, func(fp string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", fp, walkErr)
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
	fileExt := filepath.Ext(name)

	for _, ext := range fileExts {
		if strings.EqualFold(fileExt, "."+ext) {
			return true
		}
	}

	return false
}
