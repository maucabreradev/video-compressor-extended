#!/bin/bash
# ==============================================================================
# Input validation functions: configuration, resolution, FPS, hardware checks.
# ==============================================================================

if [[ -n "${_VALIDATION_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _VALIDATION_SH_INCLUDED=1

validate_flags() {
    if [[ "$DO_COMPRESS" == "false" ]] && [[ "$DO_SUBTITLE" == "false" ]]; then
        print_error "ERROR: Cannot use --no-compress and --no-sub simultaneously."
        print_info "The script would have no tasks to perform."
        exit 1
    fi
}

validate_resolution() {
    local resolution="$1"

    if [[ ! "$resolution" =~ ^[0-9]+x[0-9]+$ ]]; then
        print_error "ERROR: Invalid resolution format '$resolution'."
        print_info "Expected format: WIDTHxHEIGHT (e.g., 1280x720)"
        exit 1
    fi
}

validate_fps() {
    local fps="$1"

    if [[ ! "$fps" =~ ^[0-9]+$ ]] || [[ "$fps" -lt 1 ]] || [[ "$fps" -gt 240 ]]; then
        print_error "ERROR: Invalid FPS value '$fps'."
        print_info "FPS must be a positive integer between 1 and 240."
        exit 1
    fi
}

validate_language() {
    local lang="$1"

    if [[ ! "$lang" =~ ^[a-z]{2}$ ]]; then
        print_error "ERROR: Invalid language code '$lang'."
        print_info "Language must be a 2-letter ISO code (e.g., en, es, fr)."
        exit 1
    fi
}

parse_resolution() {
    local resolution="$1"
    WIDTH_VAL="${resolution%x*}"
    HEIGHT_VAL="${resolution#*x}"
}

find_videos() {
    shopt -s nullglob
    shopt -s nocaseglob
    shopt -s globstar

    local -ga found_videos
    found_videos=()

    for ext in mp4 mkv avi mov flv wmv webm ts m4v; do
        for f in **/*."$ext"; do
            [[ "$f" != "$OUTPUT_DIR"/* ]] && found_videos+=("$f")
        done
    done

    if [[ ${#found_videos[@]} -eq 0 ]]; then
        print_error "ERROR: No compatible video files found in $(pwd)"
        exit 1
    fi

    printf '%s\n' "${found_videos[@]}"
}
