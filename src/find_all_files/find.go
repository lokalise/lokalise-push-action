package main

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// findAllTranslationFiles scans each configured root using the chosen strategy.
//
// Rules:
//   - NAME_PATTERN (if provided) overrides layout rules and is treated as a glob under the root.
//   - Flat: collect "<root>/<baseLang>.<ext>" if present.
//   - Nested: walk "<root>/<baseLang>" and collect files ending with ".<ext>".
//   - EXCLUDE_PATTERNS are applied relative to each root after files are matched.
func findAllTranslationFiles(
	paths []string,
	flatNaming bool,
	baseLang string,
	fileExts []string,
	namePattern string,
	excludePatterns []string,
) ([]string, error) {
	collector := newFileCollector()

	for _, root := range paths {
		if root == "" {
			continue
		}

		add := func(filePath string) {
			relativePath, err := filepath.Rel(root, filePath)
			if err != nil {
				return
			}

			relativePath = filepath.ToSlash(relativePath)

			if matchesExcludePattern(relativePath, excludePatterns) {
				return
			}

			collector.add(filePath)
		}

		var err error

		switch {
		case namePattern != "":
			err = collectFilesByPattern(root, namePattern, add)

		case flatNaming:
			err = collectFlatFiles(root, baseLang, fileExts, add)

		default:
			err = collectNestedFiles(root, baseLang, fileExts, add)
		}

		if err != nil {
			return nil, fmt.Errorf(
				"cannot collect translation files under %q: %w",
				root,
				err,
			)
		}
	}

	return collector.sorted(), nil
}

func matchesExcludePattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := doublestar.Match(pattern, path)
		if matched {
			return true
		}
	}

	return false
}
