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

# Setup directories
INSTALL_DIR="$HOME/.parascope"
BINARY_DIR="$HOME/.parascope/bin"

print_info "Creating user directory $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$BINARY_DIR"

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

print_info "Installing version $VERSION"

# Download binary
curl -sL "$ASSET_URL" -o "$BINARY_DIR/para"

if [ ! -f "$BINARY_DIR/para" ]; then
    print_error "Failed to download binary"
    exit 1
fi

chmod +x "$BINARY_DIR/para"


# Determine shell profile and update PATH
PROFILE=""
case "$(basename "$SHELL")" in
    zsh) PROFILE="$HOME/.zshrc" ;;
    fish) PROFILE="$HOME/.config/fish/config.fish" ;;
    *) PROFILE="$HOME/.bashrc" ;;
esac

# Check if PATH already contains the binary directory
if ! grep -Eq 'PATH.*parascope' "$PROFILE"; then
    if [ "$SHELL" = "/usr/bin/fish" ] || [ "$SHELL" = "/bin/fish" ]; then
        # For fish shell, add the appropriate PATH export
        echo "set -gx PATH \"$BINARY_DIR\" \$PATH" >> "$PROFILE"
    else
        # For bash/zsh, add the appropriate PATH export
        echo "export PATH=\"$BINARY_DIR:\$PATH\"" >> "$PROFILE"
    fi
fi

# Success message
echo ""
print_info "Parascan successfully installed!"
echo ""
echo "Try: para help"
echo ""
print_warning "If 'para help' doesn't work, run:"
echo "source $PROFILE"
echo ""