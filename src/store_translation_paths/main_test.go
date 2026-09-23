package main

import (
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestRunWith(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		wantCfg := envConfig{
			Paths:           []string{"translations"},
			BaseLang:        "en",
			FileExts:        []string{"json"},
			NamePattern:     "",
			ExcludePatterns: []string{"*.de-DE.json"},
			FlatNaming:      true,
		}

		validateCalled := false
		storePathsCalled := false
		storeExcludedCalled := false

		createdFiles := make(map[string]*os.File)
		closedFiles := make(map[*os.File]bool)

		validate := func() (envConfig, error) {
			validateCalled = true
			return wantCfg, nil
		}

		createFile := func(name string) (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), name+"-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			createdFiles[name] = f
			return f, nil
		}

		storePaths := func(cfg envConfig, writer io.Writer) error {
			storePathsCalled = true
			assertConfigEqual(t, cfg, wantCfg)

			if writer != createdFiles[pathsFileName] {
				t.Fatalf(
					"paths writer mismatch. want=%v got=%v",
					createdFiles[pathsFileName],
					writer,
				)
			}

			return nil
		}

		storeExcluded := func(cfg envConfig, writer io.Writer) error {
			storeExcludedCalled = true
			assertConfigEqual(t, cfg, wantCfg)

			if writer != createdFiles[excludePathsFileName] {
				t.Fatalf(
					"excluded paths writer mismatch. want=%v got=%v",
					createdFiles[excludePathsFileName],
					writer,
				)
			}

			return nil
		}

		closeFile := func(file *os.File) error {
			closedFiles[file] = true
			return file.Close()
		}

		err := runWith(
			validate,
			createFile,
			storePaths,
			storeExcluded,
			closeFile,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !validateCalled {
			t.Fatal("validate was not called")
		}

		if !storePathsCalled {
			t.Fatal("storePaths was not called")
		}

		if !storeExcludedCalled {
			t.Fatal("storeExcluded was not called")
		}

		if len(createdFiles) != 2 {
			t.Fatalf("expected 2 files to be created, got %d", len(createdFiles))
		}

		for name, file := range createdFiles {
			if !closedFiles[file] {
				t.Fatalf("file %q was not closed", name)
			}
		}
	})

	t.Run("returns validate error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (envConfig, error) {
			return envConfig{}, errors.New("bad env")
		}

		createFile := func(string) (*os.File, error) {
			t.Fatal("createFile should not be called")
			return nil, nil
		}

		store := func(envConfig, io.Writer) error {
			t.Fatal("store should not be called")
			return nil
		}

		closeFile := func(*os.File) error {
			t.Fatal("closeFile should not be called")
			return nil
		}

		err := runWith(
			validate,
			createFile,
			store,
			store,
			closeFile,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "bad env") {
			t.Fatalf(
				"expected error containing %q, got %q",
				"bad env",
				err.Error(),
			)
		}
	})

	t.Run("wraps translation paths file creation error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (envConfig, error) {
			return envConfig{
				Paths:      []string{"translations"},
				BaseLang:   "en",
				FileExts:   []string{"json"},
				FlatNaming: true,
			}, nil
		}

		createFile := func(name string) (*os.File, error) {
			if name != pathsFileName {
				t.Fatalf("unexpected file requested: %q", name)
			}

			return nil, errors.New("permission denied")
		}

		store := func(envConfig, io.Writer) error {
			t.Fatal("store should not be called")
			return nil
		}

		closeFile := func(*os.File) error {
			t.Fatal("closeFile should not be called")
			return nil
		}

		err := runWith(
			validate,
			createFile,
			store,
			store,
			closeFile,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "cannot create translation paths output file") {
			t.Fatalf(
				"expected translation paths creation error, got %q",
				err.Error(),
			)
		}

		if !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf(
				"expected error containing %q, got %q",
				"permission denied",
				err.Error(),
			)
		}
	})

	t.Run("wraps excluded paths file creation error and closes translation paths file", func(t *testing.T) {
		t.Parallel()

		validate := func() (envConfig, error) {
			return envConfig{
				Paths:      []string{"translations"},
				BaseLang:   "en",
				FileExts:   []string{"json"},
				FlatNaming: true,
			}, nil
		}

		var pathsFile *os.File
		closeCalled := false

		createFile := func(name string) (*os.File, error) {
			switch name {
			case pathsFileName:
				f, err := os.CreateTemp(t.TempDir(), "paths-*")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}

				pathsFile = f
				return f, nil

			case excludePathsFileName:
				return nil, errors.New("permission denied")

			default:
				t.Fatalf("unexpected file requested: %q", name)
				return nil, nil
			}
		}

		store := func(envConfig, io.Writer) error {
			t.Fatal("store should not be called")
			return nil
		}

		closeFile := func(file *os.File) error {
			if file != pathsFile {
				t.Fatalf("unexpected file passed to closeFile")
			}

			closeCalled = true
			return file.Close()
		}

		err := runWith(
			validate,
			createFile,
			store,
			store,
			closeFile,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "cannot create excluded paths output file") {
			t.Fatalf(
				"expected excluded paths creation error, got %q",
				err.Error(),
			)
		}

		if !closeCalled {
			t.Fatal("translation paths file was not closed")
		}
	})

	t.Run("wraps translation paths store error and closes both files", func(t *testing.T) {
		t.Parallel()

		wantCfg := envConfig{
			Paths:      []string{"translations"},
			BaseLang:   "en",
			FileExts:   []string{"json"},
			FlatNaming: true,
		}

		storeErr := errors.New("disk full")

		createdFiles := make(map[string]*os.File)
		closedFiles := make(map[*os.File]bool)

		validate := func() (envConfig, error) {
			return wantCfg, nil
		}

		createFile := func(name string) (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), name+"-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			createdFiles[name] = f
			return f, nil
		}

		storePaths := func(cfg envConfig, writer io.Writer) error {
			assertConfigEqual(t, cfg, wantCfg)

			if writer != createdFiles[pathsFileName] {
				t.Fatalf("unexpected writer")
			}

			return storeErr
		}

		storeExcluded := func(envConfig, io.Writer) error {
			t.Fatal("storeExcluded should not be called")
			return nil
		}

		closeFile := func(file *os.File) error {
			closedFiles[file] = true
			return file.Close()
		}

		err := runWith(
			validate,
			createFile,
			storePaths,
			storeExcluded,
			closeFile,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, storeErr) {
			t.Fatalf("expected error wrapping %v, got %v", storeErr, err)
		}

		if !strings.Contains(err.Error(), "cannot store translation paths") {
			t.Fatalf(
				"expected error containing %q, got %q",
				"cannot store translation paths",
				err.Error(),
			)
		}

		for name, file := range createdFiles {
			if !closedFiles[file] {
				t.Fatalf("file %q was not closed", name)
			}
		}
	})

	t.Run("wraps excluded paths store error and closes both files", func(t *testing.T) {
		t.Parallel()

		wantCfg := envConfig{
			Paths:           []string{"translations"},
			BaseLang:        "en",
			FileExts:        []string{"json"},
			ExcludePatterns: []string{"*.de-DE.json"},
			FlatNaming:      true,
		}

		storeErr := errors.New("disk full")

		createdFiles := make(map[string]*os.File)
		closedFiles := make(map[*os.File]bool)

		validate := func() (envConfig, error) {
			return wantCfg, nil
		}

		createFile := func(name string) (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), name+"-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			createdFiles[name] = f
			return f, nil
		}

		storePaths := func(envConfig, io.Writer) error {
			return nil
		}

		storeExcluded := func(cfg envConfig, writer io.Writer) error {
			assertConfigEqual(t, cfg, wantCfg)

			if writer != createdFiles[excludePathsFileName] {
				t.Fatalf("unexpected writer")
			}

			return storeErr
		}

		closeFile := func(file *os.File) error {
			closedFiles[file] = true
			return file.Close()
		}

		err := runWith(
			validate,
			createFile,
			storePaths,
			storeExcluded,
			closeFile,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, storeErr) {
			t.Fatalf("expected error wrapping %v, got %v", storeErr, err)
		}

		if !strings.Contains(err.Error(), "cannot store excluded paths") {
			t.Fatalf(
				"expected error containing %q, got %q",
				"cannot store excluded paths",
				err.Error(),
			)
		}

		for name, file := range createdFiles {
			if !closedFiles[file] {
				t.Fatalf("file %q was not closed", name)
			}
		}
	})
}

func assertConfigEqual(t *testing.T, got, want envConfig) {
	t.Helper()

	if !slices.Equal(got.Paths, want.Paths) {
		t.Fatalf("paths mismatch. want=%v got=%v", want.Paths, got.Paths)
	}

	if got.BaseLang != want.BaseLang {
		t.Fatalf("baseLang mismatch. want=%q got=%q", want.BaseLang, got.BaseLang)
	}

	if !slices.Equal(got.FileExts, want.FileExts) {
		t.Fatalf("fileExts mismatch. want=%v got=%v", want.FileExts, got.FileExts)
	}

	if got.NamePattern != want.NamePattern {
		t.Fatalf(
			"namePattern mismatch. want=%q got=%q",
			want.NamePattern,
			got.NamePattern,
		)
	}

	if !slices.Equal(got.ExcludePatterns, want.ExcludePatterns) {
		t.Fatalf(
			"excludePatterns mismatch. want=%v got=%v",
			want.ExcludePatterns,
			got.ExcludePatterns,
		)
	}

	if got.FlatNaming != want.FlatNaming {
		t.Fatalf(
			"flatNaming mismatch. want=%v got=%v",
			want.FlatNaming,
			got.FlatNaming,
		)
	}
}
