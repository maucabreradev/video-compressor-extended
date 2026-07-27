#!/bin/bash
# ==============================================================================
# Video compression using VAAPI hardware acceleration.
# Uses atomic file operations (.processing) to prevent corrupted files
# if the process is interrupted.
# ==============================================================================

if [[ -n "${_COMPRESSION_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _COMPRESSION_SH_INCLUDED=1

compress_video() {
    local input="$1"
    local output="$2"
    local width="$3"
    local height="$4"
    local fps="$5"

    local temp_output="${output}.processing"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would compress: $input -> $output (${width}x${height} @ ${fps}fps)"
        return 0
    fi

    register_temp_file "$temp_output"
    log_info "Compressing: $input (${width}x${height} @ ${fps}fps)"

    if ffmpeg -y \
        -vaapi_device /dev/dri/renderD128 \
        -i "$input" \
        -vf "format=nv12,hwupload,scale_vaapi=w=${width}:h=${height}" \
        -r "$fps" \
        -c:v hevc_vaapi \
        -qp 24 \
        -c:a aac \
        -b:a 128k \
        "$temp_output" 2>/dev/null; then

        mv "$temp_output" "$output"
        log_info "Compression completed: $output"
        return 0
    else
        log_error "Compression failed: $input"
        return 1
    fi
}

check_vaapi_device() {
    if [[ ! -e "/dev/dri/renderD128" ]]; then
        log_warn "/dev/dri/renderD128 not found — hardware acceleration may fail"
        return 1
    fi
    return 0
}

get_compression_args() {
    local output="$1"
    local width="$2"
    local height="$3"
    local fps="$4"

    echo "-vaapi_device /dev/dri/renderD128"
    echo "-vf format=nv12,hwupload,scale_vaapi=w=${width}:h=${height}"
    echo "-r $fps"
    echo "-c:v hevc_vaapi -qp 24 -c:a aac -b:a 128k"
    echo "$output"
}
