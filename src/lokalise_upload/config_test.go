package main

import (
	"strings"
	"testing"
	"time"
)

var configEnvKeys = []string{
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
}

func TestPrepareConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		filePath string
		wantErr  string
		assert   func(t *testing.T, cfg UploadConfig)
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

				if cfg.ProjectID != "" {
					t.Fatalf("expected empty ProjectID, got %q", cfg.ProjectID)
				}
				if cfg.Token != "" {
					t.Fatalf("expected empty Token, got %q", cfg.Token)
				}
				if cfg.LangISO != "" {
					t.Fatalf("expected empty LangISO, got %q", cfg.LangISO)
				}
				if cfg.GitHubRefName != "" {
					t.Fatalf("expected empty GitHubRefName, got %q", cfg.GitHubRefName)
				}
				if cfg.AdditionalParams != "" {
					t.Fatalf("expected empty AdditionalParams, got %q", cfg.AdditionalParams)
				}

				if cfg.SkipTagging {
					t.Fatalf("expected SkipTagging=false, got true")
				}
				if cfg.SkipPolling {
					t.Fatalf("expected SkipPolling=false, got true")
				}
				if cfg.SkipDefaultFlags {
					t.Fatalf("expected SkipDefaultFlags=false, got true")
				}

				if cfg.MaxRetries != defaultMaxRetries {
					t.Fatalf("expected MaxRetries=%d, got %d", defaultMaxRetries, cfg.MaxRetries)
				}
				if cfg.InitialSleepTime != time.Duration(defaultInitialSleepTime)*time.Second {
					t.Fatalf("expected InitialSleepTime=%v, got %v", time.Duration(defaultInitialSleepTime)*time.Second, cfg.InitialSleepTime)
				}
				if cfg.MaxSleepTime != time.Duration(maxSleepTime)*time.Second {
					t.Fatalf("expected MaxSleepTime=%v, got %v", time.Duration(maxSleepTime)*time.Second, cfg.MaxSleepTime)
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

				if cfg.FilePath != "file.json" {
					t.Fatalf("expected FilePath=file.json, got %s", cfg.FilePath)
				}
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
				if !cfg.SkipTagging {
					t.Fatalf("expected SkipTagging=true, got false")
				}
				if !cfg.SkipPolling {
					t.Fatalf("expected SkipPolling=true, got false")
				}
				if !cfg.SkipDefaultFlags {
					t.Fatalf("expected SkipDefaultFlags=true, got false")
				}
				if cfg.MaxRetries != 10 {
					t.Fatalf("expected MaxRetries=10, got %d", cfg.MaxRetries)
				}
				if cfg.InitialSleepTime != 5*time.Second {
					t.Fatalf("expected InitialSleepTime=5s, got %v", cfg.InitialSleepTime)
				}
				if cfg.MaxSleepTime != time.Duration(maxSleepTime)*time.Second {
					t.Fatalf("expected MaxSleepTime=%v, got %v", time.Duration(maxSleepTime)*time.Second, cfg.MaxSleepTime)
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
			name: "explicit false bool envs are applied",
			env: map[string]string{
				"SKIP_TAGGING":       "false",
				"SKIP_POLLING":       "false",
				"SKIP_DEFAULT_FLAGS": "false",
			},
			filePath: "file.json",
			assert: func(t *testing.T, cfg UploadConfig) {
				t.Helper()

				if cfg.SkipTagging {
					t.Fatalf("expected SkipTagging=false, got true")
				}
				if cfg.SkipPolling {
					t.Fatalf("expected SkipPolling=false, got true")
				}
				if cfg.SkipDefaultFlags {
					t.Fatalf("expected SkipDefaultFlags=false, got true")
				}
			},
		},
		{
			name: "invalid SKIP_TAGGING returns error",
			env: map[string]string{
				"SKIP_TAGGING": "not-a-bool",
			},
			filePath: "file.json",
			wantErr:  "invalid SKIP_TAGGING",
		},
		{
			name: "invalid SKIP_POLLING returns error",
			env: map[string]string{
				"SKIP_POLLING": "not-a-bool",
			},
			filePath: "file.json",
			wantErr:  "invalid SKIP_POLLING",
		},
		{
			name: "invalid SKIP_DEFAULT_FLAGS returns error",
			env: map[string]string{
				"SKIP_DEFAULT_FLAGS": "not-a-bool",
			},
			filePath: "file.json",
			wantErr:  "invalid SKIP_DEFAULT_FLAGS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range configEnvKeys {
				t.Setenv(key, tt.env[key])
			}

			cfg, err := prepareConfig(tt.filePath)

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

			if tt.assert != nil {
				tt.assert(t, cfg)
			}
		})
	}
}
