package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		env             map[string]string
		wantPaths       []string
		wantBaseLang    string
		wantFileExt     []string
		wantNamePattern string
		wantFlatNaming  bool
		wantErr         string
	}{
		{
			name: "Valid environment variables",
			env: map[string]string{
				"TRANSLATIONS_PATH": "\npath1\n\npath2\n",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "custom_name.json",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"path1", "path2"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "custom_name.json",
			wantFlatNaming:  true,
		},
		{
			name: "Missing environment variables",
			env: map[string]string{
				"TRANSLATIONS_PATH": "",
				"BASE_LANG":         "",
				"FILE_EXT":          "",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantErr: "failed to process params: environment variable TRANSLATIONS_PATH is required",
		},
		{
			name: "NAME_PATTERN trims and normalizes path-like pattern",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "  ./custom//../custom/**/*.json  ",
				"FLAT_NAMING":       "false",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "custom/**/*.json",
			wantFlatNaming:  false,
		},
		{
			name: "Root translation path",
			env: map[string]string{
				"TRANSLATIONS_PATH": ".",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"."},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Name pattern with ../",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "../**/*.json",
				"FLAT_NAMING":       "false",
			},
			wantErr: "path escapes repo root",
		},
		{
			name: "Translation path with ../",
			env: map[string]string{
				"TRANSLATIONS_PATH": "../locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantErr: "failed to process params: invalid path",
		},
		{
			name: "TRANSLATIONS_PATH cleans to .. fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "a/../..",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantErr: "failed to process params: invalid path",
		},
		{
			name: "TRANSLATIONS_PATH ./path is OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "./path",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"path"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "TRANSLATIONS_PATH /path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "/path",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantErr: "failed to process params: invalid path",
		},
		{
			name: "NamePattern glob OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "**/*.yaml",
				"FLAT_NAMING":       "false",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "**/*.yaml",
			wantFlatNaming:  false,
		},
		{
			name: "NamePattern nested glob OK",
			env: map[string]string{
				"TRANSLATIONS_PATH": "pkg/i18n",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "en/**/custom_*.json",
				"FLAT_NAMING":       "false",
			},
			wantPaths:       []string{"pkg/i18n"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: filepath.Clean("en/**/custom_*.json"),
			wantFlatNaming:  false,
		},
		{
			name: "FILE_EXT normalization and dedupe",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          " JSON \n.yaml\njson\n YML \n .xml ",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json", "yaml", "yml", "xml"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "FILE_EXT empty after normalization fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".\n \n",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantErr: "invalid FILE_EXT",
		},
		{
			name: "Absolute NAME_PATTERN fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "/tmp/file.json",
				"FLAT_NAMING":       "true",
			},
			wantErr: "must be relative",
		},
		{
			name: "Whitespace NAME_PATTERN is treated as empty",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "   ",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "NAME_PATTERN tilde fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "~/.config/*.json",
				"FLAT_NAMING":       "true",
			},
			wantErr: "must be relative",
		},
		{
			name: "FILE_EXT skips empty entries after normalization",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "\n \n.json\n.\n yaml \n",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json", "yaml"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Leading dots and casing in FILE_EXT normalize correctly",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".json\n.JSON\n json ",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Invalid FLAT_NAMING fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "wat",
			},
			wantErr: "invalid FLAT_NAMING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"TRANSLATIONS_PATH",
				"BASE_LANG",
				"FILE_EXT",
				"NAME_PATTERN",
				"FLAT_NAMING",
			} {
				t.Setenv(key, tt.env[key])
			}

			got, err := validateEnvironment()

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

			if !reflect.DeepEqual(got.Paths, tt.wantPaths) {
				t.Fatalf("paths mismatch. want=%v got=%v", tt.wantPaths, got.Paths)
			}
			if got.BaseLang != tt.wantBaseLang {
				t.Fatalf("baseLang mismatch. want=%q got=%q", tt.wantBaseLang, got.BaseLang)
			}
			if !reflect.DeepEqual(got.FileExts, tt.wantFileExt) {
				t.Fatalf("fileExt mismatch. want=%v got=%v", tt.wantFileExt, got.FileExts)
			}
			if filepath.ToSlash(got.NamePattern) != filepath.ToSlash(tt.wantNamePattern) {
				t.Fatalf("namePattern mismatch. want=%q got=%q", tt.wantNamePattern, got.NamePattern)
			}
			if got.FlatNaming != tt.wantFlatNaming {
				t.Fatalf("flatNaming mismatch. want=%v got=%v", tt.wantFlatNaming, got.FlatNaming)
			}
		})
	}
}
