package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxRetries = 3   // Default number of retries if the upload is rate-limited
	defaultSleepTime  = 1   // Default initial sleep time in seconds between retries
	maxSleepTime      = 60  // Maximum sleep time in seconds between retries
	maxTotalTime      = 300 // Maximum total retry time in seconds
)

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	os.Exit(1)
}

// uploadFile uploads a file to Lokalise using the lokalise2 CLI tool.
// It handles rate limiting by retrying the upload with exponential backoff.
func uploadFile(filePath, projectID, token string) {
	// Retrieve necessary environment variables
	langISO := os.Getenv("BASE_LANG")
	additionalParams := os.Getenv("CLI_ADD_PARAMS")
	githubRefName := os.Getenv("GITHUB_REF_NAME")
	maxRetries := getEnvAsInt("MAX_RETRIES", defaultMaxRetries)
	sleepTime := getEnvAsInt("SLEEP_TIME", defaultSleepTime)

	// Validate required inputs
	validateFile(filePath)
	if projectID == "" || token == "" || langISO == "" {
		returnWithError("Project ID, API token, and base language are required and cannot be empty.")
	}
	if githubRefName == "" {
		returnWithError("GITHUB_REF_NAME is required and cannot be empty.")
	}

	fmt.Printf("Starting to upload file %s\n", filePath)

	startTime := time.Now()

	// Attempt to upload the file, retrying if rate-limited
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Attempt %d of %d\n", attempt, maxRetries)

		// Construct command arguments for the lokalise2 CLI tool
		args := []string{
			fmt.Sprintf("--token=%s", token),
			fmt.Sprintf("--project-id=%s", projectID),
			"file", "upload",
			fmt.Sprintf("--file=%s", filePath),
			fmt.Sprintf("--lang-iso=%s", langISO),
			"--replace-modified",
			"--include-path",
			"--distinguish-by-file",
			"--poll",
			"--poll-timeout=120s",
			"--tag-inserted-keys",
			"--tag-skipped-keys=true",
			"--tag-updated-keys",
			"--tags", githubRefName,
		}

		// Append any additional parameters specified in the environment variable
		if additionalParams != "" {
			args = append(args, strings.Fields(additionalParams)...)
		}

		// Execute the command to upload the file
		cmd := exec.Command("./bin/lokalise2", args...)
		cmd.Stdout = io.Discard // Discard standard output
		cmd.Stderr = os.Stderr  // Redirect standard error to stderr

		err := cmd.Run()
		if err == nil {
			// Upload succeeded
			fmt.Printf("Successfully uploaded file %s\n", filePath)
			return
		}

		// Check if the error is due to rate limiting (HTTP status code 429)
		if isRateLimitError(err) {
			// Sleep for the current sleep time before retrying
			time.Sleep(time.Duration(sleepTime) * time.Second)

			// Check if the total retry time has exceeded the maximum allowed time
			if time.Since(startTime).Seconds() > maxTotalTime {
				returnWithError(fmt.Sprintf("Max retry time exceeded (%d seconds) for %s. Exiting.", maxTotalTime, filePath))
			}

			// Exponentially increase the sleep time for the next retry, capped at maxSleepTime
			sleepTime = min(sleepTime*2, maxSleepTime)
			continue // Retry the upload
		}

		// If the error is not due to rate limiting, exit with an error message
		returnWithError(fmt.Sprintf("Permanent error during upload for %s: %v", filePath, err))
	}

	// If all retries have been exhausted, exit with an error message
	returnWithError(fmt.Sprintf("Failed to upload file %s after %d attempts.", filePath, maxRetries))
}

func main() {
	// Ensure the required command-line arguments are provided
	if len(os.Args) < 4 {
		returnWithError("Usage: lokalise_upload <file> <project_id> <token>")
	}

	filePath := os.Args[1]
	projectID := os.Args[2]
	token := os.Args[3]

	// Start the file upload process
	uploadFile(filePath, projectID, token)
}

// Helper functions

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
		returnWithError(fmt.Sprintf("Environment variable %s must be a positive integer.", key))
	}
	return val
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
