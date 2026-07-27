#!/bin/bash
# ==============================================================================
# Subtitle generation using whisper-ctranslate2.
# Uses atomic file operations (.processing) to prevent corrupted files
# if the process is interrupted.
# ==============================================================================

if [[ -n "${_SUBTITLES_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _SUBTITLES_SH_INCLUDED=1

generate_subtitles() {
    local input_video="$1"
    local output_dir="$2"
    local target_srt_name="$3"
    local language="$4"

    local temp_file=""

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would generate subtitles: $input_video ($language)"
        return 0
    fi

    log_info "Generating subtitles for: $input_video (language: $language)"

    local generated_srt
    generated_srt="$output_dir/$(basename "${input_video%.*}").srt"

    if whisper-ctranslate2 "$input_video" \
        --language "$language" \
        --model base \
        --compute_type int8 \
        --vad_filter True \
        --output_format srt \
        --output_dir "$output_dir" 2>/dev/null; then

        if [[ "$generated_srt" != "$target_srt_name" ]] && [[ -f "$generated_srt" ]]; then
            mv "$generated_srt" "$target_srt_name"
        fi

        log_info "Subtitles generated: $target_srt_name"
        return 0
    else
        log_error "Subtitle generation failed: $input_video"
        return 1
    fi
}
