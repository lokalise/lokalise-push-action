package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type (
	prepareFunc  func(string) (UploadConfig, error)
	validateFunc func(UploadConfig) error
	uploaderFunc func(context.Context, UploadConfig, ClientFactory) error
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	return 0
}

func run() error {
	return runWith(
		os.Args,
		prepareConfig,
		validate,
		uploadFile,
		LokaliseFactory{},
	)
}

func runWith(
	args []string,
	prepare prepareFunc,
	validate validateFunc,
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
		return "", errors.New("usage: lokalise_upload <file>")
	}

	filePath := args[1]
	if strings.TrimSpace(filePath) == "" {
		return "", errors.New("file path is empty")
	}

	return filePath, nil
}
