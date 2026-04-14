package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bodrovis/lokex/v2/client/upload"
)

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
				if !ff.called {
					t.Fatalf("expected factory.NewUploader to be called")
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
			name: "skip default flags and tagging are respected",
			cfg: UploadConfig{
				FilePath:         "/tmp/en.json",
				ProjectID:        "proj_123",
				Token:            "tok_abc",
				LangISO:          "en",
				GitHubRefName:    "main",
				SkipTagging:      true,
				SkipDefaultFlags: true,
			},
			factory: &fakeUploadFactory{
				uploader: &fakeUploader{returnPID: "upl_555"},
			},
			assert: func(t *testing.T, fu *fakeUploader, ff *fakeUploadFactory) {
				t.Helper()
				if _, ok := fu.gotParams["replace_modified"]; ok {
					t.Fatalf("replace_modified should be absent")
				}
				if _, ok := fu.gotParams["include_path"]; ok {
					t.Fatalf("include_path should be absent")
				}
				if _, ok := fu.gotParams["distinguish_by_file"]; ok {
					t.Fatalf("distinguish_by_file should be absent")
				}
				if _, ok := fu.gotParams["tags"]; ok {
					t.Fatalf("tags should be absent")
				}
				if _, ok := fu.gotParams["tag_inserted_keys"]; ok {
					t.Fatalf("tag_inserted_keys should be absent")
				}
				if _, ok := fu.gotParams["tag_skipped_keys"]; ok {
					t.Fatalf("tag_skipped_keys should be absent")
				}
				if _, ok := fu.gotParams["tag_updated_keys"]; ok {
					t.Fatalf("tag_updated_keys should be absent")
				}
			},
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
			wantErrSubstr: `failed to upload file "/tmp/en.json"`,
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
				if ff.called {
					t.Fatalf("factory.NewUploader should not be called when params are invalid")
				}
				if fu != nil && fu.called {
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

func TestUploadFile_PassesContextToUploader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fu := &fakeUploader{returnPID: "upl_123"}
	ff := &fakeUploadFactory{uploader: fu}

	cfg := UploadConfig{
		FilePath:      "/tmp/en.json",
		ProjectID:     "proj_123",
		Token:         "tok_abc",
		LangISO:       "en",
		GitHubRefName: "main",
	}

	if err := uploadFile(ctx, cfg, ff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fu.gotCtx != ctx {
		t.Fatalf("expected upload to receive the original context")
	}
}

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
	called  bool

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
	f.called = true

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
