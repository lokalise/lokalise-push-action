package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
	"github.com/bodrovis/lokex/client"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

const (
	defaultMaxRetries       = 3   // Default number of retries on rate limits
	defaultInitialSleepTime = 1   // Initial backoff (seconds); client handles exponential backoff
	maxSleepTime            = 60  // Backoff cap (seconds)
	defaultUploadTimeout    = 600 // Total timeout for a single upload (seconds)
	defaultHTTPTimeout      = 120 // Per-request HTTP timeout (seconds)
	defaultPollInitialWait  = 1   // Initial wait before first poll of async job (seconds)
	defaultPollMaxWait      = 120 // Polling overall timeout (seconds)
)

// UploadConfig aggregates all inputs required to upload a single file.
type UploadConfig struct {
	FilePath         string        // Absolute or relative path to the file on disk
	ProjectID        string        // Lokalise project ID
	Token            string        // Lokalise token
	LangISO          string        // Base language code (e.g., en, fr_FR)
	GitHubRefName    string        // Current ref/branch; used for tagging keys
	AdditionalParams string        // JSON object with extra API params (merged last)
	SkipTagging      bool          // Do not tag keys on upload
	SkipPolling      bool          // Return early without waiting for async processing
	SkipDefaultFlags bool          // Do not set our default flags (replace_modified/include_path/…)
	MaxRetries       int           // Client retry count for retryable errors
	InitialSleepTime time.Duration // Backoff start
	MaxSleepTime     time.Duration // Backoff cap
	UploadTimeout    time.Duration // Overall timeout for this upload
	HTTPTimeout      time.Duration // Per-request timeout
	PollInitialWait  time.Duration // First poll delay
	PollMaxWait      time.Duration // Polling timeout
}

// Uploader abstracts the upload client for testability.
type Uploader interface {
	Upload(ctx context.Context, params client.UploadParams, poll bool) (string, error)
}

// ClientFactory allows injecting a fake client in tests.
type ClientFactory interface {
	NewUploader(cfg UploadConfig) (Uploader, error)
}

type LokaliseFactory struct{}

// NewUploader wires lokex client with our timeouts/backoff/polling config.
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
	return client.NewUploader(lokaliseClient), nil
}

func main() {
	// Require a single CLI arg: the file to upload.
	if len(os.Args) < 2 {
		returnWithError("Usage: lokalise_upload <file>")
	}

	config := prepareConfig(os.Args[1])
	validate(config)

	// Scope the whole operation with a total timeout.
	ctx, cancel := context.WithTimeout(context.Background(), config.UploadTimeout)
	defer cancel()

	if err := uploadFile(ctx, config, &LokaliseFactory{}); err != nil {
		returnWithError(err.Error())
	}
}

// validate performs input sanity checks before any network calls.
// It fails fast with a helpful message for CI logs.
func validate(config UploadConfig) {
	validateFile(config.FilePath)

	if config.ProjectID == "" {
		returnWithError("Project ID is required and cannot be empty.")
	}
	if config.Token == "" {
		returnWithError("API token is required and cannot be empty.")
	}
	if config.LangISO == "" {
		returnWithError("Base language (BASE_LANG) is required and cannot be empty.")
	}
	if config.GitHubRefName == "" {
		returnWithError("GitHub reference name (GITHUB_REF_NAME) is required and cannot be empty.")
	}
}

