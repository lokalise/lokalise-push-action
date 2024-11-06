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
	langISO := os.Getenv("BASE_LANG")
	additionalParams := os.Getenv("CLI_ADD_PARAMS")
	githubRefName := os.Getenv("GITHUB_REF_NAME")
	maxRetries := getEnvAsInt("MAX_RETRIES", defaultMaxRetries)
	sleepTime := getEnvAsInt("SLEEP_TIME", defaultSleepTime)

	validateFile(file)
	if projectID == "" || token == "" || langISO == "" {
		returnWithError("Project ID, API token, and base language are required and cannot be empty.")
	}
	if githubRefName == "" {
		returnWithError("GITHUB_REF_NAME is required and cannot be empty.")
	}

	startTime := time.Now()

	for attempt := 1; attempt <= maxRetries; attempt++ {
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
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			return
		}

		if isRateLimitError(err) {
			time.Sleep(time.Duration(sleepTime) * time.Second)
			if time.Since(startTime).Seconds() > maxTotalTime {
				returnWithError(fmt.Sprintf("Max retry time exceeded (%d seconds) for %s. Exiting.", maxTotalTime, file))
			}
			sleepTime = min(sleepTime*2, maxSleepTime)
			continue
		}

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
