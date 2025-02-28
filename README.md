# GitHub action to push changed translation files from Lokalise

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
        uses: lokalise/lokalise-push-action@v3.1.1
        with:
          api_token: ${{ secrets.LOKALISE_API_TOKEN }}
          project_id: LOKALISE_PROJECT_ID
          base_lang: BASE_LANG_ISO
          translations_path: |
            TRANSLATIONS_PATH1
            TRANSLATIONS_PATH2
          file_format: FILE_FORMAT
          additional_params: ADDITIONAL_CLI_PARAMS
```

## Configuration

### Parameters

You'll need to provide some parameters for the action. These can be set as environment variables, secrets, or passed directly. Refer to the [General setup](https://developers.lokalise.com/docs/github-actions#general-setup-overview) section for detailed instructions.

#### Mandatory parameters

- `api_token` — Lokalise API token with read/write permissions.
- `project_id` — Your Lokalise project ID.
- `translations_path` — One or more paths to your translations. For example, if your translations are stored in the `locales` folder at the project root, use `locales` (leave out leading and trailing slashes). Defaults to `locales`.
- `file_format` — Defines the format of your translation files, such as `json` for JSON files. Defaults to `json`. This format determines how translation files are processed and also influences the file extension used when searching for them. However, some specific formats, such as `json_structured`, still use the generic `.json` extension. If you're using such a format, make sure to set the `file_ext` parameter explicitly to match the correct extension for your files. Alternatively, configure the `name_pattern` parameter.
- `base_lang` — The base language of your project (e.g., `en` for English). Defaults to `en`.

#### Optional parameters

- `file_ext` — Custom file extension to use when searching for translation files (without leading dot). By default, the extension is inferred from the file_format value. However, for certain formats (e.g., `json_structured`), the files may still have a generic extension (e.g., `.json`). In such cases, this parameter allows specifying the correct extension manually to ensure proper file matching. This parameter has no effect when the `name_pattern` is provided.
- `additional_params` — Extra parameters to pass to the [Lokalise CLI when pushing files](https://github.com/lokalise/lokalise-cli-2-go/blob/main/docs/lokalise2_file_upload.md). For example, you can use `--convert-placeholders` to handle placeholders. You can include multiple CLI arguments as needed. Defaults to an empty string.
- `flat_naming` — Use flat naming convention. Set to `true` if your translation files follow a flat naming pattern like `locales/en.json` instead of `locales/en/file.json`. Defaults to `false`.
- `name_pattern` — Custom pattern for naming translation files. Overrides default language-based naming. Must include both filename and extension if applicable (e.g., "custom_name.json" or "**.yaml"). Default behavior is used if not set.
  + When the `name_pattern` is set, the action will respect your `translations_path` but won't append any language names as folders. Therefore, if you want to upload all JSON files with custom naming for the English locale, you'll need to provide `name_pattern: "en/**/custom_*.json"`. To upload all JSON files stored directly under `translations_path`, you'll set `name_pattern: "custom_*.json"`. The latter approach is similar to `flat_naming` but enables you to define custom patterns.
- `skip_tagging` — Do not assign tags to the uploaded translation keys on Lokalise. Set this to `true` to skip adding tags like inserted, skipped, or updated keys. Defaults to `false`.
- `skip_polling` — Skips waiting for the upload operation to complete. When set to `true`, the `upload_poll_timeout` parameter is ignored. Defaults to `false`.
- `skip_default_flags` — Prevents the action from setting additional default flags for the `upload` command. By default, the action includes `--replace-modified --include-path --distinguish-by-file`. When `skip_default_flags` is `true`, these flags are not added. Defaults to `false`.
- `rambo_mode` — Always upload all translation files for the base language regardless of changes. Set this to `true` to bypass change detection and force a full upload of all base language translation files. Defaults to `false`.
- `max_retries` — Maximum number of retries on rate limit errors (HTTP 429). Defaults to `3`.
- `sleep_on_retry` — Number of seconds to sleep before retrying on rate limit errors. Defaults to `1`.
- `upload_timeout` — Timeout for the upload operation, in seconds. Defaults to `120`.
- `upload_poll_timeout` — Timeout for the upload poll operation, in seconds. Defaults to `120`.
- `platform` — Target platform for the precompiled binaries used by this action (`linux_amd64`, `linux_arm64`, `mac_amd64`, `mac_arm64`). These binaries handle tasks like uploading and processing translations. Typically, you don't need to change this, as the default (`linux_amd64`) works for most environments. Override if running on a macOS runner or a different architecture.

## Technical details

### Outputs

This action outputs the following values:

- `initial_run` — Indicates whether this is the first run on the branch. The value is `true` if the `lokalise-upload-complete` tag does not exist, otherwise `false`.
- `files_uploaded` — Indicates whether any files were uploaded to Lokalise. The value is `true` if files were successfully uploaded, otherwise `false` (e.g., no changes or upload step skipped).

### How this action works

When triggered, this action follows a multi-step process to detect changes in translation files and upload them to Lokalise:

1. **Detect changed files**:
   - The action identifies all changed translation files for the base language specified under the `translations_path`.
   - **Important**: Changes are detected **only between the latest commit and the one preceding it**. This ensures that the action processes incremental changes rather than scanning the entire repository.

2. **Upload modified files**:
   - Any detected changes are uploaded to the specified Lokalise project in parallel, with up to six requests being processed simultaneously.
   - Each translation key is tagged with the name of the branch that triggered the workflow for better traceability in Lokalise.

3. **Handle initial push**:
   - If no changes are detected between the last two commits, the action determines if it is running for the first time on the branch:
     - **First run**: The action checks for the presence of a `lokalise-upload-complete` tag.
       - If the tag is **not found**, it performs an initial upload, processing all translation files for the base language. This also happens when the `rambo_mode` is set to `true`.
       - After successfully uploading all files, the action creates a `lokalise-upload-complete` tag to mark the initial setup as complete.
     - **Subsequent runs**: If the tag is found, the action exits without uploading any files, as the initial push has already been completed.

4. **Mark completion**:
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