// prepareConfig reads env vars, validates booleans, trims strings,
// and assembles an UploadConfig for the provided file path.
func prepareConfig(filePath string) UploadConfig {
	skipTagging, err := parsers.ParseBoolEnv("SKIP_TAGGING")
	if err != nil {
		returnWithError("Invalid value for the skip_tagging parameter.")
	}

	skipPolling, err := parsers.ParseBoolEnv("SKIP_POLLING")
	if err != nil {
		returnWithError("Invalid value for the skip_polling parameter.")
	}

	skipDefaultFlags, err := parsers.ParseBoolEnv("SKIP_DEFAULT_FLAGS")
	if err != nil {
		returnWithError("Invalid value for the skip_default_flags parameter.")
	}

	filePath = strings.TrimSpace(filePath)
	if len(filePath) == 0 {
		returnWithError("File path is empty.")
	}

	return UploadConfig{
		FilePath:         filePath,
		ProjectID:        strings.TrimSpace(os.Getenv("LOKALISE_PROJECT_ID")),
		Token:            strings.TrimSpace(os.Getenv("LOKALISE_API_TOKEN")),
		LangISO:          strings.TrimSpace(os.Getenv("BASE_LANG")),
		GitHubRefName:    strings.TrimSpace(os.Getenv("GITHUB_REF_NAME")),
		AdditionalParams: strings.TrimSpace(os.Getenv("ADDITIONAL_PARAMS")),
		SkipTagging:      skipTagging,
		SkipPolling:      skipPolling,
		SkipDefaultFlags: skipDefaultFlags,
		MaxRetries:       parsers.ParseUintEnv("MAX_RETRIES", defaultMaxRetries),
		InitialSleepTime: time.Duration(parsers.ParseUintEnv("SLEEP_TIME", defaultInitialSleepTime)) * time.Second,
		MaxSleepTime:     time.Duration(maxSleepTime) * time.Second,
		UploadTimeout:    time.Duration(parsers.ParseUintEnv("UPLOAD_TIMEOUT", defaultUploadTimeout)) * time.Second,
		HTTPTimeout:      time.Duration(parsers.ParseUintEnv("HTTP_TIMEOUT", defaultHTTPTimeout)) * time.Second,
		PollInitialWait:  time.Duration(parsers.ParseUintEnv("POLL_INITIAL_WAIT", defaultPollInitialWait)) * time.Second,
		PollMaxWait:      time.Duration(parsers.ParseUintEnv("POLL_MAX_WAIT", defaultPollMaxWait)) * time.Second,
	}
}

// validateFile ensures the path exists and is a regular file (not a dir).
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

// uploadFile builds the API params and performs the upload (optionally polling for completion).
func uploadFile(ctx context.Context, cfg UploadConfig, factory ClientFactory) error {
	uploader, err := factory.NewUploader(cfg)
	if err != nil {
		return fmt.Errorf("cannot create Lokalise API client: %w", err)
	}

	params := buildUploadParams(cfg)

	fmt.Printf("Starting to upload file %s\n", cfg.FilePath)

	if _, err := uploader.Upload(ctx, params, !cfg.SkipPolling); err != nil {
		return fmt.Errorf("failed to upload file %s: %w", cfg.FilePath, err)
	}

	return nil
}

// buildUploadParams assembles the payload for the Lokalise upload endpoint.
// AdditionalParams (JSON) are merged last and can override defaults if needed.
func buildUploadParams(config UploadConfig) client.UploadParams {
	params := client.UploadParams{
		"filename": config.FilePath,
		"lang_iso": config.LangISO,
	}

	// Reasonable defaults that work well for CI-driven uploads.
	if !config.SkipDefaultFlags {
		params["replace_modified"] = true    // overwrite modified keys from file
		params["include_path"] = true        // include file path for better key scoping
		params["distinguish_by_file"] = true // treat same keys in different files distinctly
	}

	// Tagging helps trace inserted/updated/skipped keys to a branch/ref.
	if !config.SkipTagging {
		params["tag_inserted_keys"] = true
		params["tag_skipped_keys"] = true
		params["tag_updated_keys"] = true
		params["tags"] = []string{config.GitHubRefName}
	}

	// Merge arbitrary extra params from JSON (caller-controlled).
	ap := strings.TrimSpace(config.AdditionalParams)
	if ap != "" {
		add, err := parseJSONMap(ap)
		if err != nil {
			returnWithError("Invalid additional_params (must be JSON object): " + err.Error())
		}
		maps.Copy(params, add) // last write wins
	}

	return params
}

func parseJSONMap(s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
// Kept as a function var (exitFunc) to simplify unit testing without terminating the test runner.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
