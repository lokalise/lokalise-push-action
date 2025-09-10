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
// This can be overridden in tests to capture exit behavior.
var exitFunc = os.Exit

const (
	defaultMaxRetries       = 3   // Default number of retries if the upload is rate-limited
	defaultInitialSleepTime = 1   // Default initial sleep time in seconds between retries
	maxSleepTime            = 60  // Maximum sleep time in seconds between retries
	defaultUploadTimeout    = 600 // Timeout for the upload itself
	defaultHTTPTimeout      = 120
	defaultPollInitialWait  = 1
	defaultPollMaxWait      = 120
)

// UploadConfig holds all the necessary configuration for uploading a file
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

type Uploader interface {
	Upload(ctx context.Context, params client.UploadParams, poll bool) (string, error)
}

type ClientFactory interface {
	NewUploader(cfg UploadConfig) (Uploader, error)
}

type LokaliseFactory struct{}

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
	// Ensure the required command-line arguments are provided
	if len(os.Args) < 4 {
		returnWithError("Usage: lokalise_upload <file> <project_id> <token>")
	}

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

	// Create the configuration struct
	config := UploadConfig{
		FilePath:         os.Args[1],
		ProjectID:        os.Args[2],
		Token:            os.Args[3],
		LangISO:          os.Getenv("BASE_LANG"),
		GitHubRefName:    os.Getenv("GITHUB_REF_NAME"),
		AdditionalParams: os.Getenv("ADDITIONAL_PARAMS"),
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

	validate(config)

	ctx, cancel := context.WithTimeout(context.Background(), config.UploadTimeout)
	defer cancel()

	err = uploadFile(ctx, config, &LokaliseFactory{})
	if err != nil {
		returnWithError(err.Error())
	}
}

// validate checks if the configuration is valid and contains all necessary fields.
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

// validateFile checks if the file exists
func validateFile(filePath string) {
	if filePath == "" {
		returnWithError("File path is required and cannot be empty.")
	}

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

func buildUploadParams(config UploadConfig) client.UploadParams {
	params := client.UploadParams{
		"filename": config.FilePath,
		"lang_iso": config.LangISO,
	}

	if !config.SkipDefaultFlags {
		params["replace_modified"] = true
		params["include_path"] = true
		params["distinguish_by_file"] = true
	}

	if !config.SkipTagging {
		params["tag_inserted_keys"] = true
		params["tag_skipped_keys"] = true
		params["tag_updated_keys"] = true
		params["tags"] = []string{config.GitHubRefName}
	}

	ap := strings.TrimSpace(config.AdditionalParams)
	if ap != "" {
		add, err := parseJSONMap(ap)
		if err != nil {
			returnWithError("Invalid additional_params (must be JSON object): " + err.Error())
		}
		maps.Copy(params, add)
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
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
