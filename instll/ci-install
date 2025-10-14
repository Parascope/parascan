#!/bin/sh

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo "${GREEN}$1${NC}"
}

print_error() {
    echo "${RED}$1${NC}"
}

print_warning() {
    echo "${YELLOW}$1${NC}"
}

# Detect platform and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux) BIN_NAME="para-linux-$ARCH" ;;
    darwin) BIN_NAME="para-darwin-$ARCH" ;;
    *)
        print_error "Unsupported OS: $OS"
        exit 1
        ;;
esac

# GitHub repository and API
REPO="Parascope/parascan"
API_URL="https://api.github.com/repos/$REPO/releases/latest"

# Get latest release assets
RELEASE_INFO=$(curl -s "$API_URL")
if [ $? -ne 0 ]; then
    print_error "Failed to fetch release information from GitHub"
    exit 1
fi

# Extract version and binary download URL
VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | head -n1 | cut -d '"' -f 4)
ASSET_URL=$(echo "$RELEASE_INFO" | grep 'browser_download_url' | grep "$BIN_NAME" | head -n1 | cut -d '"' -f 4)

if [ -z "$ASSET_URL" ]; then
    print_error "Could not find binary $BIN_NAME in the latest release"
    exit 1
fi

echo "Downloading parascan $VERSION to current directory..."

# Download binary to current directory
curl -sL "$ASSET_URL" -o "./para"

if [ ! -f "./para" ]; then
    print_error "Failed to download binary"
    exit 1
fi

chmod +x "./para"

# Success message
echo ""
print_info "Parascan successfully installed!"
echo ""
echo "Usage:"
echo "./para scan"
echo ""