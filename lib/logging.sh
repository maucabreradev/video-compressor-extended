#!/bin/bash
# ==============================================================================
# Logging system with file output, log levels, and automatic rotation.
# ==============================================================================

if [[ -n "${_LOGGING_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _LOGGING_SH_INCLUDED=1

LOG_FILE=""
LOG_LEVEL="INFO"
LOG_ENABLED=true

readonly _LOG_LEVELS=(
    ["DEBUG"]=0
    ["INFO"]=1
    ["WARN"]=2
    ["ERROR"]=3
)

_log_level_value() {
    local level="$1"
    echo "${_LOG_LEVELS[$level]:-1}"
}

log_init() {
    if [[ "$LOG_ENABLED" == "false" ]]; then
        return
    fi

    if [[ -z "$LOG_FILE" ]]; then
        LOG_FILE="logs/vcx_$(date '+%Y%m%d_%H%M%S').log"
    fi

    local log_dir
    log_dir="$(dirname "$LOG_FILE")"
    if [[ ! -d "$log_dir" ]]; then
        mkdir -p "$log_dir"
    fi

    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] [$$] VCX session started" >> "$LOG_FILE"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] [$$] Log file: $LOG_FILE" >> "$LOG_FILE"
}

_should_log() {
    local level="$1"
    [[ "$LOG_ENABLED" == "true" ]] || return 1
    [[ $(_log_level_value "$level") -ge $(_log_level_value "$LOG_LEVEL") ]]
}

log() {
    local level="$1"
    local message="$2"

    if [[ -z "$LOG_FILE" ]]; then
        return 0
    fi

    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    if _should_log "$level"; then
        local log_dir
        log_dir="$(dirname "$LOG_FILE")"
        if [[ ! -d "$log_dir" ]]; then
            mkdir -p "$log_dir" 2>/dev/null || true
        fi
        printf '[%s] [%-5s] [%s] %s\n' "$timestamp" "$level" "$$" "$message" >> "$LOG_FILE" 2>/dev/null
    fi
}

log_debug() {
    log "DEBUG" "$1"
}

log_info() {
    log "INFO" "$1"
}

log_warn() {
    log "WARN" "$1"
}

log_error() {
    log "ERROR" "$1"
}

log_rotate() {
    local max_logs="${1:-10}"
    local log_dir="${LOG_FILE%/*}"

    if [[ ! -d "$log_dir" ]]; then
        return
    fi

    local count
    count=$(find "$log_dir" -maxdepth 1 -name "vcx_*.log" -type f 2>/dev/null | wc -l)

    if [[ "$count" -gt "$max_logs" ]]; then
        find "$log_dir" -maxdepth 1 -name "vcx_*.log" -type f -printf '%T@ %p\n' \
            | sort -n \
            | head -n $((count - max_logs)) \
            | cut -d' ' -f2- \
            | xargs rm -f
        log_debug "Rotated logs: removed $((count - max_logs)) old files"
    fi
}

log_summary() {
    if [[ -z "$LOG_FILE" ]] || [[ ! -f "$LOG_FILE" ]]; then
        return
    fi

    local processed=0
    local failed=0
    local skipped=0

    processed=$(grep -c "Completed:" "$LOG_FILE" 2>/dev/null || echo 0)
    failed=$(grep -c "ERROR:.*failed" "$LOG_FILE" 2>/dev/null || echo 0)
    skipped=$(grep -c "Skipping:" "$LOG_FILE" 2>/dev/null || echo 0)

    echo ""
    hr
    echo "Session log: $LOG_FILE"
    echo "  Processed : $processed"
    echo "  Failed    : $failed"
    echo "  Skipped   : $skipped"
    hr
}
