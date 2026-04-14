package main

import (
	"fmt"

	"github.com/bodrovis/lokalise-actions-common/v2/parsers"
	"github.com/bodrovis/lokex/v2/client/upload"
)

// buildUploadParams assembles the payload for the Lokalise upload endpoint.
// AdditionalParams are merged last and may override defaults intentionally.
func buildUploadParams(cfg UploadConfig) (upload.UploadParams, error) {
	params := upload.UploadParams{
		"filename": cfg.FilePath,
		"lang_iso": cfg.LangISO,
	}

	applyDefaultFlags(params, cfg)
	applyTagging(params, cfg)

	if err := mergeAdditionalParams(params, cfg.AdditionalParams); err != nil {
		return nil, err
	}

	return params, nil
}

// applyDefaultFlags sets the default upload behavior used by this action.
func applyDefaultFlags(params upload.UploadParams, cfg UploadConfig) {
	if cfg.SkipDefaultFlags {
		return
	}

	params["replace_modified"] = true
	params["include_path"] = true
	params["distinguish_by_file"] = true
}

// applyTagging adds branch-based tags to inserted, skipped, and updated keys.
func applyTagging(params upload.UploadParams, cfg UploadConfig) {
	if cfg.SkipTagging {
		return
	}

	params["tag_inserted_keys"] = true
	params["tag_skipped_keys"] = true
	params["tag_updated_keys"] = true
	params["tags"] = []string{cfg.GitHubRefName}
}

// mergeAdditionalParams validates and merges user-provided params into the upload payload.
func mergeAdditionalParams(params upload.UploadParams, raw string) error {
	if err := parsers.ParseAdditionalParamsAndMerge(params, raw); err != nil {
		return fmt.Errorf("invalid additional_params (must be JSON object or YAML mapping): %w", err)
	}
	return nil
}
