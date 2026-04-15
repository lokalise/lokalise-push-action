package main

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"strings"
	"testing"
)

func TestWriteUniqueLine(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		initialSeen map[string]struct{}
		wantOutput  string
		wantSeenKey string
		wantErr     string
		writer      *failingWriter
	}{
		{
			name:        "writes normalized anchored line",
			path:        "translations/en.json",
			initialSeen: map[string]struct{}{},
			wantOutput:  "translations/en.json\n",
			wantSeenKey: "translations/en.json",
		},
		{
			name: "skips duplicate line already in seen",
			path: "translations/en.json",
			initialSeen: map[string]struct{}{
				"translations/en.json": {},
			},
			wantOutput:  "",
			wantSeenKey: "translations/en.json",
		},
		{
			name:        "normalizes path traversal within relative path",
			path:        "translations/../translations/en.json",
			initialSeen: map[string]struct{}{},
			wantOutput:  "translations/en.json\n",
			wantSeenKey: "translations/en.json",
		},
		{
			name:        "returns writer error",
			path:        "translations/en.json",
			initialSeen: map[string]struct{}{},
			wantErr:     "write failed",
			writer:      &failingWriter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seen := make(map[string]struct{}, len(tt.initialSeen))
			maps.Copy(seen, tt.initialSeen)

			var buf bytes.Buffer
			var w interface{ Write([]byte) (int, error) } = &buf
			if tt.writer != nil {
				w = tt.writer
			}

			err := writeUniqueLine(w, seen, tt.path)

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

			if buf.String() != tt.wantOutput {
				t.Fatalf("output mismatch. want=%q got=%q", tt.wantOutput, buf.String())
			}

			if _, ok := seen[tt.wantSeenKey]; !ok {
				t.Fatalf("expected seen to contain %q, got=%v", tt.wantSeenKey, seen)
			}
		})
	}
}

func TestCreateOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	file, err := createOutputFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if file.Name() != "lok_action_paths_temp.txt" {
		t.Fatalf("unexpected file name: %q", file.Name())
	}

	if _, err := os.Stat("lok_action_paths_temp.txt"); err != nil {
		t.Fatalf("expected file to exist, stat failed: %v", err)
	}
}

func TestCloseOutputFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "close-test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	closeOutputFile(file)

	if _, err := file.Write([]byte("x")); err == nil {
		t.Fatal("expected write to fail after close, but it succeeded")
	}
}

type failingWriter struct{}

func (f failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
