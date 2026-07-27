#!/bin/bash
# ==============================================================================
# Configuration parsing: CLI arguments, config file, defaults.
# ==============================================================================

if [[ -n "${_CONFIG_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _CONFIG_SH_INCLUDED=1

LANG_CODE="en"
RESOLUTION="1280x720"
FPS="24"
DO_COMPRESS=true
DO_SUBTITLE=true
REMOVE_ORIGINAL=false
VERBOSE=false
QUIET=false
DRY_RUN=false
WIDTH_VAL=""
HEIGHT_VAL=""

VCX_VERSION="1.0.0"

load_config_file() {
    local config_files=(
        "${VCX_CONFIG_FILE:-}"
        "${HOME}/.vcxrc"
        "${HOME}/.config/vcx/config"
    )

    for cfg in "${config_files[@]}"; do
        if [[ -n "$cfg" ]] && [[ -f "$cfg" ]]; then
            source "$cfg" 2>/dev/null
            log_debug "Loaded config from: $cfg"
        fi
    done
}

show_help() {
    cat << EOF
${BOLD}Video Compressor Extended (VCX)${NC} v${VCX_VERSION}

${CYAN}Usage:${NC} vcx.sh [OPTIONS]

${CYAN}Compression:${NC}
  --compress WIDTHxHEIGHT   Set target resolution (default: 1280x720)
  --no-compress             Skip compression, only generate subtitles
  --fps NUMBER              Set target frame rate (default: 24)

${CYAN}Subtitles:${NC}
  --no-sub                  Skip subtitle generation, only compress
  --en, --es, --fr, ...     Set subtitle language (default: en)

${CYAN}Output:${NC}
  --remove-original         Replace original files with processed versions

${CYAN}Options:${NC}
  --dry-run                 Show what would be done without executing
  --verbose                 Enable verbose output
  --quiet                   Suppress non-error output
  --config FILE             Use specified config file instead of defaults
  --log-file FILE           Write log to specified file
  --no-log                  Disable logging
  --version                 Show version and exit
  --help                    Show this help message

${CYAN}Examples:${NC}
  vcx.sh                                    # Compress and subtitle all videos
  vcx.sh --compress 1920x1080 --fps 30     # Custom resolution and FPS
  vcx.sh --no-compress --es                # Only generate Spanish subtitles
  vcx.sh --remove-original --dry-run        # Preview replacement plan
EOF
    exit 0
}

show_version() {
    echo "VCX v${VCX_VERSION}"
    exit 0
}

parse_arguments() {
    while [[ "$#" -gt 0 ]]; do
        case "$1" in
            --compress)
                if [[ -z "$2" ]] || [[ "$2" == -* ]]; then
                    print_error "ERROR: --compress requires a WIDTHxHEIGHT value."
                    exit 1
                fi
                RESOLUTION="$2"
                DO_COMPRESS=true
                shift 2
                ;;
            --no-compress)
                DO_COMPRESS=false
                shift
                ;;
            --fps)
                if [[ -z "$2" ]] || [[ "$2" == -* ]]; then
                    print_error "ERROR: --fps requires a numeric value."
                    exit 1
                fi
                FPS="$2"
                shift 2
                ;;
            --no-sub)
                DO_SUBTITLE=false
                shift
                ;;
            --remove-original)
                REMOVE_ORIGINAL=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --quiet)
                QUIET=true
                shift
                ;;
            --config)
                if [[ -z "$2" ]] || [[ "$2" == -* ]]; then
                    print_error "ERROR: --config requires a file path."
                    exit 1
                fi
                VCX_CONFIG_FILE="$2"
                shift 2
                ;;
            --log-file)
                if [[ -z "$2" ]] || [[ "$2" == -* ]]; then
                    print_error "ERROR: --log-file requires a file path."
                    exit 1
                fi
                LOG_FILE="$2"
                shift 2
                ;;
            --no-log)
                LOG_ENABLED=false
                shift
                ;;
            --version)
                show_version
                ;;
            --help)
                show_help
                ;;
            --es|--en|--fr|--de|--pt|--it|--ja|--ko|--zh|--ru|--ar)
                LANG_CODE="${1#--}"
                shift
                ;;
            -*)
                print_error "Unknown option: $1"
                echo "Use --help for usage information."
                exit 1
                ;;
            *)
                shift
                ;;
        esac
    done
}

show_config_summary() {
    print_header "VCX Configuration"
    echo "  Compress Video  : [ $(print_info "$DO_COMPRESS") ]"
    if [[ "$DO_COMPRESS" == "true" ]]; then
        echo "    Resolution    : $RESOLUTION"
        echo "    Frame Rate    : $FPS fps"
    fi
    echo "  Generate Subs   : [ $(print_info "$DO_SUBTITLE") ]"
    if [[ "$DO_SUBTITLE" == "true" ]]; then
        echo "    Language      : $LANG_CODE"
    fi
    echo "  Replace Originals: [ $(print_info "$REMOVE_ORIGINAL") ]"
    if [[ "$DRY_RUN" == "true" ]]; then
        print_warning "  DRY-RUN mode: no files will be modified"
    fi
    hr
}
