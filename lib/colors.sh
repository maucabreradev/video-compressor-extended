#!/bin/bash
# ==============================================================================
# Color palette and output helper functions.
# ==============================================================================

if [[ -n "${_COLORS_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _COLORS_SH_INCLUDED=1

readonly RED=$'\033[0;31m'
readonly GREEN=$'\033[0;32m'
readonly YELLOW=$'\033[1;33m'
readonly BLUE=$'\033[0;34m'
readonly CYAN=$'\033[0;36m'
readonly BOLD=$'\033[1m'
readonly NC=$'\033[0m'

print_header() {
    local title="$1"
    echo "${BLUE}=========================================================${NC}"
    echo "${BOLD}${title}${NC}"
    echo "${BLUE}=========================================================${NC}"
}

print_success() {
    echo "${GREEN}${*}${NC}"
}

print_warning() {
    echo "${YELLOW}${*}${NC}"
}

print_error() {
    echo "${RED}${*}${NC}"
}

print_info() {
    echo "${CYAN}${*}${NC}"
}

print_bold() {
    echo "${BOLD}${*}${NC}"
}
