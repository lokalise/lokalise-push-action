package githuboutput

import (
	"fmt"
	"os"
)

// WriteToGitHubOutput writes a key-value pair to the GITHUB_OUTPUT file
func WriteToGitHubOutput(name, value string) error {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable is not set")
	}

	file, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening GITHUB_OUTPUT file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	if err != nil {
		return fmt.Errorf("error writing to GITHUB_OUTPUT file: %w", err)
	}

	return nil
}
