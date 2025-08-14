package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
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
	SkipTagging      bool
	SkipPolling      bool
	SkipDefaultFlags bool
	MaxRetries       int
	SleepTime        int
	PollTimeout      int
	UploadTimeout    int
}

// ringBuffer keeps only the last N bytes written.
type ringBuffer struct {
	buf   []byte
	limit int
}

func newRingBuffer(n int) *ringBuffer { return &ringBuffer{limit: n} }

func (r *ringBuffer) Write(p []byte) (int, error) {
	if r.limit <= 0 {
		return len(p), nil
	}
	// If incoming chunk is bigger than limit, keep only its tail.
	if len(p) >= r.limit {
		r.buf = append(r.buf[:0], p[len(p)-r.limit:]...)
		return len(p), nil
	}
	need := len(r.buf) + len(p) - r.limit
	if need > 0 {
		// Drop the oldest bytes to make room.
		r.buf = r.buf[need:]
	}
	r.buf = append(r.buf, p...)
	return len(p), nil
}

func (r *ringBuffer) String() string { return string(r.buf) }

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
		AdditionalParams: os.Getenv("CLI_ADD_PARAMS"),
		SkipTagging:      skipTagging,
		SkipPolling:      skipPolling,
		SkipDefaultFlags: skipDefaultFlags,
		MaxRetries:       parsers.ParseUintEnv("MAX_RETRIES", defaultMaxRetries),
		SleepTime:        parsers.ParseUintEnv("SLEEP_TIME", defaultSleepTime),
		PollTimeout:      parsers.ParseUintEnv("UPLOAD_POLL_TIMEOUT", defaultPollTimeout),
		UploadTimeout:    parsers.ParseUintEnv("UPLOAD_TIMEOUT", defaultUploadTimeout),
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

// executeUpload runs the command with a timeout, streams output to CI logs,
// and returns an error that includes the last chunk of stderr (capped).
func executeUpload(cmdPath string, args []string, uploadTimeout int) error {
	timeout := time.Duration(uploadTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, args...)

	// Stream to job logs, but keep a capped copy of stderr for the error message.
	const stderrMax = 64 * 1024 // last 64 KB
	rb := newRingBuffer(stderrMax)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, rb)

	err := cmd.Run()

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		tail := strings.TrimSpace(rb.String())
		if tail != "" {
			return fmt.Errorf("command timed out after %ds: %s", uploadTimeout, tail)
		}
		return fmt.Errorf("command timed out after %ds", uploadTimeout)
	}

	if err != nil {
		// Attach exit code if we have it, plus the stderr tail.
		var ee *exec.ExitError
		exit := ""
		if errors.As(err, &ee) {
			exit = fmt.Sprintf(" (exit %d)", ee.ExitCode())
		}
		tail := strings.TrimSpace(rb.String())
		if tail != "" {
			return fmt.Errorf("command failed%s: %s: %w", exit, tail, err)
		}
		return fmt.Errorf("command failed%s: %w", exit, err)
	}

	return nil
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
			fmt.Printf("Successfully uploaded file %s\n", config.FilePath)
			return
		}

		if isRetryableError(err) {
			if attempt == config.MaxRetries {
				returnWithError(fmt.Sprintf(
					"Failed to upload file %s after %d attempts. Last error: %v",
					config.FilePath, config.MaxRetries, err,
				))
			}

			// Will we exceed the max total time by sleeping before the next attempt?
			elapsed := time.Since(startTime)
			if elapsed+time.Duration(sleepTime)*time.Second >= time.Duration(maxTotalTime)*time.Second {
				returnWithError(fmt.Sprintf(
					"Max retry time exceeded (%d seconds) for %s. Exiting.",
					maxTotalTime, config.FilePath,
				))
			}

			fmt.Printf("Retryable error detected (%v), sleeping %ds before retry...\n", err, sleepTime)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			sleepTime = min(sleepTime*2, maxSleepTime)
			continue
		}

		// Non-retryable: fail fast.
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
	}

	if !config.SkipDefaultFlags {
		args = append(args, "--replace-modified", "--include-path", "--distinguish-by-file")
	}

	if !config.SkipPolling {
		args = append(args, "--poll", fmt.Sprintf("--poll-timeout=%ds", config.PollTimeout))
	}

	if !config.SkipTagging {
		args = append(args, "--tag-inserted-keys", "--tag-skipped-keys", "--tag-updated-keys", "--tags", config.GitHubRefName)
	}

	if config.AdditionalParams != "" {
		scanner := bufio.NewScanner(strings.NewReader(config.AdditionalParams))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				args = append(args, line)
			}
		}
		if err := scanner.Err(); err != nil {
			returnWithError(fmt.Sprintf("Failed to parse additional parameters: %v", err))
		}
	}

	return args
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if isRateLimitError(err) {
		return true
	}

	// Lowercase the message for case-insensitive matching
	msg := strings.ToLower(err.Error())

	// Timeouts or transient network failures
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "time exceeded") ||
		strings.Contains(msg, "polling time exceeded limit") {
		return true
	}

	return false
}

// isRateLimitError checks if the error is due to rate limiting (HTTP status code 429).
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "api request error 429") ||
		strings.Contains(s, "request error 429") ||
		strings.Contains(s, "rate limit")
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
