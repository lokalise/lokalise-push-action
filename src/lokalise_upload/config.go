package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
)

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

// prepareConfig reads env vars, validates booleans, trims strings,
// and assembles an UploadConfig for the provided file path.
func prepareConfig(filePath string) (UploadConfig, error) {
	skipTagging, err := parseBoolEnv("SKIP_TAGGING")
	if err != nil {
		return UploadConfig{}, err
	}

	skipPolling, err := parseBoolEnv("SKIP_POLLING")
	if err != nil {
		return UploadConfig{}, err
	}

	skipDefaultFlags, err := parseBoolEnv("SKIP_DEFAULT_FLAGS")
	if err != nil {
		return UploadConfig{}, err
	}

	githubRefName := strings.TrimSpace(os.Getenv("GITHUB_HEAD_REF"))
	if githubRefName == "" {
		githubRefName = strings.TrimSpace(os.Getenv("GITHUB_REF_NAME"))
	}

	return UploadConfig{
		FilePath:         filePath,
		ProjectID:        strings.TrimSpace(os.Getenv("LOKALISE_PROJECT_ID")),
		Token:            strings.TrimSpace(os.Getenv("LOKALISE_API_TOKEN")),
		LangISO:          strings.TrimSpace(os.Getenv("BASE_LANG")),
		GitHubRefName:    githubRefName,
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
	}, nil
}

func parseBoolEnv(key string) (bool, error) {
	value, err := parsers.ParseBoolEnv(key)
	if err != nil {
		return false, fmt.Errorf("invalid %s: expected true or false: %w", key, err)
	}
	return value, nil
}
