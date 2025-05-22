# GitHub action to push changed translation files to Lokalise

GitHub action to upload changed translation files in the base language from your GitHub repository to [Lokalise TMS](https://lokalise.com/).

**Step-by-step tutorial covering the usage of this action is available on [Lokalise Developer Hub](https://developers.lokalise.com/docs/github-actions).** To download translation files from Lokalise to GitHub, use the [lokalise-pull-action](https://github.com/lokalise/lokalise-pull-action).

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
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Push to Lokalise
        uses: lokalise/lokalise-push-action@v3.6.0
        with:
          api_token: ${{ secrets.LOKALISE_API_TOKEN }}
          project_id: LOKALISE_PROJECT_ID
          base_lang: en
          translations_path: |
            TRANSLATIONS_PATH1
            TRANSLATIONS_PATH2
          file_format: json
          additional_params: |
            --convert-placeholders
            --hidden-from-contributors
```

## Configuration

You'll need to provide some parameters for the action. These can be set as environment variables, secrets, or passed directly. Refer to the [General setup](https://developers.lokalise.com/docs/github-actions#general-setup-overview) section for detailed instructions.

### Mandatory parameters

- `api_token` — Lokalise API token with read/write permissions.
- `project_id` — Your Lokalise project ID.
- `translations_path` — One or more paths to your translations. For example, if your translations are stored in the `locales` folder at the project root, use `locales` (leave out leading and trailing slashes). Defaults to `locales`.
- `file_format` — Defines the format of your translation files, such as `json` for JSON files. Defaults to `json`. This format determines how translation files are processed and also influences the file extension used when searching for them. However, some specific formats, such as `json_structured`, still use the generic `.json` extension. If you're using such a format, make sure to set the `file_ext` parameter explicitly to match the correct extension for your files. Alternatively, configure the `name_pattern` parameter.
- `base_lang` — The base language of your project (e.g., `en` for English). Defaults to `en`.
- `file_ext` (*not strictly mandatory but still recommended*) — Custom file extension to use when searching for translation files (without leading dot, for example `yml`). By default, the extension is inferred from the `file_format` value. However, for certain formats (e.g., `json_structured`), the files may still have a generic extension (e.g., `.json`). In such cases, this parameter allows specifying the correct extension manually to ensure proper file matching. This parameter has no effect when the `name_pattern` is provided.

### File and CLI options

- `additional_params` — Extra parameters to pass to the [Lokalise CLI when pushing files](https://github.com/lokalise/lokalise-cli-2-go/blob/main/docs/lokalise2_file_upload.md). For example, you can use `--convert-placeholders` to handle placeholders. Defaults to an empty string. Be careful when setting the `include-path` additional parameter to `false`, as it will mean your keys won't be assigned with any filename upon upload: this might pose a problem if you're planning to utilize the pull action to download translation back. You can include multiple CLI arguments as needed:

```yaml
additional_params: |
  --convert-placeholders
  --hidden-from-contributors
```

- `flat_naming` — Use flat naming convention. Set to `true` if your translation files follow a flat naming pattern like `locales/en.json` instead of `locales/en/file.json`. Defaults to `false`.
- `name_pattern` — Custom pattern for naming translation files. Overrides default language-based naming. Must include both filename and extension if applicable (e.g., `"custom_name.json"` or `"**.yaml"`). Default behavior is used if not set.  
  + When `name_pattern` is set, the action respects your `translations_path` but does not append language-based folders. For example:  
    - `"en/**/custom_*.json"` will match nested files for the `en` locale  
    - `"custom_*.json"` matches files directly under the given path  
  This approach gives you fine-grained control similar to `flat_naming`, but with more flexibility.

### Behavior and retry settings

- `skip_tagging` — Do not assign tags to the uploaded translation keys on Lokalise. Set this to `true` to skip adding tags like inserted, skipped, or updated keys. Defaults to `false`.
- `skip_polling` — Skips waiting for the upload operation to complete. When set to `true`, the `upload_poll_timeout` parameter is ignored. Defaults to `false`.
- `skip_default_flags` — Prevents the action from setting additional default flags for the `upload` command. By default, the action includes `--replace-modified --include-path --distinguish-by-file`. When `skip_default_flags` is `true`, these flags are not added. Defaults to `false`.
- `rambo_mode` — Always upload all translation files for the base language regardless of changes. Set this to `true` to bypass change detection and force a full upload of all base language translation files. Defaults to `false`.
- `use_tag_tracking` — Enables branch-specific sync tracking using Git tags. When set to `true`, the action creates a unique tag for each branch to remember the last successfully synced commit. On subsequent runs, it compares the current commit against the tagged commit to detect all changes since the last successful sync — regardless of how many commits occurred in between. This feature is still experimental.
  + By default, when `use_tag_tracking` is `false`, the action compares just the last two commits (`HEAD` and `HEAD~1`) to determine what changed. Enabling `use_tag_tracking` allows the action to detect broader changes across multiple commits and ensure nothing gets skipped during uploads.
  + This parameter has no effect if the `rambo_mode` is set to `true`.
- `max_retries` — Maximum number of retries on rate limit errors (HTTP 429). Defaults to `3`.
- `sleep_on_retry` — Number of seconds to sleep before retrying on rate limit errors. Defaults to `1`.
- `upload_timeout` — Timeout for the upload operation, in seconds. Defaults to `120`.
- `upload_poll_timeout` — Timeout for the upload poll operation, in seconds. Defaults to `120`.

### Git configuration

- `git_user_name` — Optional Git username to use when tagging the initial Lokalise upload. If not provided, the action will default to the GitHub actor that triggered the workflow. This is useful if you'd like to show a more descriptive or bot-specific name in your Git history (e.g., "Lokalise Sync Bot").
- `git_user_email` — Optional Git email to associate with the Git tag for the initial Lokalise upload. If not set, the action will use a noreply address based on the username (e.g., `username@users.noreply.github.com`). Useful for customizing commit/tag authorship or when working in teams with dedicated automation accounts.

### Platform support

- `os_platform` — Target platform for the precompiled binaries used by this action (`linux_amd64`, `linux_arm64`, `mac_amd64`, `mac_arm64`). These binaries handle tasks like uploading and processing translations. Typically, you don't need to change this, as the default (`linux_amd64`) works for most environments. Override if running on a macOS runner or a different architecture.

## Technical details

### Outputs

This action outputs the following values:

- `initial_run` — Indicates whether this is the first run on the branch. The value is `true` if the `lokalise-upload-complete` tag does not exist, otherwise `false`.
- `files_uploaded` — Indicates whether any files were uploaded to Lokalise. The value is `true` if files were successfully uploaded, otherwise `false` (e.g., no changes or upload step skipped).

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

By default, the following command-line parameters are set when uploading files to Lokalise:

- `--token` — Derived from the `api_token` parameter.
- `--project-id` — Derived from the `project_id` parameter.
- `--file` — The currently uploaded file.
- `--lang-iso` — The language ISO code of the translation file.
- `--replace-modified`
- `--include-path`
- `--distinguish-by-file`
- `--poll`
- `--poll-timeout` — Derived from the `upload_poll_timeout` parameter.
- `--tag-inserted-keys`
- `--tag-skipped-keys`
- `--tag-updated-keys`
- `--tags` — Set to the branch name that triggered the workflow.

## License

Apache license version 2
