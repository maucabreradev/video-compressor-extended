#!/bin/bash
# ==============================================================================
# Utility functions: cleanup, helpers, temporary file management.
# ==============================================================================

if [[ -n "${_UTILS_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _UTILS_SH_INCLUDED=1

declare -a _TEMP_FILES=()
declare -a _TEMP_DIRS=()

check_bash_version() {
    if [[ "${BASH_VERSINFO[0]}" -lt 4 ]]; then
        echo "ERROR: Bash 4.0 or higher is required (found ${BASH_VERSION})" >&2
        exit 1
    fi
}

register_temp_file() {
    _TEMP_FILES+=("$1")
}

register_temp_dir() {
    _TEMP_DIRS+=("$1")
}

cleanup() {
    trap - SIGINT SIGTERM EXIT HUP

    echo ""
    echo "${RED}Process interrupted by user.${NC}"

    if [[ -n "${LOG_FILE:-}" ]]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] Process interrupted - cleaning up temporary files" >> "$LOG_FILE"
    fi

    for temp_file in "${_TEMP_FILES[@]}"; do
        if [[ -f "$temp_file" ]]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] Removing incomplete file: $temp_file" >> "$LOG_FILE" 2>/dev/null
            rm -f "$temp_file"
        fi
    done

    for temp_dir in "${_TEMP_DIRS[@]}"; do
        if [[ -d "$temp_dir" ]]; then
            rm -rf "$temp_dir"
        fi
    done

    find "$OUTPUT_DIR" -type d -empty -delete 2>/dev/null

    kill -- -$$ 2>/dev/null
    exit 130
}

init_traps() {
    trap cleanup SIGINT SIGTERM HUP
}

format_duration() {
    local seconds="$1"
    local hours=$((seconds / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local secs=$((seconds % 60))

    if [[ "$hours" -gt 0 ]]; then
        printf '%dh %dm %ds' "$hours" "$minutes" "$secs"
    elif [[ "$minutes" -gt 0 ]]; then
        printf '%dm %ds' "$minutes" "$secs"
    else
        printf '%ds' "$secs"
    fi
}

hr() {
    printf '%*s\n' "${COLUMNS:-80}" '' | tr ' ' '='
}

finalize_video() {
    local processed_video="$1"
    local processed_srt="$2"
    local original_dir="$3"
    local original_filename="$4"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would move processed files to $original_dir"
        return 0
    fi

    if [[ -f "$processed_video" ]]; then
        mv "$processed_video" "$original_dir/"
        log_info "Moved compressed video to: $original_dir"
    fi

    if [[ -f "$processed_srt" ]]; then
        mv "$processed_srt" "$original_dir/"
        log_info "Moved subtitles to: $original_dir"
    fi

    if [[ "$REMOVE_ORIGINAL" == "true" ]]; then
        rm -f "$original_dir/$original_filename"
        log_info "Removed original: $original_filename"
    fi
}
