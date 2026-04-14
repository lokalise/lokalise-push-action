package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFile(t *testing.T) {
	tmpFile := mustWriteTempFile(t)
	tmpDir := t.TempDir()
	missingFile := filepath.Join(t.TempDir(), "missing.json")

	tests := []struct {
		name      string
		path      string
		wantPanic bool
	}{
		{
			name: "regular file is accepted",
			path: tmpFile,
		},
		{
			name:      "missing file exits",
			path:      missingFile,
			wantPanic: true,
		},
		{
			name:      "directory exits",
			path:      tmpDir,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				requirePanicExit(t, func() { validateFile(tt.path) })
				return
			}
			validateFile(tt.path)
		})
	}
}

func TestValidate(t *testing.T) {
	validFile := mustWriteTempFile(t)

	tests := []struct {
		name      string
		cfg       UploadConfig
		wantPanic bool
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
			name: "missing project id exits",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantPanic: true,
		},
		{
			name: "directory path exits before field validation",
			cfg: UploadConfig{
				FilePath:      t.TempDir(),
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantPanic: true,
		},
		{
			name: "missing token exits",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantPanic: true,
		},
		{
			name: "missing language exits",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "",
				GitHubRefName: "ref",
			},
			wantPanic: true,
		},
		{
			name: "missing GitHubRefName exits when tagging enabled",
			cfg: UploadConfig{
				FilePath:      validFile,
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "",
				SkipTagging:   false,
			},
			wantPanic: true,
		},
		{
			name: "missing file path exits",
			cfg: UploadConfig{
				FilePath:      filepath.Join(t.TempDir(), "missing.json"),
				ProjectID:     "p",
				Token:         "t",
				LangISO:       "en",
				GitHubRefName: "ref",
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				requirePanicExit(t, func() { validate(tt.cfg) })
				return
			}
			validate(tt.cfg)
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		cfg       UploadConfig
		wantPanic bool
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
			name: "missing project id exits",
			cfg: UploadConfig{
				Token:   "t",
				LangISO: "en",
			},
			wantPanic: true,
		},
		{
			name: "missing token exits",
			cfg: UploadConfig{
				ProjectID: "p",
				LangISO:   "en",
			},
			wantPanic: true,
		},
		{
			name: "missing language exits",
			cfg: UploadConfig{
				ProjectID: "p",
				Token:     "t",
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				requirePanicExit(t, func() { validateRequiredFields(tt.cfg) })
				return
			}
			validateRequiredFields(tt.cfg)
		})
	}
}

func TestValidateTaggingInputs(t *testing.T) {
	tests := []struct {
		name      string
		cfg       UploadConfig
		wantPanic bool
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
			name: "tagging enabled without ref name exits",
			cfg: UploadConfig{
				SkipTagging:   false,
				GitHubRefName: "",
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				requirePanicExit(t, func() { validateTaggingInputs(tt.cfg) })
				return
			}
			validateTaggingInputs(tt.cfg)
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
