#!/bin/sh
set -e

REPO="richardartoul/smooth"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="Darwin" ;;
    linux) OS="Linux" ;;
    mingw*|msys*|cygwin*) OS="Windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Set extension
EXT="tar.gz"
if [ "$OS" = "Windows" ]; then
    EXT="zip"
fi

# Get latest version
echo "Finding latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Error: Could not find latest release."
    echo "Please check https://github.com/${REPO}/releases for available versions."
    exit 1
fi

VERSION="${LATEST#v}"

# Download URL
URL="https://github.com/${REPO}/releases/download/${LATEST}/smooth_${VERSION}_${OS}_${ARCH}.${EXT}"

echo "Downloading smooth ${LATEST} for ${OS}/${ARCH}..."
echo "URL: ${URL}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download and extract
cd "$TMP_DIR"
curl -fsSL "$URL" -o "smooth.${EXT}"

if [ "$EXT" = "tar.gz" ]; then
    tar -xzf "smooth.${EXT}"
else
    unzip -q "smooth.${EXT}"
fi

# Install
echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
    mv smooth "$INSTALL_DIR/"
else
    sudo mv smooth "$INSTALL_DIR/"
fi

echo ""
echo "✓ smooth installed successfully!"
echo ""

# Check if INSTALL_DIR is in PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        echo "Run 'smooth' to get started."
        ;;
    *)
        echo "Add smooth to your PATH by adding this to your shell config:"
        echo ""
        echo "  # For bash (~/.bashrc or ~/.bash_profile):"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        echo "  # For zsh (~/.zshrc):"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        echo "Then restart your terminal or run: source ~/.zshrc"
        ;;
esac

