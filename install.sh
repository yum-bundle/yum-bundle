#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="yum-bundle/yum-bundle"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_RELEASES="https://github.com/${REPO}/releases"

# Error handling
error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

info() {
    echo -e "${GREEN}Info:${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

# Check if running on an RPM-based system
check_system() {
    if [ ! -f /etc/redhat-release ] && [ ! -f /etc/fedora-release ] && \
       [ ! -f /etc/rocky-release ] && [ ! -f /etc/almalinux-release ] && \
       ! grep -qi "rhel\|centos\|fedora\|rocky\|almalinux\|oracle" /etc/os-release 2>/dev/null; then
        error "This installer is for RPM-based systems only (RHEL, CentOS, Fedora, Rocky Linux, AlmaLinux)"
    fi

    if ! command -v rpm >/dev/null 2>&1; then
        error "rpm is required but not installed"
    fi
}

# Detect system architecture
detect_arch() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64)
            echo "x86_64"
            ;;
        aarch64)
            echo "aarch64"
            ;;
        armv7l|armv7hl)
            echo "armv7hl"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "This script must be run as root (use sudo)"
    fi
}

# Fetch latest release info
get_latest_release() {
    local token="${GITHUB_TOKEN:-}"
    local auth_header=""

    if [ -n "$token" ]; then
        auth_header="-H \"Authorization: token ${token}\""
    fi

    local latest_tag
    latest_tag=$(curl -s ${auth_header} "${GITHUB_API}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/' || true)

    if [ -z "$latest_tag" ]; then
        error "Failed to fetch latest release. Check your internet connection and GitHub access."
    fi

    echo "$latest_tag"
}

# Download and install .rpm package
install_package() {
    local tag=$1
    local arch=$2
    local package_name="yum-bundle_${tag#v}_linux_${arch}.rpm"
    local download_url="${GITHUB_RELEASES}/download/${tag}/${package_name}"
    local temp_file
    temp_file=$(mktemp --suffix=.rpm)

    info "Downloading ${package_name}..."

    if ! curl -fsSL -o "$temp_file" "$download_url"; then
        rm -f "$temp_file"
        error "Failed to download package. URL: ${download_url}"
    fi

    info "Installing package..."
    if command -v dnf >/dev/null 2>&1; then
        dnf install -y "$temp_file" || error "Failed to install package with dnf"
    elif command -v yum >/dev/null 2>&1; then
        yum install -y "$temp_file" || error "Failed to install package with yum"
    else
        rpm -i "$temp_file" || error "Failed to install package with rpm"
    fi

    rm -f "$temp_file"
    info "yum-bundle installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v yum-bundle >/dev/null 2>&1; then
        local ver
        ver=$(yum-bundle --version 2>/dev/null || echo "unknown")
        info "Installation verified. Version: ${ver}"
    else
        warn "yum-bundle command not found in PATH. You may need to log out and back in."
    fi
}

# Main execution
main() {
    info "yum-bundle installer"
    info "Repository: ${REPO}"

    check_system
    check_root

    local arch
    arch=$(detect_arch)
    info "Detected architecture: ${arch}"

    info "Fetching latest release..."
    local latest_tag
    latest_tag=$(get_latest_release)
    info "Latest release: ${latest_tag}"

    install_package "$latest_tag" "$arch"
    verify_installation

    echo ""
    info "Installation complete! Run 'yum-bundle --help' to get started."
}

# Run main function
main "$@"
