#!/bin/bash

store_translation_paths() {
    > paths.txt

    local translations_path="$TRANSLATIONS_PATH"
    local flat_naming="$FLAT_NAMING"
    local base_lang="$BASE_LANG"
    local file_format="$FILE_FORMAT"

    while IFS= read -r path; do
        path=$(echo "$path" | xargs)
        if [ -n "$path" ]; then
            if [[ "$flat_naming" == "true" ]]; then
                echo "./${path}/${base_lang}.${file_format}" >> paths.txt
            else
                echo "./${path}/${base_lang}/**/*.${file_format}" >> paths.txt
            fi
        fi
    done <<< "$translations_path"
}

find_all_translation_files() {
    local paths=()
    while IFS= read -r path; do
        path=$(echo "$path" | xargs)
        if [ -n "$path" ]; then
            if [[ "$FLAT_NAMING" == "true" ]]; then
                target_file="${path}/${BASE_LANG}.${FILE_FORMAT}"
                if [ -f "$target_file" ]; then
                    paths+=("$target_file")
                else
                    echo "File $target_file does not exist."
                fi
            else
                target_dir="${path}/${BASE_LANG}"
                if [ -d "$target_dir" ]; then
                    paths+=("$target_dir")
                else
                    echo "Directory $target_dir does not exist."
                fi
            fi
        fi
    done <<< "$TRANSLATIONS_PATH"

    if [ ${#paths[@]} -eq 0 ]; then
        return 1
    fi

    if [[ "$FLAT_NAMING" == "true" ]]; then
        echo "DEBUG: Flat naming is true, setting ALL_FILES_ARRAY directly from paths"
        mapfile -t ALL_FILES_ARRAY <<< "${paths[@]}"
    else
        echo "DEBUG: Flat naming is false, finding files in directories"
        mapfile -t ALL_FILES_ARRAY < <(find "${paths[@]}" -name "*.${FILE_FORMAT}" -type f)
    fi

    if [ ${#ALL_FILES_ARRAY[@]} -eq 0 ]; then
        return 1
    else
        ALL_FILES=$(printf "%s," "${ALL_FILES_ARRAY[@]}")
        ALL_FILES="${ALL_FILES%,}"

        echo "ALL_FILES=$ALL_FILES" >> $GITHUB_OUTPUT
        return 0
    fi
}
