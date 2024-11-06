package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxRetries = 3
	defaultSleepTime  = 1
	maxSleepTime      = 60
	maxTotalTime      = 300
)

func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	os.Exit(1)
}

func uploadFile(file, projectID, token string) {
	langISO := getEnv("BASE_LANG", true)
	additionalParams := os.Getenv("CLI_ADD_PARAMS")
	githubRefName := os.Getenv("GITHUB_REF_NAME")
	maxRetries := getEnvAsInt("MAX_RETRIES", defaultMaxRetries)
	sleepTime := getEnvAsInt("SLEEP_TIME", defaultSleepTime)

	// Ensure essential arguments and file existence
	validateFile(file)
	if projectID == "" || token == "" {
		returnWithError("Project ID and token are required and cannot be empty.")
	}

	fmt.Printf("Starting upload for %s\n", file)
	startTime := time.Now()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Attempt %d of %d\n", attempt, maxRetries)

		// Build command arguments
		args := []string{
			fmt.Sprintf("--token=%s", token),
			fmt.Sprintf("--project-id=%s", projectID),
			"file", "upload",
			fmt.Sprintf("--file=%s", file),
			fmt.Sprintf("--lang-iso=%s", langISO),
			"--replace-modified", "--include-path", "--distinguish-by-file",
			"--poll", "--poll-timeout=120s",
			"--tag-inserted-keys", "--tag-skipped-keys=true",
			"--tag-updated-keys", "--tags", githubRefName,
		}

		if additionalParams != "" {
			args = append(args, strings.Fields(additionalParams)...)
		}

		cmd := exec.Command("./bin/lokalise2", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run the command and handle output
		err := cmd.Run()
		if err == nil {
			fmt.Printf("Successfully uploaded %s\n", file)
			return
		}

		// Handle API rate limit error
		if isRateLimitError(err) {
			fmt.Printf("Rate limit reached on attempt %d for %s. Retrying in %d seconds...\n", attempt, file, sleepTime)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			if time.Since(startTime).Seconds() > maxTotalTime {
				returnWithError(fmt.Sprintf("Max retry time exceeded (%d seconds) for %s. Exiting.", maxTotalTime, file))
			}
			sleepTime = min(sleepTime*2, maxSleepTime) // Exponential backoff with max limit
			continue
		}

		// For other errors, exit immediately
		returnWithError(fmt.Sprintf("Permanent error during upload for %s: %v", file, err))
	}

	returnWithError(fmt.Sprintf("Failed to upload file %s after %d attempts.", file, maxRetries))
}

func main() {
	if len(os.Args) < 4 {
		returnWithError("Usage: lokalise_upload <file> <project_id> <token>")
	}

	file := os.Args[1]
	projectID := os.Args[2]
	token := os.Args[3]

	uploadFile(file, projectID, token)
}

// Helper functions

func getEnv(key string, required bool) string {
	val := os.Getenv(key)
	if required && val == "" {
		returnWithError(fmt.Sprintf("Environment variable %s is required.", key))
	}
	return val
}

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

func validateFile(file string) {
	if file == "" {
		returnWithError("File path is required and cannot be empty.")
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		returnWithError(fmt.Sprintf("File %s does not exist.", file))
	}
}

func isRateLimitError(err error) bool {
	return strings.Contains(err.Error(), "API request error 429")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
