# GitHub action to push changed translation files to Lokalise

![GitHub Release](https://img.shields.io/github/v/release/lokalise/lokalise-push-action)
![CI](https://github.com/lokalise/lokalise-push-action/actions/workflows/ci.yml/badge.svg)

GitHub action to upload changed translation files in the base language from your GitHub repository to [Lokalise TMS](https://lokalise.com/).

* Step-by-step tutorial covering the usage of this action is available on [Lokalise Developer Hub](https://developers.lokalise.com/docs/github-actions).
* If you're looking for an in-depth tutorial, [check out our blog post](https://lokalise.com/blog/github-actions-for-lokalise-translation/)

> To download translation files from Lokalise to GitHub, use the [lokalise-pull-action](https://github.com/lokalise/lokalise-pull-action).

## Usage

Use this action in the following way:

```yaml
name: Demo push with tags
on:
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Repo
        uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Push to Lokalise
        uses: lokalise/lokalise-push-action@v5.0.0
        with:
          api_token: ${{ secrets.LOKALISE_API_TOKEN }}
          project_id: LOKALISE_PROJECT_ID
          base_lang: en
          translations_path: |
            TRANSLATIONS_PATH1
            TRANSLATIONS_PATH2
          file_ext: |
            json
          additional_params: >
            {
              "convert_placeholders": true,
              "hidden_from_contributors": true
            }
```

## Configuration

You'll need to provide some parameters for the action. These can be set as environment variables, secrets, or passed directly. Refer to the [General setup](https://developers.lokalise.com/docs/github-actions#general-setup-overview) section for detailed instructions.

### Mandatory parameters

- `api_token` — Lokalise API token with read/write permissions.
  + Keep in mind that the API tokens are created on a per-user basis. If this contributor does not have proper access rights within a project (*Upload files* permission), the uploads will fail.
- `project_id` — Your Lokalise project ID.
- `translations_path` (*default: `locales`*) — One or more paths to your translations without leading and trailing slashes. For example, if your translations are stored in the `./locales/` folder at the project root, use `locales`.
- `file_format` — **⚠️ DEPRECATED**: use `file_ext` instead. *This will be removed in the next major release.* Defines the format of your translation files, such as `json` for JSON files. Defaults to `json`. This format determines how translation files are processed and also influences the file extension used when searching for them. However, some specific formats, such as `json_structured`, still use the generic `.json` extension. If you're using such a format, make sure to set the `file_ext` parameter explicitly to match the correct extension for your files. Alternatively, configure the `name_pattern` parameter.
- `base_lang` (*default: `en`*) — The base language of your project (e.g., `en` for English).
- `file_ext` (*default: `json`*) — File extension(s) to use when searching for translation files without leading dot. This parameter has no effect when the `name_pattern` is provided.

```yaml
file_ext: json

# OR

file_ext: |
  strings
  stringsdict
```

### File and API options

- `flat_naming` (*default: `false`*) — Use flat naming convention. Set to `true` if your translation files follow a flat naming pattern like `locales/en.json` instead of `locales/en/file.json`.
- `name_pattern` (*default: empty string*) — Custom pattern for naming translation files. Overrides default language-based naming. Must include both filename and extension if applicable (e.g., `"custom_name.json"` or `"**/*.yaml"`). Default behavior is used if not set.
  + When `name_pattern` is set, the action respects your `translations_path` but does not append language-based folders. For example:
    - `"en/**/custom_*.json"` will match nested files for the `en` locale
    - `"custom_*.json"` matches files directly under the given path
  This approach gives you fine-grained control similar to `flat_naming`, but with more flexibility.
- `additional_params` (*default: empty*) — Extra parameters to pass to the [Upload file API endpoint](https://developers.lokalise.com/reference/upload-a-file). Must contain valid JSON or YAML. Defaults to an empty string. Be careful when setting the `include_path` additional parameter to `false`, as it will mean your keys won't be assigned with any filename upon upload: this might pose a problem if you're planning to utilize the pull action to download translation back. You can include multiple API parameters as needed:

```yaml
additional_params: >
  {
    "convert_placeholders": true,
    "hidden_from_contributors": true
  }

# OR

additional_params: |
  convert_placeholders: true
  hidden_from_contributors: true
```

### Behavior settings

- `skip_tagging` (*default: `false`*) — Do not assign tags to the uploaded translation keys on Lokalise. Set this to `true` to skip adding tags like inserted, skipped, or updated keys.
- `skip_polling` (*default: `false`*) — Skips waiting for the upload operation to complete. When set to `true`, the `poll_initial_wait` and `poll_max_wait` parameters are ignored.
- `skip_default_flags` (*default: `false`*) — Prevents the action from setting additional default flags for the `upload` command. By default, the action includes `replace_modified`, `include_path`, and `distinguish_by_file` set to `true`. When `skip_default_flags` is `true`, these parameters are not added. Defaults to `false`.
- `rambo_mode` (*default: `false`*) — Always upload all translation files for the base language regardless of changes. Enable to bypass change detection and force a full upload of all base language translation files.
- `use_tag_tracking` (*default: `false`*) — Enables branch-specific sync tracking using Git tags. When set to `true`, the action creates a unique tag for each branch to remember the last successfully synced commit. On subsequent runs, it compares the current commit against the tagged commit to detect all changes since the last successful sync — regardless of how many commits occurred in between. This feature is still experimental.
  + By default, when `use_tag_tracking` is `false`, the action compares just the last two commits (`HEAD` and `HEAD~1`) to determine what changed. Enabling `use_tag_tracking` allows the action to detect broader changes across multiple commits and ensure nothing gets skipped during uploads.
  + This parameter has no effect if the `rambo_mode` is set to `true`.

### Retries and timeouts

- `max_retries` (*default: `3`*) — Maximum number of retries on rate limit (HTTP 429) and other retryable errors.
- `sleep_on_retry` (*default: `1`*) — Number of seconds to sleep before retrying on retryable errors (exponential backoff applies).
- `upload_timeout` (*default: `600`*) — Timeout for the whole upload operation, in seconds.
- `poll_initial_wait` (*default: `1`*) — Initial timeout for the upload poll operation, in seconds.
- `poll_max_wait` (*default: `120`*) — Maximum timeout for the upload poll operation, in seconds.
- `http_timeout` (*default: `120`*) — Timeout in seconds for every HTTP operation.

### Git configuration

- `git_user_name` (*default: empty string*) — Optional Git username to use when tagging the initial Lokalise upload. If not provided, the action will default to the GitHub actor who triggered the workflow. This is useful if you'd like to show a more descriptive or bot-specific name in your Git history (e.g., "Lokalise Sync Bot").
- `git_user_email` (*default: empty string*) — Optional Git email to associate with the Git tag for the initial Lokalise upload. If not set, the action will use a noreply address based on the username (e.g., `username@users.noreply.github.com`). Useful for customizing commit/tag authorship or when working in teams with dedicated automation accounts.

### Platform support

- `os_platform` (*default: empty — auto-detected*) — Platform for the precompiled binaries used by this action. If not set, the action automatically determines the correct platform based on the GitHub runner. You only need to set this manually when using unusual or self-hosted runners. In all other cases, auto-detection should work. Supported values:
  - `linux_amd64`
  - `linux_arm64`
  - `mac_amd64`
  - `mac_arm64`

## Technical details

### Outputs

This action outputs the following values:

- `initial_run` — Indicates whether this is the first run on the branch. The value is `true` if the `lokalise-upload-complete` tag does not exist, otherwise `false`.
- `files_uploaded` — Indicates whether any files were uploaded to Lokalise. The value is `true` if files were successfully uploaded, otherwise `false` (e.g., no changes or upload step skipped).

### Required permissions

This actions requires the following permissions:

```yaml
permissions:
  contents: write
```

### How this action works

When triggered, this action follows a multi-step process to detect changes in translation files and upload them to Lokalise:

1. **Detect changed files**:
   - The action identifies all changed translation files for the base language specified under the `translations_path`.
   - By default, changes are detected **only between the latest commit and the one preceding it**.
   - You can enable detection across multiple commits using the `use_tag_tracking` option:
     - When `use_tag_tracking` is set to `true`, the action compares the current commit with the last known synced commit on the branch (stored as a Git tag).
     - This ensures that any files changed across **multiple previous commits** are still uploaded, even when the action is run manually or after a batch push.

2. **Upload modified files**:
   - Any detected changes are uploaded to the specified Lokalise project in parallel, with up to six requests being processed simultaneously.
   - Each translation key is tagged with the name of the branch that triggered the workflow for better traceability in Lokalise. This also helps pulling your files back using the lokalise-pull action.

3. **Handle initial push**:
   - If no changes are detected, the action determines if it is running for the first time on the branch:
     - **First run**: The action checks for the presence of a `lokalise-upload-complete` tag.
       - If the tag is **not found**, it performs an initial upload, processing all translation files for the base language. This also happens when the `rambo_mode` is set to `true`.
       - After successfully uploading all files, the action creates a `lokalise-upload-complete` tag to mark the initial setup as complete.
     - **Subsequent runs**: If the tag is found and no new changes are detected (or no new commits when using `use_tag_tracking`), the action exits early without uploading any files.

4. **Track synced commits per branch** (optional):
   - When `use_tag_tracking` is enabled and files are uploaded, the action creates or updates a branch-specific tag named `lokalise-sync-<branch-name>` pointing to the latest synced commit.
   - This tag is used on future runs to determine the delta of changed files, preventing missed uploads.

5. **Mark completion**:
   - For the first run on the branch, after completing the initial upload, the action pushes the `lokalise-upload-complete` tag to the remote repository.
   - **Recommendation**: Pull the changes to your local repository to ensure the tag is included in your local Git history.

For more information on assumptions, refer to the [Assumptions and defaults](https://developers.lokalise.com/docs/github-actions#assumptions-and-defaults) section.

### Default parameters for the push action

By default, the following API parameters and headers are set when uploading files to Lokalise:

- `X-Api-Token` header — Derived from the `api_token` parameter.
- `project_id` GET param — Derived from the `project_id` parameter.
- `filename` — The currently uploaded file.
- `lang_iso` — The language ISO code of the translation file.
- `replace_modified` — Set to `true`.
- `include_path` — Set to `true`.
- `distinguish_by_file` — Set to `true`.
- `tag_inserted_keys` — Set to `true`.
- `tag_skipped_keys` — Set to `true`.
- `tag_updated_keys` — Set to `true`.
- `tags` — Set to the branch name that triggered the workflow.

## Checksums and attestation

You'll find checksums for the compiled binaries in the `bin/` directory. The checksums are also signed and attested. To verify, install Cosign, clone the repo, and run the following commands in the project root:

```
cosign verify-blob-attestation --bundle bin/checksums.txt.attestation --certificate-identity "https://github.com/lokalise/lokalise-push-action/.github/workflows/build-to-bin.yml@refs/heads/main" --certificate-oidc-issuer "https://token.actions.githubusercontent.com" --type custom bin/checksums.txt

cosign verify-blob --bundle bin/checksums.txt.sigstore --certificate-identity-regexp "^https://github.com/lokalise/lokalise-push-action/\.github/workflows/build-to-bin\.yml@.*$" --certificate-oidc-issuer "https://token.actions.githubusercontent.com" bin/checksums.txt
```

## License

Apache license version 2
