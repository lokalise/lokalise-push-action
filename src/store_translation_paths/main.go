package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	// Read and validate inputs from the environment.
	// This step makes sure we have enough info to derive a set of pathspecs.
	translationsPaths, baseLang, fileExts, namePattern := validateEnvironment()

	// FLAT_NAMING determines whether translations are flat (e.g., locales/en.json)
	// or nested by language (e.g., locales/en/**/*.json).
	flatNaming, err := parsers.ParseBoolEnv("FLAT_NAMING")
	if err != nil {
		returnWithError("invalid value for FLAT_NAMING environment variable; expected true or false")
	}

	// We persist the generated pathspecs to a file that is later consumed by
	// tj-actions/changed-files via `files_from_source_file`.
	file, err := os.Create("lok_action_paths_temp.txt")
	if err != nil {
		returnWithError(fmt.Sprintf("cannot create output file: %v", err))
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Non-fatal warning; the action will likely still succeed if writes completed.
			fmt.Fprintf(os.Stderr, "Warning: failed to close file properly: %v\n", cerr)
		}
	}()

	// Emit one pathspec per line. Consumers expect newline-separated patterns.
	// Each line can be a direct file path or a glob (git pathspec-style).
	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileExts, namePattern, file); err != nil {
		returnWithError(fmt.Sprintf("cannot store translation paths: %v", err))
	}
}

// validateEnvironment reads required variables and applies simple inference.
// Returns: (paths, base language code, file extensions, optional custom name pattern).
func validateEnvironment() ([]string, string, []string, string) {
	paths, err := parsers.ParseRepoRelativePathsEnv("TRANSLATIONS_PATH")
	if err != nil {
		returnWithError(fmt.Sprintf("failed to process params: %v", err))
	}

	baseLang := os.Getenv("BASE_LANG")
	if baseLang == "" {
		returnWithError("BASE_LANG is not set or is empty")
	}

	namePattern := os.Getenv("NAME_PATTERN")

	if namePattern != "" {
		// forbid absolute / escaping
		if np, err := ensureRepoRelative(namePattern); err != nil {
			returnWithError(fmt.Sprintf("invalid NAME_PATTERN %q: %v", namePattern, err))
		} else {
			namePattern = np
		}
	}

	// Support single or multiple FILE_EXT values (newline-separated).
	exts := parsers.ParseStringArrayEnv("FILE_EXT")
	if len(exts) == 0 {
		if v := os.Getenv("FILE_FORMAT"); v != "" {
			exts = []string{v}
		}
	}
	if len(exts) == 0 {
		returnWithError("Cannot infer file extension. Make sure FILE_FORMAT or FILE_EXT environment variables are set")
	}

	// normalize + dedupe (lowercase, trim, drop leading dot)
	seen := make(map[string]struct{}, len(exts))
	norm := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(e, ".")))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		norm = append(norm, e)
	}
	if len(norm) == 0 {
		returnWithError("no valid file extensions after normalization")
	}

	return paths, baseLang, norm, namePattern
}

// storeTranslationPaths emits one pathspec per root and (if applicable) per extension.
// Output is newline-separated, ready for consumption by changed-files (files_from_source_file).
// Rules:
//   - If namePattern is set, it fully overrides defaults and is written once per root.
//     The pattern may include globs (e.g., "**/*.yaml") and/or a concrete filename.
//   - If flatNaming is true  -> "<root>/<baseLang>.<ext>"
//   - If flatNaming is false -> "<root>/<baseLang>/**/*.ext"
func storeTranslationPaths(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string, writer io.Writer) error {
	seen := make(map[string]struct{}) // avoid duplicates across roots/exts

	writeLine := func(p string) error {
		// Normalize to forward slashes for cross-platform consistency and
		// anchor to repo root with a leading "./" (helps avoid CWD surprises).
		line := filepath.ToSlash(filepath.Join(".", p))
		if _, ok := seen[line]; ok {
			return nil
		}
		seen[line] = struct{}{}
		if _, err := writer.Write([]byte(line + "\n")); err != nil {
			return err
		}
		return nil
	}

	for _, root := range paths {
		if namePattern != "" {
			// Custom pattern takes precedence; caller is responsible for including
			// filename/ext or globs. We don't expand it per-extension.
			if err := writeLine(filepath.Join(root, namePattern)); err != nil {
				return err
			}
			continue
		}

		// Generate per-extension patterns based on layout.
		exts := append([]string(nil), fileExts...)
		sort.Strings(exts)

		for _, ext := range exts {
			ext = strings.TrimSpace(ext)
			if ext == "" {
				continue
			}

			var pat string
			if flatNaming {
				// <root>/<baseLang>.<ext>
				pat = filepath.Join(root, fmt.Sprintf("%s.%s", baseLang, ext))
			} else {
				// <root>/<baseLang>/**/*.ext
				pat = filepath.Join(root, baseLang, "**", fmt.Sprintf("*.%s", ext))
			}

			if err := writeLine(pat); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureRepoRelative(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}

	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("path must be relative to repo: %q", p)
	}

	s := filepath.ToSlash(clean)

	if strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("path must be relative to repo: %q", p)
	}

	if s == ".." || strings.HasPrefix(s, "../") {
		return "", fmt.Errorf("path escapes repo root: %q", p)
	}

	return clean, nil
}

// returnWithError prints an error and exits non-zero.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
