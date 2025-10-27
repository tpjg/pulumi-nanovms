#!/bin/bash
# install_plugin.sh - Install the Pulumi NanoVMs provider plugin
#
# This script downloads and installs the provider plugin binary from GitHub releases.
# The plugin is required for all languages (TypeScript, Python, Go, .NET).
#
# Usage:
#   ./install_plugin.sh [VERSION]
#
# If VERSION is not specified, the latest version (0.1.2) is used.

set -e

# Default to latest version
VERSION="${1:-0.1.2}"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    arm64)
        ARCH="arm64"
        ;;
    amd64)
        ARCH="amd64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        echo "Supported architectures: x86_64 (amd64), arm64 (aarch64)"
        exit 1
        ;;
esac

# Validate OS
case "$OS" in
    linux|darwin)
        ;;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        echo "Supported systems: Linux, macOS (darwin)"
        exit 1
        ;;
esac

# Check if pulumi CLI is installed
if ! command -v pulumi &> /dev/null; then
    echo -e "${RED}Error: Pulumi CLI is not installed${NC}"
    echo "Please install Pulumi first: https://www.pulumi.com/docs/install/"
    exit 1
fi

# Build download URL
TARBALL="pulumi-nanovms-v${VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/tpjg/pulumi-nanovms/releases/download/v${VERSION}/${TARBALL}"

echo -e "${GREEN}Installing Pulumi NanoVMs Provider Plugin${NC}"
echo "  Version: $VERSION"
echo "  Platform: $OS-$ARCH"
echo "  URL: $URL"
echo ""

# Download tarball
echo -e "${YELLOW}📦 Downloading provider plugin...${NC}"
if ! curl -fLO "$URL"; then
    echo -e "${RED}Error: Failed to download plugin from GitHub releases${NC}"
    echo "URL: $URL"
    echo ""
    echo "Please check:"
    echo "  1. Version $VERSION exists in GitHub releases"
    echo "  2. You have internet connectivity"
    echo "  3. The binary for your platform ($OS-$ARCH) is available"
    exit 1
fi

# Verify download
if [ ! -f "$TARBALL" ]; then
    echo -e "${RED}Error: Downloaded file not found: $TARBALL${NC}"
    exit 1
fi

# Install plugin using Pulumi CLI
echo -e "${YELLOW}⚙️  Installing plugin...${NC}"
if ! pulumi plugin install resource nanovms "$VERSION" -f "$TARBALL"; then
    echo -e "${RED}Error: Failed to install plugin${NC}"
    rm -f "$TARBALL"
    exit 1
fi

# Cleanup
rm -f "$TARBALL"

# Verify installation
echo ""
echo -e "${GREEN}✅ Installation complete!${NC}"
echo ""
echo "Installed plugin:"
pulumi plugin ls | grep nanovms || echo -e "${YELLOW}Warning: Plugin not showing in list${NC}"

echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "  1. Install the SDK for your language:"
echo "     • TypeScript/JavaScript: bun install @tpjg/pulumi-nanovms"
echo "     • Python: pip install tpjg-pulumi-nanovms"
echo "     • Go: go get github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms"
echo "     • .NET: dotnet add package Tpjg.PulumiNanovms"
echo ""
echo "  2. Use in your Pulumi program:"
echo "     See examples in ./examples/ or docs at ./docs/_index.md"
