package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bodrovis/lokex/v2/client/upload"
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

func TestPrepareConfig(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		filePath  string
		wantPanic bool
		assert    func(t *testing.T, cfg UploadConfig)
	}{
		{
			name: "defaults are applied",
			env: map[string]string{
				"LOKALISE_PROJECT_ID": "",
				"LOKALISE_API_TOKEN":  "",
				"BASE_LANG":           "",
				"GITHUB_REF_NAME":     "",
				"ADDITIONAL_PARAMS":   "",
				"SKIP_TAGGING":        "",
				"SKIP_POLLING":        "",
				"SKIP_DEFAULT_FLAGS":  "",
				"MAX_RETRIES":         "",
				"SLEEP_TIME":          "",
				"UPLOAD_TIMEOUT":      "",
				"HTTP_TIMEOUT":        "",
				"POLL_INITIAL_WAIT":   "",
				"POLL_MAX_WAIT":       "",
			},
			filePath: "test.json",
			assert: func(t *testing.T, cfg UploadConfig) {
				t.Helper()
				if cfg.FilePath != "test.json" {
					t.Fatalf("expected FilePath=test.json, got %s", cfg.FilePath)
				}
				if cfg.MaxRetries != defaultMaxRetries {
					t.Fatalf("expected MaxRetries=%d, got %d", defaultMaxRetries, cfg.MaxRetries)
				}
				if cfg.InitialSleepTime != time.Duration(defaultInitialSleepTime)*time.Second {
					t.Fatalf("expected InitialSleepTime=%v, got %v", time.Duration(defaultInitialSleepTime)*time.Second, cfg.InitialSleepTime)
				}
				if cfg.UploadTimeout != time.Duration(defaultUploadTimeout)*time.Second {
					t.Fatalf("expected UploadTimeout=%v, got %v", time.Duration(defaultUploadTimeout)*time.Second, cfg.UploadTimeout)
				}
				if cfg.HTTPTimeout != time.Duration(defaultHTTPTimeout)*time.Second {
					t.Fatalf("expected HTTPTimeout=%v, got %v", time.Duration(defaultHTTPTimeout)*time.Second, cfg.HTTPTimeout)
				}
				if cfg.PollInitialWait != time.Duration(defaultPollInitialWait)*time.Second {
					t.Fatalf("expected PollInitialWait=%v, got %v", time.Duration(defaultPollInitialWait)*time.Second, cfg.PollInitialWait)
				}
				if cfg.PollMaxWait != time.Duration(defaultPollMaxWait)*time.Second {
					t.Fatalf("expected PollMaxWait=%v, got %v", time.Duration(defaultPollMaxWait)*time.Second, cfg.PollMaxWait)
				}
			},
		},
		{
			name: "env overrides are applied and trimmed",
			env: map[string]string{
				"LOKALISE_PROJECT_ID": "  proj123  ",
				"LOKALISE_API_TOKEN":  "  token123  ",
				"BASE_LANG":           "  en  ",
				"GITHUB_REF_NAME":     "  refs/heads/main  ",
				"ADDITIONAL_PARAMS":   "  {\"custom\": true}  ",
				"SKIP_TAGGING":        "true",
				"SKIP_POLLING":        "true",
				"SKIP_DEFAULT_FLAGS":  "true",
				"MAX_RETRIES":         "10",
				"SLEEP_TIME":          "5",
				"UPLOAD_TIMEOUT":      "42",
				"HTTP_TIMEOUT":        "11",
				"POLL_INITIAL_WAIT":   "7",
				"POLL_MAX_WAIT":       "8",
			},
			filePath: "file.json",
			assert: func(t *testing.T, cfg UploadConfig) {
				t.Helper()
				if cfg.ProjectID != "proj123" {
					t.Fatalf("expected ProjectID=proj123, got %s", cfg.ProjectID)
				}
				if cfg.Token != "token123" {
					t.Fatalf("expected Token=token123, got %s", cfg.Token)
				}
				if cfg.LangISO != "en" {
					t.Fatalf("expected LangISO=en, got %s", cfg.LangISO)
				}
				if cfg.GitHubRefName != "refs/heads/main" {
					t.Fatalf("expected GitHubRefName=refs/heads/main, got %s", cfg.GitHubRefName)
				}
				if cfg.AdditionalParams != "{\"custom\": true}" {
					t.Fatalf("expected trimmed AdditionalParams, got %q", cfg.AdditionalParams)
				}
				if !cfg.SkipTagging || !cfg.SkipPolling || !cfg.SkipDefaultFlags {
					t.Fatalf("expected all skip flags to be true: %#v", cfg)
				}
				if cfg.MaxRetries != 10 {
					t.Fatalf("expected MaxRetries=10, got %d", cfg.MaxRetries)
				}
				if cfg.InitialSleepTime != 5*time.Second {
					t.Fatalf("expected InitialSleepTime=5s, got %v", cfg.InitialSleepTime)
				}
				if cfg.UploadTimeout != 42*time.Second {
					t.Fatalf("expected UploadTimeout=42s, got %v", cfg.UploadTimeout)
				}
				if cfg.HTTPTimeout != 11*time.Second {
					t.Fatalf("expected HTTPTimeout=11s, got %v", cfg.HTTPTimeout)
				}
				if cfg.PollInitialWait != 7*time.Second {
					t.Fatalf("expected PollInitialWait=7s, got %v", cfg.PollInitialWait)
				}
				if cfg.PollMaxWait != 8*time.Second {
					t.Fatalf("expected PollMaxWait=8s, got %v", cfg.PollMaxWait)
				}
			},
		},
		{
			name: "invalid SKIP_TAGGING exits",
			env: map[string]string{
				"SKIP_TAGGING": "not-a-bool",
			},
			filePath:  "file.json",
			wantPanic: true,
		},
		{
			name: "invalid SKIP_POLLING exits",
			env: map[string]string{
				"SKIP_POLLING": "not-a-bool",
			},
			filePath:  "file.json",
			wantPanic: true,
		},
		{
			name: "invalid SKIP_DEFAULT_FLAGS exits",
			env: map[string]string{
				"SKIP_DEFAULT_FLAGS": "not-a-bool",
			},
			filePath:  "file.json",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"LOKALISE_PROJECT_ID",
				"LOKALISE_API_TOKEN",
				"BASE_LANG",
				"GITHUB_REF_NAME",
				"ADDITIONAL_PARAMS",
				"SKIP_TAGGING",
				"SKIP_POLLING",
				"SKIP_DEFAULT_FLAGS",
				"MAX_RETRIES",
				"SLEEP_TIME",
				"UPLOAD_TIMEOUT",
				"HTTP_TIMEOUT",
				"POLL_INITIAL_WAIT",
				"POLL_MAX_WAIT",
			} {
				t.Setenv(key, tt.env[key])
			}

			if tt.wantPanic {
				requirePanicExit(t, func() { _ = prepareConfig(tt.filePath) })
				return
			}

			cfg := prepareConfig(tt.filePath)
			if tt.assert != nil {
				tt.assert(t, cfg)
			}
		})
	}
}

