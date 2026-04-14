package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// storeTranslationPaths emits one pathspec per root and (if applicable) per extension.
// Output is newline-separated, ready for consumption by changed-files (files_from_source_file).
// Rules:
//   - If namePattern is set, it fully overrides defaults and is written once per root.
//     The pattern may include globs (e.g., "**/*.yaml") and/or a concrete filename.
//   - If flatNaming is true  -> "<root>/<baseLang>.<ext>"
//   - If flatNaming is false -> "<root>/<baseLang>/**/*.ext"
func storeTranslationPaths(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string, writer io.Writer) error {
	seen := make(map[string]struct{}) // avoid duplicates across roots/exts

	// Sort once to keep output deterministic across runs.
	exts := append([]string(nil), fileExts...)
	sort.Strings(exts)

	for _, root := range paths {
		if namePattern != "" {
			// Custom pattern takes precedence; caller is responsible for including
			// filename/ext or globs. We don't expand it per-extension.
			if err := writeUniqueLine(writer, seen, filepath.Join(root, namePattern)); err != nil {
				return err
			}
			continue
		}

		// Generate per-extension patterns based on layout.
		for _, ext := range exts {
			ext = strings.TrimSpace(ext)
			if ext == "" {
				continue
			}

			pattern := buildTranslationPattern(root, flatNaming, baseLang, ext)
			if err := writeUniqueLine(writer, seen, pattern); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildTranslationPattern builds the pathspec for a single root/extension pair.
func buildTranslationPattern(root string, flatNaming bool, baseLang, ext string) string {
	if flatNaming {
		// <root>/<baseLang>.<ext>
		return filepath.Join(root, fmt.Sprintf("%s.%s", baseLang, ext))
	}

	// <root>/<baseLang>/**/*.ext
	return filepath.Join(root, baseLang, "**", fmt.Sprintf("*.%s", ext))
}
