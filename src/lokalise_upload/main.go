package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// exitFunc is a function variable that defaults to os.Exit.
// This can be overridden in tests to capture exit behavior.
var exitFunc = os.Exit

const (
	defaultMaxRetries    = 3   // Default number of retries if the upload is rate-limited
	defaultSleepTime     = 1   // Default initial sleep time in seconds between retries
	maxSleepTime         = 60  // Maximum sleep time in seconds between retries
	maxTotalTime         = 300 // Maximum total retry time in seconds
	defaultPollTimeout   = 120 // Upload poll timeout
	defaultUploadTimeout = 120 // Timeout for the upload itself
)

// UploadConfig holds all the necessary configuration for uploading a file
type UploadConfig struct {
	FilePath         string
	ProjectID        string
	Token            string
	LangISO          string
	GitHubRefName    string
	AdditionalParams string
	MaxRetries       int
	SleepTime        int
	PollTimeout      int
	UploadTimeout    int
}

func main() {
	// Ensure the required command-line arguments are provided
	if len(os.Args) < 4 {
		returnWithError("Usage: lokalise_upload <file> <project_id> <token>")
	}

	// Create the configuration struct
	config := UploadConfig{
		FilePath:         os.Args[1],
		ProjectID:        os.Args[2],
		Token:            os.Args[3],
		LangISO:          os.Getenv("BASE_LANG"),
		GitHubRefName:    os.Getenv("GITHUB_REF_NAME"),
		AdditionalParams: os.Getenv("CLI_ADD_PARAMS"),
		MaxRetries:       getEnvAsInt("MAX_RETRIES", defaultMaxRetries),
		SleepTime:        getEnvAsInt("SLEEP_TIME", defaultSleepTime),
		PollTimeout:      getEnvAsInt("UPLOAD_POLL_TIMEOUT", defaultPollTimeout),
		UploadTimeout:    getEnvAsInt("UPLOAD_TIMEOUT", defaultUploadTimeout),
	}

	validate(config)

	uploadFile(config, executeUpload)
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

// validateFile checks if the file exists and is not empty.
func validateFile(filePath string) {
	if filePath == "" {
		returnWithError("File path is required and cannot be empty.")
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		returnWithError(fmt.Sprintf("File %s does not exist.", filePath))
	}
}

// Call lokalise2 to upload files
func executeUpload(cmdPath string, args []string, uploadTimeout int) error {
	// Timeout for the lokalise2 call
	timeout := time.Duration(uploadTimeout) * time.Second

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// Check if the context was canceled due to timeout
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New("command timed out")
	}

	return err
}

// uploadFile uploads a file to Lokalise using the lokalise2 CLI tool.
// It handles rate limiting by retrying the upload with exponential backoff.
func uploadFile(config UploadConfig, uploadExecutor func(cmdPath string, args []string, uploadTimeout int) error) {
	fmt.Printf("Starting to upload file %s\n", config.FilePath)

	args := constructArgs(config)
	startTime := time.Now()

	sleepTime := config.SleepTime

	// Attempt to upload the file, retrying if rate-limited
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		fmt.Printf("Attempt %d of %d\n", attempt, config.MaxRetries)

		err := uploadExecutor("./bin/lokalise2", args, config.UploadTimeout)
		if err == nil {
			// Upload succeeded
			fmt.Printf("Successfully uploaded file %s\n", config.FilePath)
			return
		}

		// Check if the error is due to rate limiting (HTTP status code 429)
		if isRateLimitError(err) {
			// Sleep for the current sleep time before retrying
			time.Sleep(time.Duration(sleepTime) * time.Second)

			// Check if the total retry time has exceeded the maximum allowed time
			if time.Since(startTime).Seconds() > maxTotalTime {
				returnWithError(fmt.Sprintf("Max retry time exceeded (%d seconds) for %s. Exiting.", maxTotalTime, config.FilePath))
			}

			// Exponentially increase the sleep time for the next retry, capped at maxSleepTime
			sleepTime = min(sleepTime*2, maxSleepTime)
			continue // Retry the upload
		}

		// If the error is not due to rate limiting, exit with an error message
		returnWithError(fmt.Sprintf("Permanent error during upload for %s: %v", config.FilePath, err))
	}

	// If all retries have been exhausted, exit with an error message
	returnWithError(fmt.Sprintf("Failed to upload file %s after %d attempts.", config.FilePath, config.MaxRetries))
}

// constructArgs prepares the arguments for the lokalise2 CLI.
func constructArgs(config UploadConfig) []string {
	args := []string{
		fmt.Sprintf("--token=%s", config.Token),
		fmt.Sprintf("--project-id=%s", config.ProjectID),
		"file", "upload",
		fmt.Sprintf("--file=%s", config.FilePath),
		fmt.Sprintf("--lang-iso=%s", config.LangISO),
		"--replace-modified",
		"--include-path",
		"--distinguish-by-file",
		"--poll",
		fmt.Sprintf("--poll-timeout=%ds", config.PollTimeout),
		"--tag-inserted-keys",
		"--tag-skipped-keys=true",
		"--tag-updated-keys",
		"--tags", config.GitHubRefName,
	}

	// Add additional params from the environment variable
	if config.AdditionalParams != "" {
		args = append(args, strings.Fields(config.AdditionalParams)...)
	}

	return args
}

// getEnvAsInt retrieves an environment variable as an integer.
// Returns the default value if the variable is not set.
// Exits with an error if the value is not a positive integer.
func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val < 1 {
		fmt.Printf("Invalid or missing value for %s, using default: %d\n", key, defaultVal)
		return defaultVal
	}
	return val
}

// isRateLimitError checks if the error is due to rate limiting (HTTP status code 429).
func isRateLimitError(err error) bool {
	return strings.Contains(err.Error(), "API request error 429")
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
