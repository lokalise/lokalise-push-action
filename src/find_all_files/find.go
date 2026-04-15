package main

import (
	"fmt"
	"os"
)

// findAllTranslationFiles scans each configured root using the chosen strategy.
// Rules:
//   - NAME_PATTERN (if provided) overrides layout rules and is treated as a glob under the root.
//   - Flat:   collect "<root>/<baseLang>.<ext>" if present.
//   - Nested: walk "<root>/<baseLang>" and collect files ending with ".<ext>".
func findAllTranslationFiles(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string) ([]string, error) {
	collector := newFileCollector()

	for _, root := range paths {
		if root == "" {
			continue
		}

		var err error
		switch {
		case namePattern != "":
			err = collectFilesByPattern(root, namePattern, collector.add)
		case flatNaming:
			err = collectFlatFiles(root, baseLang, fileExts, collector.add)
		default:
			err = collectNestedFiles(root, baseLang, fileExts, collector.add)
		}

		if err != nil {
			return nil, err
		}
	}

	files := collector.sorted()
	fmt.Fprintf(os.Stderr, "Found %d unique files\n", len(files))

	return files, nil
}
