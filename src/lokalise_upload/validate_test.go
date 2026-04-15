package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFile(t *testing.T) {
	tmpFile := mustWriteTempFile(t)
	tmpDir := t.TempDir()
	missingFile := filepath.Join(t.TempDir(), "missing.json")

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name: "regular file is accepted",
			path: tmpFile,
		},
		{
			name:    "missing file returns error",
			path:    missingFile,
			wantErr: "does not exist",
		},
		{
			name:    "directory returns error",
			path:    tmpDir,
			wantErr: "is a directory, not a file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFile(tt.path)

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
		})
	}
}

func TestValidate(t *testing.T) {
	validFile := mustWriteTempFile(t)

	tests := []struct {
		name    string
		cfg     UploadConfig
		wantErr string
	}{
		{
			name: "valid config passes",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
		},
		{
			name: "skip tagging allows empty GitHubRefName",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "",
				SkipTagging:   true,
			},
		},
		{
			name: "missing project id returns error",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantErr: "project ID is required",
		},
		{
			name: "directory path fails before field validation",
			cfg: UploadConfig{
				FilePath:      t.TempDir(),
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantErr: "is a directory, not a file",
		},
		{
			name: "missing token returns error",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantErr: "API token is required",
		},
		{
			name: "missing language returns error",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "",
				GitHubRefName: "ref",
			},
			wantErr: "base language (BASE_LANG) is required",
		},
		{
			name: "missing GitHubRefName returns error when tagging enabled",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "",
				SkipTagging:   false,
			},
			wantErr: "GitHub reference name (GITHUB_REF_NAME) is required",
		},
		{
			name: "missing file path returns error",
			cfg: UploadConfig{
				FilePath:      filepath.Join(t.TempDir(), "missing.json"),
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantErr: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validate(tt.cfg)

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
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     UploadConfig
		wantErr string
	}{
		{
			name: "all required fields present",
			cfg: UploadConfig{
				ProjectID: "p",
				Token:     "t",
				LangISO:   "en",
			},
		},
		{
			name: "missing project id returns error",
			cfg: UploadConfig{
				Token:   "t",
				LangISO: "en",
			},
			wantErr: "project ID is required",
		},
		{
			name: "missing token returns error",
			cfg: UploadConfig{
				ProjectID: "p",
				LangISO:   "en",
			},
			wantErr: "API token is required",
		},
		{
			name: "missing language returns error",
			cfg: UploadConfig{
				ProjectID: "p",
				Token:     "t",
			},
			wantErr: "base language (BASE_LANG) is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateRequiredFields(tt.cfg)

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
		})
	}
}

func TestValidateTaggingInputs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     UploadConfig
		wantErr string
	}{
		{
			name: "tagging disabled allows empty ref name",
			cfg: UploadConfig{
				SkipTagging:   true,
				GitHubRefName: "",
			},
		},
		{
			name: "tagging enabled with ref name passes",
			cfg: UploadConfig{
				SkipTagging:   false,
				GitHubRefName: "main",
			},
		},
		{
			name: "tagging enabled without ref name returns error",
			cfg: UploadConfig{
				SkipTagging:   false,
				GitHubRefName: "",
			},
			wantErr: "GitHub reference name (GITHUB_REF_NAME) is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTaggingInputs(tt.cfg)

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
		})
	}
}

func mustWriteTempFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "en.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}
