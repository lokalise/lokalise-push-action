package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
	"github.com/bodrovis/lokex/v2/client"
	"github.com/bodrovis/lokex/v2/client/upload"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

const (
	defaultMaxRetries       = 3   // Default number of retries on rate limits.
	defaultInitialSleepTime = 1   // Initial backoff in seconds; client applies exponential backoff.
	maxSleepTime            = 60  // Maximum backoff in seconds.
	defaultUploadTimeout    = 600 // Total timeout for a single upload in seconds.
	defaultHTTPTimeout      = 120 // Per-request HTTP timeout in seconds.
	defaultPollInitialWait  = 1   // Initial wait before the first poll in seconds.
	defaultPollMaxWait      = 120 // Total polling timeout in seconds.
)

// UploadConfig aggregates all inputs required to upload a single file.
type UploadConfig struct {
	FilePath         string
	ProjectID        string
	Token            string
	LangISO          string
	GitHubRefName    string
	AdditionalParams string

	SkipTagging      bool
	SkipPolling      bool
	SkipDefaultFlags bool

	MaxRetries       int
	InitialSleepTime time.Duration
	MaxSleepTime     time.Duration
	UploadTimeout    time.Duration
	HTTPTimeout      time.Duration
	PollInitialWait  time.Duration
	PollMaxWait      time.Duration
}

// Uploader abstracts the upload client for testability.
type Uploader interface {
	Upload(ctx context.Context, params upload.UploadParams, srcPath string, poll bool) (string, error)
}

// ClientFactory allows injecting a fake client in tests.
type ClientFactory interface {
	NewUploader(cfg UploadConfig) (Uploader, error)
}

type LokaliseFactory struct{}

// NewUploader wires lokex client with our retry, timeout, and polling settings.
func (f *LokaliseFactory) NewUploader(cfg UploadConfig) (Uploader, error) {
	lokaliseClient, err := client.NewClient(
		cfg.Token,
		cfg.ProjectID,
		client.WithMaxRetries(cfg.MaxRetries),
		client.WithHTTPTimeout(cfg.HTTPTimeout),
		client.WithBackoff(cfg.InitialSleepTime, cfg.MaxSleepTime),
		client.WithPollWait(cfg.PollInitialWait, cfg.PollMaxWait),
		client.WithUserAgent("lokalise-push-action/lokex"),
	)
	if err != nil {
		return nil, err
	}

	return upload.NewUploader(lokaliseClient), nil
}

func main() {
	// Require a single CLI arg: the file to upload.
	filePath := parseCLIArgs()

	cfg := prepareConfig(filePath)
	validate(cfg)

	// Scope the whole upload operation with a total timeout.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.UploadTimeout)
	defer cancel()

	if err := uploadFile(ctx, cfg, &LokaliseFactory{}); err != nil {
		returnWithError(err.Error())
	}
}

// parseCLIArgs validates the CLI input and returns the target file path.
func parseCLIArgs() string {
	if len(os.Args) < 2 {
		returnWithError("Usage: lokalise_upload <file>")
	}

	filePath := strings.TrimSpace(os.Args[1])
	if filePath == "" {
		returnWithError("File path is empty.")
	}

	return filePath
}

// validate performs input sanity checks before any network calls.
// It fails fast with actionable messages for CI logs.
func validate(cfg UploadConfig) {
	validateFile(cfg.FilePath)
	validateRequiredFields(cfg)
	validateTaggingInputs(cfg)
}

// validateRequiredFields checks the minimum required Lokalise settings.
func validateRequiredFields(cfg UploadConfig) {
	if cfg.ProjectID == "" {
		returnWithError("Project ID is required and cannot be empty.")
	}
	if cfg.Token == "" {
		returnWithError("API token is required and cannot be empty.")
	}
	if cfg.LangISO == "" {
		returnWithError("Base language (BASE_LANG) is required and cannot be empty.")
	}
}

// validateTaggingInputs ensures branch metadata is available when tagging is enabled.
func validateTaggingInputs(cfg UploadConfig) {
	if !cfg.SkipTagging && cfg.GitHubRefName == "" {
		returnWithError("GitHub reference name (GITHUB_REF_NAME) is required and cannot be empty.")
	}
}

