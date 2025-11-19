#!/bin/sh
set -e

# --- Configuration ---
OWNER="major-technology"
REPO="cli"
BINARY="major"
# ---------------------

# Detect OS and Arch
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  OS="Linux" ;;
    Darwin) OS="Darwin" ;;
    *)      echo "OS $OS not supported"; exit 1 ;;
esac

case "$ARCH" in
    x86_64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)      echo "Architecture $ARCH not supported"; exit 1 ;;
esac

# GoReleaser v2 Default Archive Name Template:
# {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.tar.gz
# Example: major_1.0.0_Darwin_arm64.tar.gz

echo "Finding latest release for $OWNER/$REPO..."

# Get latest release tag from GitHub API
LATEST_TAG=$(curl -s "https://api.github.com/repos/$OWNER/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Could not find latest release tag."
    exit 1
fi

# Remove 'v' prefix for version number if your assets use strict numbering (major_1.0.0 vs major_v1.0.0)
# GoReleaser usually strips the 'v' in the version template variable {{ .Version }}
VERSION=${LATEST_TAG#v}

# Construct the asset name
ASSET_NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$OWNER/$REPO/releases/download/$LATEST_TAG/$ASSET_NAME"

echo "Downloading $ASSET_NAME from version $LATEST_TAG..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME" || { echo "Failed to download $DOWNLOAD_URL"; exit 1; }

# Extract and Install
echo "Installing to /usr/local/bin/..."
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"
# Use sudo if we can't write to the destination
if [ -w "/usr/local/bin" ]; then
    mv "$TMP_DIR/$BINARY" "/usr/local/bin/$BINARY"
else
    sudo mv "$TMP_DIR/$BINARY" "/usr/local/bin/$BINARY"
fi

# Cleanup
rm -rf "$TMP_DIR"

echo "Successfully installed $BINARY $LATEST_TAG"