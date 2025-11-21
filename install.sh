#!/bin/bash
set -e

# --- Configuration ---
BINARY="major"
INSTALL_DIR="$HOME/.major/bin"
S3_BUCKET_URL="https://major-cli-releases.s3.us-west-1.amazonaws.com"
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

# Get latest version from S3
LATEST_VERSION=$(curl -fsSL "$S3_BUCKET_URL/latest-version")

if [ -z "$LATEST_VERSION" ]; then
    print_error "Could not find latest release version"
    exit 1
fi

VERSION="$LATEST_VERSION"

# Construct the asset name
ASSET_NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS_NAME="${BINARY}_${VERSION}_checksums.txt"
DOWNLOAD_URL="$S3_BUCKET_URL/$VERSION/$ASSET_NAME"
CHECKSUMS_URL="$S3_BUCKET_URL/$VERSION/$CHECKSUMS_NAME"

print_step "Downloading ${BINARY} v${VERSION}..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)

# Download Asset
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"; then
    print_error "Failed to download binary from $DOWNLOAD_URL"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Download Checksums
if ! curl -fsSL "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt"; then
    print_error "Failed to download checksums from $CHECKSUMS_URL"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Verify Checksum
print_step "Verifying checksum..."
cd "$TMP_DIR"

# Extract the checksum for our specific asset
EXPECTED_CHECKSUM=$(grep "$ASSET_NAME" checksums.txt | awk '{print $1}')

if [ -z "$EXPECTED_CHECKSUM" ]; then
    print_error "Could not find checksum for $ASSET_NAME in checksums.txt"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Calculate actual checksum
if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM=$(sha256sum "$ASSET_NAME" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM=$(shasum -a 256 "$ASSET_NAME" | awk '{print $1}')
else
    print_error "Neither sha256sum nor shasum found to verify checksum"
    rm -rf "$TMP_DIR"
    exit 1
fi

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    print_error "Checksum verification failed!"
    printf "  Expected: %s\n" "$EXPECTED_CHECKSUM"
    printf "  Actual:   %s\n" "$ACTUAL_CHECKSUM"
    rm -rf "$TMP_DIR"
    exit 1
fi

print_success "Checksum verified"

# Extract and Install
print_step "Installing to $INSTALL_DIR..."
tar -xzf "$ASSET_NAME"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Move binary to install directory
mv "$BINARY" "$INSTALL_DIR/$BINARY"

# Make sure it's executable
chmod +x "$INSTALL_DIR/$BINARY"

# Cleanup
cd - >/dev/null
rm -rf "$TMP_DIR"

# Run the internal install command to setup shell integration
print_step "Setting up shell integration..."
"$INSTALL_DIR/$BINARY" install

# Verify installation
print_step "Verifying installation..."

# We verify using the absolute path since PATH might not be updated in the current shell yet
INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY" --version 2>&1 | head -n 1 || echo "unknown")
print_success "Successfully installed ${BINARY} v${VERSION}"

# Print welcome message
printf "\n${BOLD}${GREEN}🎉 Welcome to Major!${RESET}\n\n"
printf "Get started with these commands:\n\n"
printf "  ${BOLD}major user login${RESET}      Log in to your Major account\n"
printf "  ${BOLD}major app create${RESET}      Create a new application\n"
printf "  ${BOLD}major --help${RESET}          View all available commands\n"