// prepareConfig reads env vars, validates booleans, trims strings,
// and assembles an UploadConfig for the provided file path.
func prepareConfig(filePath string) UploadConfig {
	return UploadConfig{
		FilePath:         filePath,
		ProjectID:        strings.TrimSpace(os.Getenv("LOKALISE_PROJECT_ID")),
		Token:            strings.TrimSpace(os.Getenv("LOKALISE_API_TOKEN")),
		LangISO:          strings.TrimSpace(os.Getenv("BASE_LANG")),
		GitHubRefName:    strings.TrimSpace(os.Getenv("GITHUB_REF_NAME")),
		AdditionalParams: strings.TrimSpace(os.Getenv("ADDITIONAL_PARAMS")),

		SkipTagging:      parseRequiredBoolEnv("SKIP_TAGGING", "Invalid value for the skip_tagging parameter."),
		SkipPolling:      parseRequiredBoolEnv("SKIP_POLLING", "Invalid value for the skip_polling parameter."),
		SkipDefaultFlags: parseRequiredBoolEnv("SKIP_DEFAULT_FLAGS", "Invalid value for the skip_default_flags parameter."),

		MaxRetries:       parsers.ParseUintEnv("MAX_RETRIES", defaultMaxRetries),
		InitialSleepTime: time.Duration(parsers.ParseUintEnv("SLEEP_TIME", defaultInitialSleepTime)) * time.Second,
		MaxSleepTime:     time.Duration(maxSleepTime) * time.Second,
		UploadTimeout:    time.Duration(parsers.ParseUintEnv("UPLOAD_TIMEOUT", defaultUploadTimeout)) * time.Second,
		HTTPTimeout:      time.Duration(parsers.ParseUintEnv("HTTP_TIMEOUT", defaultHTTPTimeout)) * time.Second,
		PollInitialWait:  time.Duration(parsers.ParseUintEnv("POLL_INITIAL_WAIT", defaultPollInitialWait)) * time.Second,
		PollMaxWait:      time.Duration(parsers.ParseUintEnv("POLL_MAX_WAIT", defaultPollMaxWait)) * time.Second,
	}
}

// parseRequiredBoolEnv parses a boolean env var and exits with a custom message on failure.
func parseRequiredBoolEnv(key, errMsg string) bool {
	value, err := parsers.ParseBoolEnv(key)
	if err != nil {
		returnWithError(errMsg)
	}
	return value
}

// validateFile ensures the path exists and points to a regular file.
func validateFile(filePath string) {
	fi, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		returnWithError(fmt.Sprintf("File %s does not exist.", filePath))
	}
	if err != nil {
		returnWithError(fmt.Sprintf("Cannot stat file %s: %v", filePath, err))
	}
	if fi.IsDir() {
		returnWithError(fmt.Sprintf("Path %s is a directory, not a file.", filePath))
	}
}

// uploadFile builds upload params, creates a client, and performs the upload.
// Polling is enabled unless SkipPolling is true.
func uploadFile(ctx context.Context, cfg UploadConfig, factory ClientFactory) error {
	uploader, err := factory.NewUploader(cfg)
	if err != nil {
		return fmt.Errorf("cannot create Lokalise API client: %w", err)
	}

	params, err := buildUploadParams(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Starting to upload file %s\n", cfg.FilePath)

	if _, err := uploader.Upload(ctx, params, "", !cfg.SkipPolling); err != nil {
		return fmt.Errorf("failed to upload file %s: %w", cfg.FilePath, err)
	}

	return nil
}

// buildUploadParams assembles the payload for the Lokalise upload endpoint.
// AdditionalParams are merged last and may override defaults intentionally.
func buildUploadParams(cfg UploadConfig) (upload.UploadParams, error) {
	params := upload.UploadParams{
		"filename": cfg.FilePath,
		"lang_iso": cfg.LangISO,
	}

	applyDefaultFlags(params, cfg)
	applyTagging(params, cfg)

	if err := mergeAdditionalParams(params, cfg.AdditionalParams); err != nil {
		return nil, err
	}

	return params, nil
}

// applyDefaultFlags sets the default upload behavior used by this action.
func applyDefaultFlags(params upload.UploadParams, cfg UploadConfig) {
	if cfg.SkipDefaultFlags {
		return
	}

	params["replace_modified"] = true
	params["include_path"] = true
	params["distinguish_by_file"] = true
}

// applyTagging adds branch-based tags to inserted, skipped, and updated keys.
func applyTagging(params upload.UploadParams, cfg UploadConfig) {
	if cfg.SkipTagging {
		return
	}

	params["tag_inserted_keys"] = true
	params["tag_skipped_keys"] = true
	params["tag_updated_keys"] = true
	params["tags"] = []string{cfg.GitHubRefName}
}

// mergeAdditionalParams validates and merges user-provided params into the upload payload.
func mergeAdditionalParams(params upload.UploadParams, raw string) error {
	if err := parsers.ParseAdditionalParamsAndMerge(params, raw); err != nil {
		return fmt.Errorf("invalid additional_params (must be JSON object or YAML mapping): %w", err)
	}
	return nil
}

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
// Kept as a function var (exitFunc) to simplify unit testing without terminating the test runner.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
