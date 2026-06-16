#!/bin/bash
set -e

REPO="zenmakek/parcel"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="parcel"
GITHUB_API="https://api.github.com/repos/$REPO/releases/latest"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
MUTED='\033[0;37m'
BOLD='\033[1m'
RESET='\033[0m'

print_banner() {
    echo ""
    echo -e "${CYAN}${BOLD}"
    echo "  ██████╗  █████╗ ██████╗  ██████╗███████╗██╗     "
    echo "  ██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔════╝██║     "
    echo "  ██████╔╝███████║██████╔╝██║     █████╗  ██║     "
    echo "  ██╔═══╝ ██╔══██║██╔══██╗██║     ██╔══╝  ██║     "
    echo "  ██║     ██║  ██║██║  ██║╚██████╗███████╗███████╗"
    echo "  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚══════╝╚══════╝"
    echo -e "${RESET}"
    echo -e "${MUTED}  No accounts. No links. Just a code.${RESET}"
    echo ""
}

info()    { echo -e "  ${CYAN}→${RESET} $1"; }
success() { echo -e "  ${GREEN}✓${RESET} $1"; }
error()   { echo -e "  ${RED}✗ Error:${RESET} $1"; exit 1; }

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)  OS_NAME="linux" ;;
        Darwin) OS_NAME="darwin" ;;
        *)      error "Unsupported OS: $OS. Windows users please download manually from https://github.com/$REPO/releases" ;;
    esac

    case "$ARCH" in
        x86_64)  ARCH_NAME="amd64" ;;
        aarch64) ARCH_NAME="arm64" ;;
        arm64)   ARCH_NAME="arm64" ;;
        *)       error "Unsupported architecture: $ARCH" ;;
    esac

    PLATFORM="${OS_NAME}-${ARCH_NAME}"
    info "Detected platform: $PLATFORM"
}

check_dependencies() {
    for cmd in curl grep sed; do
        if ! command -v "$cmd" &>/dev/null; then
            error "$cmd is required but not installed."
        fi
    done
}

fetch_latest_version() {
    info "Fetching latest release..."
    VERSION=$(curl -fsSL "$GITHUB_API" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        error "Could not fetch latest version. Check your internet connection or visit https://github.com/$REPO/releases"
    fi

    info "Latest version: $VERSION"
}

download_binary() {
    BINARY_FILE="parcel-${PLATFORM}"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_FILE"

    info "Downloading $BINARY_FILE..."

    TMP_FILE="$(mktemp)"

    if ! curl -fsSL --progress-bar "$DOWNLOAD_URL" -o "$TMP_FILE"; then
        rm -f "$TMP_FILE"
        error "Download failed. Check https://github.com/$REPO/releases for available binaries."
    fi

    chmod +x "$TMP_FILE"
    echo "$TMP_FILE"
}

install_binary() {
    TMP_FILE="$1"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        info "Requesting sudo to install to $INSTALL_DIR..."
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi

    success "Installed to $INSTALL_DIR/$BINARY_NAME"
}

verify_install() {
    if command -v "$BINARY_NAME" &>/dev/null; then
        success "Parcel $VERSION installed successfully"
    else
        echo ""
        echo -e "  ${MUTED}Parcel was installed but '$BINARY_NAME' is not in your PATH.${RESET}"
        echo -e "  ${MUTED}Add this to your ~/.bashrc or ~/.zshrc:${RESET}"
        echo ""
        echo -e "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
    fi
}

print_next_steps() {
    echo ""
    echo -e "  ${BOLD}Get started:${RESET}"
    echo ""
    echo -e "  ${CYAN}parcel${RESET}             open Parcel"
    echo ""
    echo -e "  ${MUTED}Files are saved to ~/Downloads by default.${RESET}"
    echo -e "  ${MUTED}OTPs expire after 5 minutes.${RESET}"
    echo ""
    echo -e "  ${MUTED}Docs: https://github.com/$REPO${RESET}"
    echo ""
}

main() {
    print_banner
    check_dependencies
    detect_platform
    fetch_latest_version
    TMP_FILE=$(download_binary)
    install_binary "$TMP_FILE"
    verify_install
    print_next_steps
}

main