package main

import (
	"path/filepath"
	"reflect"
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
		wantPanic       bool
	}{
		{
			name: "Valid environment variables",
			env: map[string]string{
				"TRANSLATIONS_PATH": "\npath1\npath2\n\n",
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
			wantPanic: true,
		},
		{
			name: "Roots are cleaned and remain relative",
			env: map[string]string{
				"TRANSLATIONS_PATH": ".\n./locales\nlocales/../locales/en/..",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantPaths:       []string{".", "locales"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  false,
		},
		{
			name: "Absolute translations path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "/etc/locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantPanic: true,
		},
		{
			name: "Parent escape translations path fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "../locales",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "false",
			},
			wantPanic: true,
		},
		{
			name: "Name pattern glob variants are allowed",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "en/**/custom_*.json",
				"FLAT_NAMING":       "false",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "en/**/custom_*.json",
			wantFlatNaming:  false,
		},
		{
			name: "Absolute name pattern fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "/tmp/**/*.json",
				"FLAT_NAMING":       "false",
			},
			wantPanic: true,
		},
		{
			name: "File extensions are normalized and deduplicated",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          " JSON \n.yaml\njson\n YML \n .xml ",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json", "yaml", "yml", "xml"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Empty file extensions after normalization fail",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".\n \n",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPanic: true,
		},
		{
			name: "Leading dots and casing in file extensions normalize correctly",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          ".json\n.JSON\n json ",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Whitespace name pattern is treated as empty",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "   ",
				"FLAT_NAMING":       "true",
			},
			wantPaths:       []string{"translations"},
			wantBaseLang:    "en",
			wantFileExt:     []string{"json"},
			wantNamePattern: "",
			wantFlatNaming:  true,
		},
		{
			name: "Invalid FLAT_NAMING fails",
			env: map[string]string{
				"TRANSLATIONS_PATH": "translations",
				"BASE_LANG":         "en",
				"FILE_EXT":          "json",
				"NAME_PATTERN":      "",
				"FLAT_NAMING":       "wat",
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"TRANSLATIONS_PATH", "BASE_LANG", "FILE_EXT", "NAME_PATTERN", "FLAT_NAMING"} {
				t.Setenv(key, tt.env[key])
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic")
					}
				}()
			}

			got := validateEnvironment()

			if tt.wantPanic {
				return
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
