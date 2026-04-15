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

type uploaderFunc func(context.Context, UploadConfig, ClientFactory) error

func main() {
	if err := run(); err != nil {
		returnWithError(err.Error())
	}
}

func run() error {
	return runWith(
		os.Args,
		prepareConfig,
		validate,
		uploadFile,
		&LokaliseFactory{},
	)
}

func runWith(
	args []string,
	prepare func(string) (UploadConfig, error),
	validate func(UploadConfig) error,
	upload uploaderFunc,
	factory ClientFactory,
) error {
	filePath, err := parseCLIArgs(args)
	if err != nil {
		return err
	}

	cfg, err := prepare(filePath)
	if err != nil {
		return err
	}

	if err := validate(cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.UploadTimeout)
	defer cancel()

	return upload(ctx, cfg, factory)
}

// parseCLIArgs validates the CLI input and returns the target file path.
func parseCLIArgs(args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("usage: lokalise_upload <file>")
	}

	filePath := strings.TrimSpace(args[1])
	if filePath == "" {
		return "", fmt.Errorf("file path is empty")
	}

	return filePath, nil
}

// returnWithError prints an error message to stderr and exits the program with a non-zero status code.
func returnWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	exitFunc(1)
}
