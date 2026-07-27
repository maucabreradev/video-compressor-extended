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

    if [[ -n "${LOG_FILE:-}" ]]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] Process interrupted - cleaning up" >> "$LOG_FILE"
    fi

    for temp_file in "${_TEMP_FILES[@]}"; do
        if [[ -f "$temp_file" ]]; then
            rm -f "$temp_file"
        fi
    done

    for temp_dir in "${_TEMP_DIRS[@]}"; do
        if [[ -d "$temp_dir" ]]; then
            rm -rf "$temp_dir"
        fi
    done

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
