package main

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
)

type storePathsFunc func(cfg envConfig, writer io.Writer) error

// storeTranslationPaths emits include pathspecs for translation files.
// Output is newline-separated, ready for consumption by changed-files
// (files_from_source_file).
// Rules:
//   - If namePattern is set, it fully overrides defaults and is written once per root.
//     The pattern may include globs (e.g., "**/*.yaml") and/or a concrete filename.
//   - If flatNaming is true  -> "<root>/<baseLang>.<ext>"
//   - If flatNaming is false -> "<root>/<baseLang>/**/*.ext"
func storeTranslationPaths(cfg envConfig, writer io.Writer) error {
	seen := make(map[string]struct{}) // avoid duplicates across roots/exts

	// Sort extensions for stable output independent of input order,
	// while preserving root order.
	exts := slices.Clone(cfg.FileExts)
	slices.Sort(exts)

	for _, root := range cfg.Paths {
		if cfg.NamePattern != "" {
			// Custom pattern takes precedence; caller is responsible for including
			// filename/ext or globs. We don't expand it per-extension.
			if err := writeUniqueLine(writer, seen, filepath.Join(root, cfg.NamePattern)); err != nil {
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

			pattern := buildTranslationPattern(root, cfg.FlatNaming, cfg.BaseLang, ext)
			if err := writeUniqueLine(writer, seen, pattern); err != nil {
				return err
			}
		}
	}

	return nil
}

// storeExcludedPaths emits exclude pathspecs relative to each configured root.
//
// EXCLUDE_PATTERNS are interpreted relative to every TRANSLATIONS_PATH entry.
// Output is newline-separated, ready for consumption by changed-files
// (files_ignore_from_source_file).
func storeExcludedPaths(cfg envConfig, writer io.Writer) error {
	seen := make(map[string]struct{})

	patterns := slices.Clone(cfg.ExcludePatterns)
	slices.Sort(patterns)

	for _, root := range cfg.Paths {
		for _, pattern := range patterns {
			if pattern == "" {
				continue
			}

			if err := writeUniqueLine(
				writer,
				seen,
				filepath.Join(root, filepath.FromSlash(pattern)),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildTranslationPattern builds the pathspec for a single root/extension pair.
func buildTranslationPattern(root string, flatNaming bool, baseLang, ext string) string {
	if flatNaming {
		return filepath.Join(root, baseLang+"."+ext)
	}

	return filepath.Join(root, baseLang, "**", "*."+ext)
}
