package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// exitFunc is a function variable that defaults to os.Exit.
// Overridable in tests to assert exit behavior without terminating the process.
var exitFunc = os.Exit

func main() {
	if err := run(); err != nil {
		returnWithError(err.Error())
	}
}

func run() error {
	filePath := parseCLIArgs()
	cfg := prepareConfig(filePath)
	validate(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.UploadTimeout)
	defer cancel()

	return uploadFile(ctx, cfg, &LokaliseFactory{})
}

// parseCLIArgs validates the CLI input and returns the target file path.
func parseCLIArgs() string {
	if len(os.Args) != 2 {
		returnWithError("Usage: lokalise_upload <file>")
	}

	filePath := strings.TrimSpace(os.Args[1])
	if filePath == "" {
		returnWithError("File path is empty.")
	}

	return filePath
}

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
// Kept as a function var (exitFunc) to simplify unit testing without terminating the test runner.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
