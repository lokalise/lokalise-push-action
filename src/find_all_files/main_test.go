package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestRunWith(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		wantCfg := config{
			Paths:       []string{"translations", "locales"},
			BaseLang:    "en",
			FileExts:    []string{"json", "yaml"},
			NamePattern: "",
			FlatNaming:  true,
		}
		wantFiles := []string{"translations/en.json", "locales/en.yaml"}

		validateCalled := false
		findCalled := false
		processCalled := false
		writeCalled := false

		validate := func() (config, error) {
			validateCalled = true
			return wantCfg, nil
		}

		find := func(paths []string, flatNaming bool, baseLang string, fileExts []string, namePattern string) ([]string, error) {
			findCalled = true

			if !reflect.DeepEqual(paths, wantCfg.Paths) {
				t.Fatalf("paths mismatch. want=%v got=%v", wantCfg.Paths, paths)
			}
			if flatNaming != wantCfg.FlatNaming {
				t.Fatalf("flatNaming mismatch. want=%v got=%v", wantCfg.FlatNaming, flatNaming)
			}
			if baseLang != wantCfg.BaseLang {
				t.Fatalf("baseLang mismatch. want=%q got=%q", wantCfg.BaseLang, baseLang)
			}
			if !reflect.DeepEqual(fileExts, wantCfg.FileExts) {
				t.Fatalf("fileExts mismatch. want=%v got=%v", wantCfg.FileExts, fileExts)
			}
			if namePattern != wantCfg.NamePattern {
				t.Fatalf("namePattern mismatch. want=%q got=%q", wantCfg.NamePattern, namePattern)
			}

			return wantFiles, nil
		}

		process := func(allFiles []string, writeOutput func(string, string) bool) error {
			processCalled = true

			if !reflect.DeepEqual(allFiles, wantFiles) {
				t.Fatalf("allFiles mismatch. want=%v got=%v", wantFiles, allFiles)
			}

			if !writeOutput("probe", "ok") {
				t.Fatal("expected writeOutput probe to succeed")
			}

			return nil
		}

		write := func(key, value string) bool {
			writeCalled = true
			return key == "probe" && value == "ok"
		}

		err := runWith(validate, find, process, write)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !validateCalled {
			t.Fatal("validate was not called")
		}
		if !findCalled {
			t.Fatal("find was not called")
		}
		if !processCalled {
			t.Fatal("process was not called")
		}
		if !writeCalled {
			t.Fatal("write was not called")
		}
	})

	t.Run("returns validation error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (config, error) {
			return config{}, errors.New("bad env")
		}

		find := func([]string, bool, string, []string, string) ([]string, error) {
			t.Fatal("find should not be called")
			return nil, nil
		}

		process := func([]string, func(string, string) bool) error {
			t.Fatal("process should not be called")
			return nil
		}

		write := func(string, string) bool {
			t.Fatal("write should not be called")
			return true
		}

		err := runWith(validate, find, process, write)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "bad env") {
			t.Fatalf("expected error containing %q, got %q", "bad env", err.Error())
		}
	})

	t.Run("wraps discovery error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (config, error) {
			return config{
				Paths:       []string{"translations"},
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "",
				FlatNaming:  false,
			}, nil
		}

		find := func([]string, bool, string, []string, string) ([]string, error) {
			return nil, errors.New("glob exploded")
		}

		process := func([]string, func(string, string) bool) error {
			t.Fatal("process should not be called")
			return nil
		}

		write := func(string, string) bool {
			t.Fatal("write should not be called")
			return true
		}

		err := runWith(validate, find, process, write)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unable to find translation files") {
			t.Fatalf("expected wrapped error containing %q, got %q", "unable to find translation files", err.Error())
		}
		if !strings.Contains(err.Error(), "glob exploded") {
			t.Fatalf("expected wrapped error containing %q, got %q", "glob exploded", err.Error())
		}
	})

	t.Run("returns process error", func(t *testing.T) {
		t.Parallel()

		wantFiles := []string{"translations/en.json"}

		validate := func() (config, error) {
			return config{
				Paths:       []string{"translations"},
				BaseLang:    "en",
				FileExts:    []string{"json"},
				NamePattern: "",
				FlatNaming:  false,
			}, nil
		}

		find := func([]string, bool, string, []string, string) ([]string, error) {
			return wantFiles, nil
		}

		process := func(allFiles []string, writeOutput func(string, string) bool) error {
			if !reflect.DeepEqual(allFiles, wantFiles) {
				t.Fatalf("allFiles mismatch. want=%v got=%v", wantFiles, allFiles)
			}
			return errors.New("cannot write ALL_FILES to GITHUB_OUTPUT")
		}

		write := func(string, string) bool {
			t.Fatal("write should not be called directly in this test")
			return true
		}

		err := runWith(validate, find, process, write)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "cannot write ALL_FILES to GITHUB_OUTPUT") {
			t.Fatalf("expected error containing %q, got %q", "cannot write ALL_FILES to GITHUB_OUTPUT", err.Error())
		}
	})
}
