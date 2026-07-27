#!/bin/bash
# ==============================================================================
# Video Compressor Extended (VCX)
# Batch video compression and subtitle generation using VAAPI and Whisper.
# ==============================================================================

set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/lib/colors.sh"
source "${SCRIPT_DIR}/lib/logging.sh"
source "${SCRIPT_DIR}/lib/utils.sh"
source "${SCRIPT_DIR}/lib/config.sh"
source "${SCRIPT_DIR}/lib/validation.sh"
source "${SCRIPT_DIR}/lib/dependencies.sh"
source "${SCRIPT_DIR}/lib/compression.sh"
source "${SCRIPT_DIR}/lib/subtitles.sh"

main() {
    local start_time
    start_time=$(date +%s)

    parse_arguments "$@"
    load_config_file
    validate_flags
    check_dependencies

    if [[ "$DO_COMPRESS" == "true" ]]; then
        validate_resolution "$RESOLUTION"
        validate_fps "$FPS"
        parse_resolution "$RESOLUTION"
        check_vaapi_device
    fi

    if [[ "$DO_SUBTITLE" == "true" ]]; then
        validate_language "$LANG_CODE"
    fi

    log_init
    log_info "VCX v${VCX_VERSION} started"
    log_info "Configuration: compress=$DO_COMPRESS subtitle=$DO_SUBTITLE remove_original=$REMOVE_ORIGINAL"
    log_rotate 10

    init_traps

    if [[ "$REMOVE_ORIGINAL" == "true" ]]; then
        OUTPUT_DIR=".vcx_temp"
    else
        OUTPUT_DIR="vcx_processed"
    fi

    mkdir -p "$OUTPUT_DIR"
    register_temp_dir "$OUTPUT_DIR"

    show_config_summary

    local -a videos
    mapfile -t videos < <(find_videos)

    if [[ ${#videos[@]} -eq 0 ]]; then
        print_error "ERROR: No compatible video files found."
        exit 1
    fi

    print_info "Found ${#videos[@]} video(s) to process."

    local MAX_JOBS=2
    local -a pids=()
    local processed_count=0
    local failed_count=0
    local skipped_count=0

    for video in "${videos[@]}"; do
        local dir_name filename filename_no_ext target_sub_dir
        dir_name=$(dirname -- "$video")
        filename=$(basename -- "$video")
        filename_no_ext="${filename%.*}"

        if [[ "$REMOVE_ORIGINAL" == "true" ]]; then
            target_sub_dir="${OUTPUT_DIR}/${dir_name}"
        else
            target_sub_dir="${OUTPUT_DIR}/${dir_name}"
        fi
        mkdir -p "$target_sub_dir"

        local target_video srt_file
        if [[ "$DO_COMPRESS" == "true" ]]; then
            target_video="${target_sub_dir}/${filename_no_ext}_${RESOLUTION}.mp4"
            srt_file="${target_sub_dir}/${filename_no_ext}_${RESOLUTION}.srt"
        else
            target_video="$video"
            srt_file="${target_sub_dir}/${filename_no_ext}.srt"
        fi

        if [[ "$DO_COMPRESS" == "true" ]] && [[ "$DO_SUBTITLE" == "false" ]] && [[ -f "$target_video" ]]; then
            print_warning "Skipping: $filename (compressed already exists)"
            log_info "Skipping $filename: already compressed"
            ((skipped_count++))
            continue
        elif [[ "$DO_COMPRESS" == "false" ]] && [[ "$DO_SUBTITLE" == "true" ]] && [[ -f "$srt_file" ]]; then
            print_warning "Skipping: $filename (subtitles already exist)"
            log_info "Skipping $filename: subtitles already exist"
            ((skipped_count++))
            continue
        fi

        while [[ ${#pids[@]} -ge $MAX_JOBS ]]; do
            local -a new_pids=()
            for pid in "${pids[@]}"; do
                if kill -0 "$pid" 2>/dev/null; then
                    new_pids+=("$pid")
                fi
            done
            pids=("${new_pids[@]}")

            if [[ ${#pids[@]} -ge $MAX_JOBS ]]; then
                sleep 0.5
            fi
        done

        (
            trap "" SIGINT SIGTERM

            print_bold "Processing: $filename"
            log_info "Processing: $video"

            local success=true

            if [[ "$DO_COMPRESS" == "true" ]]; then
                if ! compress_video "$video" "$target_video" "$WIDTH_VAL" "$HEIGHT_VAL" "$FPS"; then
                    print_error "ERROR: Compression failed for $filename"
                    success=false
                fi
            fi

            if [[ "$success" == "true" ]] && [[ "$DO_SUBTITLE" == "true" ]]; then
                local subtitle_source
                if [[ "$DO_COMPRESS" == "true" ]]; then
                    subtitle_source="$target_video"
                else
                    subtitle_source="$video"
                fi

                if ! generate_subtitles "$subtitle_source" "$target_sub_dir" "$srt_file" "$LANG_CODE"; then
                    print_error "ERROR: Subtitle generation failed for $filename"
                    success=false
                fi
            fi

            if [[ "$success" == "true" ]]; then
                if [[ "$REMOVE_ORIGINAL" == "true" ]]; then
                    finalize_video "$target_video" "$srt_file" "$dir_name" "$filename"
                fi
                print_success "Completed: $filename"
            fi

            if [[ "$success" == "true" ]]; then
                exit 0
            else
                exit 1
            fi
        ) &

        pids+=($!)
    done

    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            ((processed_count++))
        else
            ((failed_count++))
        fi
    done

    if [[ "$REMOVE_ORIGINAL" == "true" ]] && [[ -d "$OUTPUT_DIR" ]]; then
        find "$OUTPUT_DIR" -name "*.processing" -type f 2>/dev/null | while read -r f; do
            log_warn "Found incomplete file (will be removed): $f"
        done

        if find "$OUTPUT_DIR" -name "*.processing" | grep -q .; then
            print_warning "Some files were not fully processed. Original files preserved."
        else
            rm -rf "$OUTPUT_DIR"
            log_info "Removed temporary directory: $OUTPUT_DIR"
        fi
    fi

    local end_time
    end_time=$(date +%s)
    local elapsed=$((end_time - start_time))

    hr
    print_success "Processing finished in $(format_duration $elapsed)"
    print_info "  Processed: $processed_count | Failed: $failed_count | Skipped: $skipped_count"
    log_info "Session finished: $processed_count processed, $failed_count failed, $skipped_count skipped in $(format_duration $elapsed)"
    hr

    if [[ "$LOG_ENABLED" == "true" ]]; then
        echo ""
        print_info "Log saved to: $LOG_FILE"
    fi
}

main "$@"
