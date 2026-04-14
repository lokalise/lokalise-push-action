package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Hijack os.Exit so tests can assert hard exits.
	exitFunc = func(code int) { panic(fmt.Sprintf("Exit called with code %d", code)) }

	code := m.Run()

	// Restore.
	exitFunc = os.Exit
	os.Exit(code)
}

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      string
		wantPanic bool
	}{
		{
			name:      "missing CLI arg exits",
			args:      []string{"lokalise_upload"},
			wantPanic: true,
		},
		{
			name:      "empty CLI arg exits",
			args:      []string{"lokalise_upload", "   "},
			wantPanic: true,
		},
		{
			name: "valid CLI arg is trimmed",
			args: []string{"lokalise_upload", "  file.json  "},
			want: "file.json",
		},
		{
			name:      "too many CLI args exits",
			args:      []string{"lokalise_upload", "file.json", "extra"},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			if tt.wantPanic {
				requirePanicExit(t, func() { _ = parseCLIArgs() })
				return
			}

			got := parseCLIArgs()
			if got != tt.want {
				t.Fatalf("parseCLIArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

// requirePanicExit asserts the TestMain exit panic is thrown.
func requirePanicExit(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic from exitFunc, got none")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "Exit called with code 1") {
			t.Fatalf("expected exit code 1 panic, got: %v", r)
		}
	}()
	fn()
}
