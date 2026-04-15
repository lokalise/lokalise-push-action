package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Hijack os.Exit so tests can assert hard exits.
	exitFunc = func(code int) { panic(fmt.Sprintf("Exit called with code %d", code)) }

	code := m.Run()

	// Restore.
	exitFunc = os.Exit
	os.Exit(code)
}

func TestRunWith(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		args := []string{"lokalise_upload", "  file.json  "}
		wantCfg := UploadConfig{
			FilePath:      "file.json",
			ProjectID:     "proj",
			Token:         "token",
			LangISO:       "en",
			GitHubRefName: "main",
			UploadTimeout: 5 * time.Second,
		}

		prepareCalled := false
		validateCalled := false
		uploadCalled := false

		factory := &LokaliseFactory{}

		prepare := func(filePath string) (UploadConfig, error) {
			prepareCalled = true
			if filePath != "file.json" {
				t.Fatalf("prepare got filePath=%q, want %q", filePath, "file.json")
			}
			return wantCfg, nil
		}

		validateFn := func(cfg UploadConfig) error {
			validateCalled = true
			if cfg != wantCfg {
				t.Fatalf("validate got cfg=%#v, want %#v", cfg, wantCfg)
			}
			return nil
		}

		upload := func(ctx context.Context, cfg UploadConfig, gotFactory ClientFactory) error {
			uploadCalled = true

			if cfg != wantCfg {
				t.Fatalf("upload got cfg=%#v, want %#v", cfg, wantCfg)
			}
			if gotFactory != factory {
				t.Fatalf("upload got unexpected factory: %#v", gotFactory)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("upload context has no deadline")
			}

			return nil
		}

		err := runWith(args, prepare, validateFn, upload, factory)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !prepareCalled {
			t.Fatal("prepare was not called")
		}
		if !validateCalled {
			t.Fatal("validate was not called")
		}
		if !uploadCalled {
			t.Fatal("upload was not called")
		}
	})

	t.Run("returns parse args error and stops", func(t *testing.T) {
		t.Parallel()

		args := []string{"lokalise_upload"}

		prepare := func(string) (UploadConfig, error) {
			t.Fatal("prepare should not be called")
			return UploadConfig{}, nil
		}

		validateFn := func(UploadConfig) error {
			t.Fatal("validate should not be called")
			return nil
		}

		upload := func(context.Context, UploadConfig, ClientFactory) error {
			t.Fatal("upload should not be called")
			return nil
		}

		err := runWith(args, prepare, validateFn, upload, &LokaliseFactory{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "usage: lokalise_upload <file>") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns prepare error and stops", func(t *testing.T) {
		t.Parallel()

		args := []string{"lokalise_upload", "file.json"}

		prepare := func(filePath string) (UploadConfig, error) {
			if filePath != "file.json" {
				t.Fatalf("prepare got filePath=%q, want %q", filePath, "file.json")
			}
			return UploadConfig{}, errors.New("bad config")
		}

		validateFn := func(UploadConfig) error {
			t.Fatal("validate should not be called")
			return nil
		}

		upload := func(context.Context, UploadConfig, ClientFactory) error {
			t.Fatal("upload should not be called")
			return nil
		}

		err := runWith(args, prepare, validateFn, upload, &LokaliseFactory{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "bad config") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns validate error and stops", func(t *testing.T) {
		t.Parallel()

		args := []string{"lokalise_upload", "file.json"}

		wantCfg := UploadConfig{
			FilePath:      "file.json",
			UploadTimeout: 5 * time.Second,
		}

		prepare := func(filePath string) (UploadConfig, error) {
			return wantCfg, nil
		}

		validateFn := func(cfg UploadConfig) error {
			if cfg != wantCfg {
				t.Fatalf("validate got cfg=%#v, want %#v", cfg, wantCfg)
			}
			return errors.New("invalid upload config")
		}

		upload := func(context.Context, UploadConfig, ClientFactory) error {
			t.Fatal("upload should not be called")
			return nil
		}

		err := runWith(args, prepare, validateFn, upload, &LokaliseFactory{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid upload config") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns upload error", func(t *testing.T) {
		t.Parallel()

		args := []string{"lokalise_upload", "file.json"}

		wantCfg := UploadConfig{
			FilePath:      "file.json",
			ProjectID:     "proj",
			Token:         "token",
			LangISO:       "en",
			GitHubRefName: "main",
			UploadTimeout: 5 * time.Second,
		}

		factory := &LokaliseFactory{}

		prepare := func(string) (UploadConfig, error) {
			return wantCfg, nil
		}

		validateFn := func(cfg UploadConfig) error {
			if cfg != wantCfg {
				t.Fatalf("validate got cfg=%#v, want %#v", cfg, wantCfg)
			}
			return nil
		}

		upload := func(ctx context.Context, cfg UploadConfig, gotFactory ClientFactory) error {
			if cfg != wantCfg {
				t.Fatalf("upload got cfg=%#v, want %#v", cfg, wantCfg)
			}
			if gotFactory != factory {
				t.Fatalf("upload got unexpected factory: %#v", gotFactory)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("upload context has no deadline")
			}
			return errors.New("upload failed")
		}

		err := runWith(args, prepare, validateFn, upload, factory)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "upload failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name:    "missing CLI arg returns error",
			args:    []string{"lokalise_upload"},
			wantErr: "usage: lokalise_upload <file>",
		},
		{
			name:    "empty CLI arg returns error",
			args:    []string{"lokalise_upload", "   "},
			wantErr: "file path is empty",
		},
		{
			name: "valid CLI arg is trimmed",
			args: []string{"lokalise_upload", "  file.json  "},
			want: "file.json",
		},
		{
			name:    "too many CLI args returns error",
			args:    []string{"lokalise_upload", "file.json", "extra"},
			wantErr: "usage: lokalise_upload <file>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCLIArgs(tt.args)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseCLIArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}
