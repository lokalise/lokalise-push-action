package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var baseTestDir string // Shared read-only test fixture directory for this package.

func TestMain(m *testing.M) {
	// Create shared directory structure.
	dir, err := os.MkdirTemp(".", "find-all-files-test-*")
	if err != nil {
		panic(err)
	}

	relDir, err := filepath.Rel(".", dir)
	if err != nil {
		panic(err)
	}

	baseTestDir = relDir

	err = setupTestFileStructure(baseTestDir)
	if err != nil {
		panic(err)
	}

	// Override exitFunc for testing.
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	code := m.Run()

	// Cleanup.
	err = os.RemoveAll(baseTestDir)
	if err != nil {
		log.Printf("Failed to remove %s: %v", baseTestDir, err)
	}

	// Restore exitFunc after testing.
	exitFunc = os.Exit
	os.Exit(code)
}

func setupTestFileStructure(baseDir string) error {
	dirs := []string{
		"flat/translations",
		"nested/en",
		"nested/en/deeper",
		"nested/es",
		"empty",
		"special chars dir",
		"multiple/dir1/en",
		"multiple/dir2/en",
		"multiple/dir3/es",
		"locales/en/sub1",
		"locales/fr",
		"i18n/en/sub2",
		"dup/locales/en/sub",
		"pattern-only/sub",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"flat/translations/en.json":       "{}",
		"flat/translations/en.yaml":       "{}",
		"flat/translations/en-US.json":    "{}",
		"flat/translations/fr.json":       "{}",
		"flat/translations/unrelated.txt": "skip",

		"nested/en/file1.json":        "{}",
		"nested/en/file2.json":        "{}",
		"nested/en/file3.YAML":        "{}",
		"nested/en/deeper/file4.json": "{}",
		"nested/es/file1.json":        "{}",

		"special chars dir/en-US.json": "{}",

		"multiple/dir1/en/file1.json": "{}",
		"multiple/dir2/en/file2.json": "{}",
		"multiple/dir3/es/file3.json": "{}",

		"locales/en/sub1/custom_abc.json": "{}",
		"locales/fr/whatever.json":        "{}",
		"i18n/en/sub2/custom_xyz.json":    "{}",

		"dup/locales/en/sub/shared.json": "{}",

		"pattern-only/sub/custom_name.json": "{}",

		"en.json": "{}",
	}

	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
