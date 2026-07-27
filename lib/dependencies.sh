#!/bin/bash
# ==============================================================================
# Dependency checking with OS detection.
# Verifies required tools are installed and offers distribution-specific
# installation instructions when they are missing.
# ==============================================================================

if [[ -n "${_DEPENDENCIES_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _DEPENDENCIES_SH_INCLUDED=1

DETECTED_OS=""
DETECTED_OS_FAMILY=""

detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        DETECTED_OS="linux"

        if [[ -f /etc/os-release ]]; then
            local os_id os_name
            os_id="$(grep -oP '(?<=^ID=).*' /etc/os-release | tr -d '"')"
            os_name="$(grep -oP '(?<=^NAME=).*' /etc/os-release | tr -d '"')"

            case "${os_id,,}" in
                ubuntu|debian|linuxmint|pop|elementary|zorin)
                    DETECTED_OS_FAMILY="debian"
                    DETECTED_OS="${os_name:-$os_id}"
                    ;;
                fedora|rhel|centos|rocky|almalinux)
                    DETECTED_OS_FAMILY="rhel"
                    DETECTED_OS="${os_name:-$os_id}"
                    ;;
                arch|endeavouros|manjaro)
                    DETECTED_OS_FAMILY="arch"
                    DETECTED_OS="${os_name:-$os_id}"
                    ;;
                opensuse*|sles)
                    DETECTED_OS_FAMILY="suse"
                    DETECTED_OS="${os_name:-$os_id}"
                    ;;
                *)
                    DETECTED_OS_FAMILY="unknown"
                    DETECTED_OS="${os_name:-$os_id}"
                    ;;
            esac
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        DETECTED_OS="macOS"
        DETECTED_OS_FAMILY="macos"
    else
        DETECTED_OS="$OSTYPE"
        DETECTED_OS_FAMILY="unknown"
    fi

    log_debug "Detected OS: $DETECTED_OS (family: $DETECTED_OS_FAMILY)"
}

check_command() {
    local cmd="$1"
    command -v "$cmd" &>/dev/null
}

check_ffmpeg_vaapi() {
    if ffmpeg -hide_banner -encoders 2>/dev/null | grep -q hevc_vaapi; then
        return 0
    fi
    return 1
}

show_install_instructions() {
    local missing=("$@")

    if [[ ${#missing[@]} -eq 0 ]]; then
        return
    fi

    print_error "The following required tools are missing:"
    for tool in "${missing[@]}"; do
        echo "  - $tool"
    done

    echo ""
    print_info "Installation instructions for $DETECTED_OS:"

    case "$DETECTED_OS_FAMILY" in
        debian)
            for tool in "${missing[@]}"; do
                case "$tool" in
                    ffmpeg)
                        echo "  sudo apt update && sudo apt install ffmpeg"
                        ;;
                    whisper-ctranslate2|whisper)
                        echo "  pip install whisper-ctranslate2"
                        ;;
                esac
            done
            ;;
        rhel)
            for tool in "${missing[@]}"; do
                case "$tool" in
                    ffmpeg)
                        echo "  sudo dnf install rpmfusion-free-release"
                        echo "  sudo dnf install ffmpeg"
                        ;;
                    whisper-ctranslate2|whisper)
                        echo "  pip install whisper-ctranslate2"
                        ;;
                esac
            done
            ;;
        arch)
            for tool in "${missing[@]}"; do
                case "$tool" in
                    ffmpeg)
                        echo "  sudo pacman -S ffmpeg"
                        ;;
                    whisper-ctranslate2|whisper)
                        echo "  yay -S whisper-ctranslate2"
                        echo "  # or: pip install whisper-ctranslate2"
                        ;;
                esac
            done
            ;;
        macos)
            for tool in "${missing[@]}"; do
                case "$tool" in
                    ffmpeg)
                        echo "  brew install ffmpeg"
                        ;;
                    whisper-ctranslate2|whisper)
                        echo "  pip install whisper-ctranslate2"
                        ;;
                esac
            done
            ;;
        *)
            print_warning "Automatic install instructions not available for $DETECTED_OS."
            echo "  Please install the missing tools manually."
            ;;
    esac

    echo ""
}

check_dependencies() {
    detect_os
    log_debug "Checking dependencies on $DETECTED_OS"

    local missing=()

    if ! check_command "ffmpeg"; then
        missing+=("ffmpeg")
    fi

    if [[ "$DO_COMPRESS" == "true" ]]; then
        if check_command "ffmpeg"; then
            if ! check_ffmpeg_vaapi; then
                print_warning "ffmpeg found but hevc_vaapi encoder not available."
                print_warning "Hardware acceleration will not work. Video compression may fail."
            fi
        fi
    fi

    if [[ "$DO_SUBTITLE" == "true" ]]; then
        if ! check_command "whisper-ctranslate2"; then
            missing+=("whisper-ctranslate2")
        fi
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        show_install_instructions "${missing[@]}"
        print_error "Please install the missing tools and try again."
        exit 1
    fi

    log_info "All required dependencies are installed"
}
