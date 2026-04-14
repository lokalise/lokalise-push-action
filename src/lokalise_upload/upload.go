package main

import (
	"context"
	"fmt"

	"github.com/bodrovis/lokex/v2/client"
	"github.com/bodrovis/lokex/v2/client/upload"
)

// Uploader abstracts the upload client for testability.
type Uploader interface {
	Upload(ctx context.Context, params upload.UploadParams, srcPath string, poll bool) (string, error)
}

// ClientFactory allows injecting a fake client in tests.
type ClientFactory interface {
	NewUploader(cfg UploadConfig) (Uploader, error)
}

type LokaliseFactory struct{}

// NewUploader wires lokex client with our retry, timeout, and polling settings.
func (f *LokaliseFactory) NewUploader(cfg UploadConfig) (Uploader, error) {
	lokaliseClient, err := client.NewClient(
		cfg.Token,
		cfg.ProjectID,
		client.WithMaxRetries(cfg.MaxRetries),
		client.WithHTTPTimeout(cfg.HTTPTimeout),
		client.WithBackoff(cfg.InitialSleepTime, cfg.MaxSleepTime),
		client.WithPollWait(cfg.PollInitialWait, cfg.PollMaxWait),
		client.WithUserAgent("lokalise-push-action/lokex"),
	)
	if err != nil {
		return nil, err
	}

	return upload.NewUploader(lokaliseClient), nil
}

// uploadFile builds upload params, creates a client, and performs the upload.
// Polling is enabled unless SkipPolling is true.
func uploadFile(ctx context.Context, cfg UploadConfig, factory ClientFactory) error {
	params, err := buildUploadParams(cfg)
	if err != nil {
		return err
	}

	uploader, err := factory.NewUploader(cfg)
	if err != nil {
		return fmt.Errorf("cannot create Lokalise API client: %w", err)
	}

	fmt.Printf("Starting to upload file %q\n", cfg.FilePath)

	if _, err := uploader.Upload(ctx, params, "", !cfg.SkipPolling); err != nil {
		return fmt.Errorf("failed to upload file %q: %w", cfg.FilePath, err)
	}

	return nil
}
