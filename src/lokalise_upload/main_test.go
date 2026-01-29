package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bodrovis/lokex/v2/client"
)

func TestMain(m *testing.M) {
	// hijack os.Exit so we can assert hard exits
	exitFunc = func(code int) { panic(fmt.Sprintf("Exit called with code %d", code)) }

	code := m.Run()

	// restore
	exitFunc = os.Exit

	os.Exit(code)
}

// ---------- prepareConfig tests --------------
func TestPrepareConfig_Defaults(t *testing.T) {
	t.Setenv("LOKALISE_PROJECT_ID", "")
	t.Setenv("LOKALISE_API_TOKEN", "")
	t.Setenv("BASE_LANG", "")
	t.Setenv("GITHUB_REF_NAME", "")

	cfg := prepareConfig("test.json")

	if cfg.FilePath != "test.json" {
		t.Errorf("expected FilePath=test.json, got %s", cfg.FilePath)
	}
	if cfg.MaxRetries != defaultMaxRetries {
		t.Errorf("expected MaxRetries=%d, got %d", defaultMaxRetries, cfg.MaxRetries)
	}
	if cfg.InitialSleepTime != time.Duration(defaultInitialSleepTime)*time.Second {
		t.Errorf("expected InitialSleepTime=%v, got %v", time.Duration(defaultInitialSleepTime)*time.Second, cfg.InitialSleepTime)
	}
	if cfg.UploadTimeout != time.Duration(defaultUploadTimeout)*time.Second {
		t.Errorf("expected UploadTimeout=%v, got %v", time.Duration(defaultUploadTimeout)*time.Second, cfg.UploadTimeout)
	}
}

func TestPrepareConfig_WithEnvOverrides(t *testing.T) {
	t.Setenv("LOKALISE_PROJECT_ID", "proj123")
	t.Setenv("LOKALISE_API_TOKEN", "token123")
	t.Setenv("BASE_LANG", "en")
	t.Setenv("GITHUB_REF_NAME", "refs/heads/main")
	t.Setenv("MAX_RETRIES", "10")
	t.Setenv("SLEEP_TIME", "5")
	t.Setenv("UPLOAD_TIMEOUT", "42")

	cfg := prepareConfig("file.json")

	if cfg.ProjectID != "proj123" {
		t.Errorf("expected ProjectID=proj123, got %s", cfg.ProjectID)
	}
	if cfg.Token != "token123" {
		t.Errorf("expected Token=token123, got %s", cfg.Token)
	}
	if cfg.LangISO != "en" {
		t.Errorf("expected LangISO=en, got %s", cfg.LangISO)
	}
	if cfg.GitHubRefName != "refs/heads/main" {
		t.Errorf("expected GitHubRefName=refs/heads/main, got %s", cfg.GitHubRefName)
	}
	if cfg.MaxRetries != 10 {
		t.Errorf("expected MaxRetries=10, got %d", cfg.MaxRetries)
	}
	if cfg.InitialSleepTime != 5*time.Second {
		t.Errorf("expected InitialSleepTime=5s, got %v", cfg.InitialSleepTime)
	}
	if cfg.UploadTimeout != 42*time.Second {
		t.Errorf("expected UploadTimeout=42s, got %v", cfg.UploadTimeout)
	}
}

func TestPrepareConfig_ParseInvalidBoolEnv(t *testing.T) {
	t.Setenv("SKIP_TAGGING", "not-a-bool")

	requirePanicExit(t, func() { _ = prepareConfig("file.json") })
}

func TestPrepareConfig_EmptyFilePath(t *testing.T) {
	requirePanicExit(t, func() { _ = prepareConfig("  ") })
}

// ---------- buildUploadParams tests ----------

func TestBuildUploadParams_MergesAndDefaults(t *testing.T) {
	cfg := UploadConfig{
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
	}

	p := buildUploadParams(cfg)

	want := client.UploadParams{
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
	}

	normalize := func(v any) any {
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
	got := map[string]any{}
	for k, v := range p {
		got[k] = normalize(v)
	}
	exp := map[string]any{}
	for k, v := range want {
		exp[k] = normalize(v)
	}

	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("params mismatch.\n got: %#v\nwant: %#v", got, exp)
	}
}

func TestBuildUploadParams_EmptyAdditional_UsesDefaults(t *testing.T) {
	cfg := UploadConfig{
		FilePath:         "/tmp/en.json",
		LangISO:          "en",
		GitHubRefName:    "release-1",
		SkipTagging:      false,
		SkipDefaultFlags: false,
		AdditionalParams: "",
	}
	p := buildUploadParams(cfg)

	if p["filename"] != "/tmp/en.json" {
		t.Fatalf("filename wrong: %v", p["filename"])
	}
	if p["lang_iso"] != "en" {
		t.Fatalf("lang_iso wrong: %v", p["lang_iso"])
	}

	if p["replace_modified"] != true || p["include_path"] != true || p["distinguish_by_file"] != true {
		t.Fatalf("default flags not set correctly: %#v", p)
	}

	switch tags := p["tags"].(type) {
	case []string:
		if len(tags) != 1 || tags[0] != "release-1" {
			t.Fatalf("tags wrong: %#v", tags)
		}
	case []any:
		if len(tags) != 1 || tags[0] != "release-1" {
			t.Fatalf("tags wrong: %#v", tags)
		}
	default:
		t.Fatalf("tags type wrong: %T", p["tags"])
	}
	if p["tag_inserted_keys"] != true || p["tag_skipped_keys"] != true || p["tag_updated_keys"] != true {
		t.Fatalf("tag_* flags missing")
	}
}

