package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Override exitFunc for testing.
	exitFunc = func(code int) {
		panic(fmt.Sprintf("Exit called with code %d", code))
	}

	code := m.Run()

	// Restore exitFunc after testing.
	exitFunc = os.Exit

	os.Exit(code)
}

func TestRunWith(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		wantCfg := envConfig{
			Paths:       []string{"translations"},
			BaseLang:    "en",
			FileExts:    []string{"json"},
			NamePattern: "",
			FlatNaming:  true,
		}

		validateCalled := false
		createCalled := false
		storeCalled := false
		closeCalled := false

		var createdFile *os.File

		validate := func() (envConfig, error) {
			validateCalled = true
			return wantCfg, nil
		}

		createFile := func() (*os.File, error) {
			createCalled = true

			f, err := os.CreateTemp(t.TempDir(), "pathspecs-*.txt")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			createdFile = f
			return f, nil
		}

		store := func(cfg envConfig, writer io.Writer) error {
			storeCalled = true

			if !reflect.DeepEqual(cfg, wantCfg) {
				t.Fatalf("cfg mismatch.\nwant=%#v\ngot=%#v", wantCfg, cfg)
			}
			if writer != createdFile {
				t.Fatalf("writer mismatch. want=%v got=%v", createdFile, writer)
			}

			return nil
		}

		closeFile := func(file *os.File) {
			closeCalled = true
			if file != createdFile {
				t.Fatalf("closeFile got unexpected file. want=%v got=%v", createdFile, file)
			}
			_ = file.Close()
		}

		err := runWith(validate, createFile, store, closeFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !validateCalled {
			t.Fatal("validate was not called")
		}
		if !createCalled {
			t.Fatal("createFile was not called")
		}
		if !storeCalled {
			t.Fatal("store was not called")
		}
		if !closeCalled {
			t.Fatal("closeFile was not called")
		}
	})

	t.Run("returns validate error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (envConfig, error) {
			return envConfig{}, errors.New("bad env")
		}

		createFile := func() (*os.File, error) {
			t.Fatal("createFile should not be called")
			return nil, nil
		}

		store := func(envConfig, io.Writer) error {
			t.Fatal("store should not be called")
			return nil
		}

		closeFile := func(*os.File) {
			t.Fatal("closeFile should not be called")
		}

		err := runWith(validate, createFile, store, closeFile)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "bad env") {
			t.Fatalf("expected error containing %q, got %q", "bad env", err.Error())
		}
	})

	t.Run("wraps create output file error and stops", func(t *testing.T) {
		t.Parallel()

		validate := func() (envConfig, error) {
			return envConfig{
				Paths:      []string{"translations"},
				BaseLang:   "en",
				FileExts:   []string{"json"},
				FlatNaming: true,
			}, nil
		}

		createFile := func() (*os.File, error) {
			return nil, errors.New("permission denied")
		}

		store := func(envConfig, io.Writer) error {
			t.Fatal("store should not be called")
			return nil
		}

		closeFile := func(*os.File) {
			t.Fatal("closeFile should not be called")
		}

		err := runWith(validate, createFile, store, closeFile)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "cannot create output file") {
			t.Fatalf("expected wrapped error containing %q, got %q", "cannot create output file", err.Error())
		}
		if !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("expected wrapped error containing %q, got %q", "permission denied", err.Error())
		}
	})

	t.Run("wraps store error and still closes file", func(t *testing.T) {
		t.Parallel()

		wantCfg := envConfig{
			Paths:      []string{"translations"},
			BaseLang:   "en",
			FileExts:   []string{"json"},
			FlatNaming: true,
		}

		var createdFile *os.File
		closeCalled := false

		validate := func() (envConfig, error) {
			return wantCfg, nil
		}

		createFile := func() (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), "pathspecs-*.txt")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			createdFile = f
			return f, nil
		}

		store := func(cfg envConfig, writer io.Writer) error {
			if !reflect.DeepEqual(cfg, wantCfg) {
				t.Fatalf("cfg mismatch.\nwant=%#v\ngot=%#v", wantCfg, cfg)
			}
			if writer != createdFile {
				t.Fatalf("writer mismatch. want=%v got=%v", createdFile, writer)
			}
			return errors.New("disk full")
		}

		closeFile := func(file *os.File) {
			closeCalled = true
			if file != createdFile {
				t.Fatalf("closeFile got unexpected file. want=%v got=%v", createdFile, file)
			}
			_ = file.Close()
		}

		err := runWith(validate, createFile, store, closeFile)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "cannot store translation paths") {
			t.Fatalf("expected wrapped error containing %q, got %q", "cannot store translation paths", err.Error())
		}
		if !strings.Contains(err.Error(), "disk full") {
			t.Fatalf("expected wrapped error containing %q, got %q", "disk full", err.Error())
		}
		if !closeCalled {
			t.Fatal("closeFile was not called")
		}
	})
}
