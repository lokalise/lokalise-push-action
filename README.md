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
        uses: lokalise/lokalise-push-action@v3.0.0
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

#### **Mandatory Parameters**

- `api_token` — Lokalise API token with read/write permissions.
- `project_id` — Your Lokalise project ID.
- `translations_path` — One or more paths to your translations. For example, if your translations are stored in the `locales` folder at the project root, use `locales` (leave out leading and trailing slashes). Defaults to `locales`.
- `file_format` — Translation file format. For example, if you're using JSON files, just put `json` (no leading dot needed). Defaults to `json`.
- `base_lang` — The base language of your project (e.g., `en` for English). Defaults to `en`.

#### **Optional Parameters**

- `additional_params` — Extra parameters to pass to the [Lokalise CLI when pushing files](https://github.com/lokalise/lokalise-cli-2-go/blob/main/docs/lokalise2_file_upload.md). For example, you can use `--convert-placeholders` to handle placeholders. You can include multiple CLI arguments as needed. Defaults to an empty string.
- `flat_naming` — Use flat naming convention. Set to `true` if your translation files follow a flat naming pattern like `locales/en.json` instead of `locales/en/file.json`. Defaults to `false`.
- `max_retries` — Maximum number of retries on rate limit errors (HTTP 429). Defaults to `3`.
- `sleep_on_retry` — Number of seconds to sleep before retrying on rate limit errors. Defaults to `1`.
- `upload_timeout` — Timeout for the upload operation, in seconds. Defaults to `120`.
- `upload_poll_timeout` — Timeout for the upload poll operation, in seconds. Defaults to `120`.

## Technical details

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
       - If the tag is **not found**, it performs an initial upload, processing all translation files for the base language.
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
- `--tag-skipped-keys=true`
- `--tag-updated-keys`
- `--tags` — Set to the branch name that triggered the workflow.

## License

Apache license version 2