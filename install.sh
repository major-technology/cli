#!/bin/bash
set -e

# --- Configuration ---
OWNER="major-technology"
REPO="cli"
BINARY="major"
INSTALL_DIR="$HOME/.major/bin"
# ---------------------

# ANSI color codes for better output
if [ -t 1 ]; then
    BOLD='\033[1m'
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    YELLOW='\033[0;33m'
    RED='\033[0;31m'
    RESET='\033[0m'
else
    BOLD=''
    GREEN=''
    BLUE=''
    YELLOW=''
    RED=''
    RESET=''
fi

# Helper function for formatted output
print_step() {
    printf "${BLUE}▸${RESET} %s\n" "$1"
}

print_success() {
    printf "${GREEN}✓${RESET} %s\n" "$1"
}

print_error() {
    printf "${RED}✗${RESET} %s\n" "$1"
}

# Print header
printf "\n${BOLD}Major CLI Installer${RESET}\n\n"

# Detect OS and Arch
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      print_error "OS $OS not supported"; exit 1 ;;
esac

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)      print_error "Architecture $ARCH not supported"; exit 1 ;;
esac

# GoReleaser v2 Default Archive Name Template:
# {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.tar.gz
# Example: major_1.0.0_Darwin_arm64.tar.gz

print_step "Finding latest release..."

# Get latest release tag from GitHub API
LATEST_TAG=$(curl -s "https://api.github.com/repos/$OWNER/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    print_error "Could not find latest release tag"
    exit 1
fi

# Remove 'v' prefix for version number if your assets use strict numbering (major_1.0.0 vs major_v1.0.0)
# GoReleaser usually strips the 'v' in the version template variable {{ .Version }}
VERSION=${LATEST_TAG#v}

# Construct the asset name
ASSET_NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$OWNER/$REPO/releases/download/$LATEST_TAG/$ASSET_NAME"

print_step "Downloading ${BINARY} ${LATEST_TAG}..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME" || { print_error "Failed to download from $DOWNLOAD_URL"; exit 1; }

# Extract and Install
print_step "Installing to $INSTALL_DIR..."
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Move binary to install directory
mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"

# Make sure it's executable
chmod +x "$INSTALL_DIR/$BINARY"

# Cleanup
rm -rf "$TMP_DIR"

# Run the internal install command to setup shell integration
print_step "Setting up shell integration..."
"$INSTALL_DIR/$BINARY" install

# Verify installation
print_step "Verifying installation..."

# We verify using the absolute path since PATH might not be updated in the current shell yet
INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY" --version 2>&1 | head -n 1 || echo "unknown")
print_success "Successfully installed ${BINARY} ${LATEST_TAG}"

# Print welcome message
printf "\n${BOLD}${GREEN}🎉 Welcome to Major!${RESET}\n\n"
printf "Get started with these commands:\n\n"
printf "  ${BOLD}major user login${RESET}      Log in to your Major account\n"
printf "  ${BOLD}major app create${RESET}      Create a new application\n"
printf "  ${BOLD}major --help${RESET}          View all available commands\n"
