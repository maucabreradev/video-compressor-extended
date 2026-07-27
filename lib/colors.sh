#!/bin/bash
# ==============================================================================
# Color palette and output helper functions.
# ==============================================================================

if [[ -n "${_COLORS_SH_INCLUDED:-}" ]]; then
    return 0
fi
readonly _COLORS_SH_INCLUDED=1

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

print_header() {
    local title="$1"
    echo -e "${BLUE}=========================================================${NC}"
    echo -e "${BOLD}${title}${NC}"
    echo -e "${BLUE}=========================================================${NC}"
}

print_success() {
    echo -e "${GREEN}${*}${NC}"
}

print_warning() {
    echo -e "${YELLOW}${*}${NC}"
}

print_error() {
    echo -e "${RED}${*}${NC}"
}

print_info() {
    echo -e "${CYAN}${*}${NC}"
}

print_bold() {
    echo -e "${BOLD}${*}${NC}"
}