func TestBuildUploadParams_SkipFlagsAndTags(t *testing.T) {
	cfg := UploadConfig{
		FilePath:         "/tmp/en.json",
		LangISO:          "en",
		GitHubRefName:    "release-1",
		SkipTagging:      true,
		SkipDefaultFlags: true,
	}
	p := buildUploadParams(cfg)

	if _, ok := p["replace_modified"]; ok {
		t.Fatalf("replace_modified should be omitted when SkipDefaultFlags=true")
	}
	if _, ok := p["tags"]; ok {
		t.Fatalf("tags should be omitted when SkipTagging=true")
	}
	if _, ok := p["tag_inserted_keys"]; ok {
		t.Fatalf("tag_* should be omitted when SkipTagging=true")
	}
}

func TestBuildUploadParams_BadJSON_Aborts(t *testing.T) {
	cfg := UploadConfig{
		FilePath:         "/tmp/en.json",
		LangISO:          "en",
		GitHubRefName:    "ref",
		AdditionalParams: `{"convert_placeholders": true,`, // broken
	}
	requirePanicExit(t, func() { _ = buildUploadParams(cfg) })
}

// ---------- uploadFile tests ----------

func TestUploadFile_Success_PollingEnabled(t *testing.T) {
	cfg := UploadConfig{
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
	}

	fu := &fakeUploader{returnPID: "upl_123"}
	ff := &fakeUploadFactory{uploader: fu}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := uploadFile(ctx, cfg, ff); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

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
	if fu.gotParams["filename"] != "tmp/en.json" || fu.gotParams["lang_iso"] != "en" {
		t.Fatalf("params wrong: %#v", fu.gotParams)
	}

	if fu.gotParams["replace_modified"] != true || fu.gotParams["include_path"] != true {
		t.Fatalf("default flags missing: %#v", fu.gotParams)
	}
}

func TestUploadFile_Success_PollingDisabled(t *testing.T) {
	cfg := UploadConfig{
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
	}

	fu := &fakeUploader{returnPID: "upl_999"}
	ff := &fakeUploadFactory{uploader: fu}

	if err := uploadFile(context.Background(), cfg, ff); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if fu.gotPoll != false {
		t.Fatalf("expected poll=false, got %v", fu.gotPoll)
	}
}

func TestUploadFile_FactoryError(t *testing.T) {
	cfg := UploadConfig{
		FilePath:         "/tmp/en.json",
		ProjectID:        "proj_123",
		Token:            "tok_abc",
		LangISO:          "en",
		GitHubRefName:    "main",
		MaxRetries:       1,
		InitialSleepTime: 1 * time.Second,
		MaxSleepTime:     5 * time.Second,
		HTTPTimeout:      10 * time.Second,
	}

	ff := &fakeUploadFactory{wantErr: errors.New("boom")}
	err := uploadFile(context.Background(), cfg, ff)
	if err == nil || !strings.Contains(err.Error(), "cannot create Lokalise API client") {
		t.Fatalf("expected factory error to propagate, got: %v", err)
	}
}

func TestUploadFile_UploadError(t *testing.T) {
	cfg := UploadConfig{
		FilePath:      "/tmp/en.json",
		ProjectID:     "proj_123",
		Token:         "tok_abc",
		LangISO:       "en",
		GitHubRefName: "main",
	}

	fu := &fakeUploader{returnErr: errors.New("network down")}
	ff := &fakeUploadFactory{uploader: fu}

	err := uploadFile(context.Background(), cfg, ff)
	if err == nil || !strings.Contains(err.Error(), "failed to upload file") {
		t.Fatalf("expected wrapped upload error, got: %v", err)
	}
}

// ---------- validate() / validateFile() tests ----------

func TestValidate_ExitsOnMissingFields(t *testing.T) {
	// missing file path
	requirePanicExit(t, func() {
		validate(UploadConfig{
			FilePath:      "",
			ProjectID:     "p",
			Token:         "t",
			LangISO:       "en",
			GitHubRefName: "ref",
		})
	})

	// missing project id
	requirePanicExit(t, func() {
		validate(UploadConfig{
			FilePath:      "/tmp/en.json",
			ProjectID:     "",
			Token:         "t",
			LangISO:       "en",
			GitHubRefName: "ref",
		})
	})

	// missing token
	requirePanicExit(t, func() {
		validate(UploadConfig{
			FilePath:      "/tmp/en.json",
			ProjectID:     "p",
			Token:         "",
			LangISO:       "en",
			GitHubRefName: "ref",
		})
	})

	// missing lang
	requirePanicExit(t, func() {
		validate(UploadConfig{
			FilePath:      "/tmp/en.json",
			ProjectID:     "p",
			Token:         "t",
			LangISO:       "",
			GitHubRefName: "ref",
		})
	})

	// missing GitHubRefName
	requirePanicExit(t, func() {
		validate(UploadConfig{
			FilePath:      "/tmp/en.json",
			ProjectID:     "p",
			Token:         "t",
			LangISO:       "en",
			GitHubRefName: "",
		})
	})
}

// ---------- fakes & helpers ----------

type fakeUploader struct {
	called    bool
	gotCtx    context.Context
	gotParams client.UploadParams
	gotPoll   bool

	returnPID string
	returnErr error
}

func (f *fakeUploader) Upload(ctx context.Context, params client.UploadParams, srcPath string, poll bool) (string, error) {
	f.called = true
	f.gotCtx = ctx
	f.gotParams = params
	f.gotPoll = poll
	return f.returnPID, f.returnErr
}

type fakeUploadFactory struct {
	wantErr error

	// capture args to assert
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