func TestBuildUploadParams(t *testing.T) {
	tests := []struct {
		name       string
		cfg        UploadConfig
		want       upload.UploadParams
		absentKeys []string
		wantErr    bool
	}{
		{
			name: "defaults tagging and merge all work",
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
			name: "skip flags and tags omits action defaults",
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

func TestUploadFile(t *testing.T) {
	tests := []struct {
		name          string
		cfg           UploadConfig
		factory       *fakeUploadFactory
		wantErrSubstr string
		assert        func(t *testing.T, fu *fakeUploader, ff *fakeUploadFactory)
	}{
		{
			name: "success with polling enabled",
			cfg: UploadConfig{
				FilePath:         "tmp/en.json",
				ProjectID:        "proj_123",
				Token:            "tok_abc",
				LangISO:          "en",
				GitHubRefName:    "v1.0.0",
				SkipTagging:      false,
				SkipDefaultFlags: false,
				SkipPolling:      false,
				MaxRetries:       7,
				InitialSleepTime: 2 * time.Second,
				MaxSleepTime:     30 * time.Second,
				HTTPTimeout:      25 * time.Second,
				PollInitialWait:  1 * time.Second,
				PollMaxWait:      10 * time.Second,
			},
			factory: &fakeUploadFactory{
				uploader: &fakeUploader{returnPID: "upl_123"},
			},
			assert: func(t *testing.T, fu *fakeUploader, ff *fakeUploadFactory) {
				t.Helper()
				if ff.gotToken != "tok_abc" || ff.gotProjectID != "proj_123" {
					t.Fatalf("factory creds wrong: tok=%s proj=%s", ff.gotToken, ff.gotProjectID)
				}
				if ff.gotRetries != 7 || ff.gotHTTPTO != 25*time.Second {
					t.Fatalf("retries/httpTO wrong: %d / %v", ff.gotRetries, ff.gotHTTPTO)
				}
				if ff.gotInitialBackoff != 2*time.Second || ff.gotMaxBackoff != 30*time.Second {
					t.Fatalf("backoff wrong: %v / %v", ff.gotInitialBackoff, ff.gotMaxBackoff)
				}
				if ff.gotPollInit != 1*time.Second || ff.gotPollMax != 10*time.Second {
					t.Fatalf("poll waits wrong: %v / %v", ff.gotPollInit, ff.gotPollMax)
				}
				if !fu.called {
					t.Fatalf("expected Upload to be called")
				}
				if fu.gotPoll != true {
					t.Fatalf("expected poll=true, got %v", fu.gotPoll)
				}
				if fu.gotSrcPath != "" {
					t.Fatalf("expected srcPath to be empty string, got %q", fu.gotSrcPath)
				}
				if fu.gotParams["filename"] != "tmp/en.json" || fu.gotParams["lang_iso"] != "en" {
					t.Fatalf("params wrong: %#v", fu.gotParams)
				}
				if fu.gotParams["replace_modified"] != true || fu.gotParams["include_path"] != true {
					t.Fatalf("default flags missing: %#v", fu.gotParams)
				}
			},
		},
		{
			name: "success with polling disabled",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				ProjectID:        "proj_123",
				Token:            "tok_abc",
				LangISO:          "en",
				GitHubRefName:    "v1.0.0",
				SkipPolling:      true,
				MaxRetries:       3,
				InitialSleepTime: 1 * time.Second,
				MaxSleepTime:     10 * time.Second,
				HTTPTimeout:      20 * time.Second,
				PollInitialWait:  2 * time.Second,
				PollMaxWait:      5 * time.Second,
			},
			factory: &fakeUploadFactory{
				uploader: &fakeUploader{returnPID: "upl_999"},
			},
			assert: func(t *testing.T, fu *fakeUploader, ff *fakeUploadFactory) {
				t.Helper()
				if fu.gotPoll != false {
					t.Fatalf("expected poll=false, got %v", fu.gotPoll)
				}
			},
		},
		{
			name: "factory error is wrapped",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				ProjectID:        "proj_123",
				Token:            "tok_abc",
				LangISO:          "en",
				GitHubRefName:    "main",
				MaxRetries:       1,
				InitialSleepTime: 1 * time.Second,
				MaxSleepTime:     5 * time.Second,
				HTTPTimeout:      10 * time.Second,
			},
			factory: &fakeUploadFactory{
				wantErr: errors.New("boom"),
			},
			wantErrSubstr: "cannot create Lokalise API client",
		},
		{
			name: "upload error is wrapped",
			cfg: UploadConfig{
				FilePath:      "/tmp/en.json",
				ProjectID:     "proj_123",
				Token:         "tok_abc",
				LangISO:       "en",
				GitHubRefName: "main",
			},
			factory: &fakeUploadFactory{
				uploader: &fakeUploader{returnErr: errors.New("network down")},
			},
			wantErrSubstr: "failed to upload file",
		},
		{
			name: "invalid additional params return error before upload",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				ProjectID:        "proj_123",
				Token:            "tok_abc",
				LangISO:          "en",
				GitHubRefName:    "main",
				AdditionalParams: `{"broken": true,`,
			},
			factory: &fakeUploadFactory{
				uploader: &fakeUploader{},
			},
			wantErrSubstr: "invalid additional_params",
			assert: func(t *testing.T, fu *fakeUploader, ff *fakeUploadFactory) {
				t.Helper()
				if fu.called {
					t.Fatalf("uploader.Upload should not be called when params are invalid")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := uploadFile(ctx, tt.cfg, tt.factory)

			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErrSubstr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.assert != nil {
				fu, _ := tt.factory.uploader.(*fakeUploader)
				tt.assert(t, fu, tt.factory)
			}
		})
	}
}

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
			name: "bad file path exits before field validation",
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

// ---------- fakes & helpers ----------

type fakeUploader struct {
	called     bool
	gotCtx     context.Context
	gotParams  upload.UploadParams
	gotSrcPath string
	gotPoll    bool

	returnPID string
	returnErr error
}

func (f *fakeUploader) Upload(ctx context.Context, params upload.UploadParams, srcPath string, poll bool) (string, error) {
	f.called = true
	f.gotCtx = ctx
	f.gotParams = params
	f.gotSrcPath = srcPath
	f.gotPoll = poll
	return f.returnPID, f.returnErr
}

type fakeUploadFactory struct {
	wantErr error

	// Capture args to assert.
	gotToken          string
	gotProjectID      string
	gotRetries        int
	gotHTTPTO         time.Duration
	gotInitialBackoff time.Duration
	gotMaxBackoff     time.Duration
	gotPollInit       time.Duration
	gotPollMax        time.Duration

	uploader Uploader
}

func (f *fakeUploadFactory) NewUploader(cfg UploadConfig) (Uploader, error) {
	f.gotToken = cfg.Token
	f.gotProjectID = cfg.ProjectID
	f.gotRetries = cfg.MaxRetries
	f.gotHTTPTO = cfg.HTTPTimeout
	f.gotInitialBackoff = cfg.InitialSleepTime
	f.gotMaxBackoff = cfg.MaxSleepTime
	f.gotPollInit = cfg.PollInitialWait
	f.gotPollMax = cfg.PollMaxWait

	if f.wantErr != nil {
		return nil, f.wantErr
	}
	if f.uploader == nil {
		return &fakeUploader{returnPID: "upl_default"}, nil
	}
	return f.uploader, nil
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

// requirePanicExit asserts the TestMain exit panic is thrown.
func requirePanicExit(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic from exitFunc, got none")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "Exit called with code") {
			t.Fatalf("expected exit panic, got: %v", r)
		}
	}()
	fn()
}
