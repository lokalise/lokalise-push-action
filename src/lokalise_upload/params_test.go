package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bodrovis/lokex/v2/client/upload"
)

func TestBuildUploadParams(t *testing.T) {
	tests := []struct {
		name       string
		cfg        UploadConfig
		want       upload.UploadParams
		absentKeys []string
		wantErr    bool
	}{
		{
			name: "defaults tagging and additional params are merged",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "release-2025-08-21",
				SkipTagging:      false,
				SkipDefaultFlags: false,
				AdditionalParams: `
{
  "convert_placeholders": true,
  "custom_bool": false,
  "tags": ["custom-tag-1","custom-tag-2"]
}`,
			},
			want: upload.UploadParams{
				"filename":             "/tmp/en.json",
				"lang_iso":             "en",
				"replace_modified":     true,
				"include_path":         true,
				"distinguish_by_file":  true,
				"convert_placeholders": true,
				"custom_bool":          false,
				"tags":                 []any{"custom-tag-1", "custom-tag-2"},
				"tag_inserted_keys":    true,
				"tag_skipped_keys":     true,
				"tag_updated_keys":     true,
			},
		},
		{
			name: "empty additional params use defaults",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "release-1",
				SkipTagging:      false,
				SkipDefaultFlags: false,
				AdditionalParams: "",
			},
			want: upload.UploadParams{
				"filename":            "/tmp/en.json",
				"lang_iso":            "en",
				"replace_modified":    true,
				"include_path":        true,
				"distinguish_by_file": true,
				"tags":                []string{"release-1"},
				"tag_inserted_keys":   true,
				"tag_skipped_keys":    true,
				"tag_updated_keys":    true,
			},
		},
		{
			name: "skip default flags and tagging omits action params",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "release-1",
				SkipTagging:      true,
				SkipDefaultFlags: true,
			},
			want: upload.UploadParams{
				"filename": "/tmp/en.json",
				"lang_iso": "en",
			},
			absentKeys: []string{
				"replace_modified",
				"include_path",
				"distinguish_by_file",
				"tags",
				"tag_inserted_keys",
				"tag_skipped_keys",
				"tag_updated_keys",
			},
		},
		{
			name: "additional params can override defaults",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "release-1",
				SkipTagging:      false,
				SkipDefaultFlags: false,
				AdditionalParams: `
include_path: false
replace_modified: false
custom_number: 42
`,
			},
			want: upload.UploadParams{
				"filename":            "/tmp/en.json",
				"lang_iso":            "en",
				"replace_modified":    false,
				"include_path":        false,
				"distinguish_by_file": true,
				"custom_number":       42,
				"tags":                []string{"release-1"},
				"tag_inserted_keys":   true,
				"tag_skipped_keys":    true,
				"tag_updated_keys":    true,
			},
		},
		{
			name: "additional params can set tags even when tagging is skipped",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				SkipTagging:      true,
				SkipDefaultFlags: false,
				AdditionalParams: `
{
  "tags": ["manual-tag"]
}`,
			},
			want: upload.UploadParams{
				"filename":            "/tmp/en.json",
				"lang_iso":            "en",
				"replace_modified":    true,
				"include_path":        true,
				"distinguish_by_file": true,
				"tags":                []any{"manual-tag"},
			},
			absentKeys: []string{
				"tag_inserted_keys",
				"tag_skipped_keys",
				"tag_updated_keys",
			},
		},
		{
			name: "additional params can override base params",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "release-1",
				SkipTagging:      false,
				SkipDefaultFlags: false,
				AdditionalParams: `
lang_iso: de
filename: /tmp/override.json
`,
			},
			want: upload.UploadParams{
				"filename":            "/tmp/override.json",
				"lang_iso":            "de",
				"replace_modified":    true,
				"include_path":        true,
				"distinguish_by_file": true,
				"tags":                []string{"release-1"},
				"tag_inserted_keys":   true,
				"tag_skipped_keys":    true,
				"tag_updated_keys":    true,
			},
		},
		{
			name: "invalid additional params return error",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				LangISO:          "en",
				GitHubRefName:    "ref",
				AdditionalParams: `{"convert_placeholders": true,`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildUploadParams(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "invalid additional_params") {
					t.Fatalf("expected wrapped error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotNorm := normalizeUploadParams(got)
			wantNorm := normalizeUploadParams(tt.want)

			if !reflect.DeepEqual(gotNorm, wantNorm) {
				t.Fatalf("params mismatch.\n got: %#v\nwant: %#v", gotNorm, wantNorm)
			}

			for _, key := range tt.absentKeys {
				if _, ok := got[key]; ok {
					t.Fatalf("key %q should be absent, got value %#v", key, got[key])
				}
			}
		})
	}
}

func normalizeUploadParams(p upload.UploadParams) map[string]any {
	out := make(map[string]any, len(p))
	for k, v := range p {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(v any) any {
	switch s := v.(type) {
	case []string:
		out := make([]any, len(s))
		for i := range s {
			out[i] = s[i]
		}
		return out
	default:
		return v
	}
}
