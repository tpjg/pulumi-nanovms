#!/bin/sh
set -e

# Get version from environment variable or git tag, default to 0.1.0
VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.1.0")}

echo "Building provider version $VERSION..."

# Build the provider binary with version injection
echo "Building provider binary..."
go build -ldflags "-X main.Version=$VERSION -s -w" -o pulumi-resource-nanovms

# Generate schema from the built provider
echo "Generating schema..."
pulumi package get-schema ./pulumi-resource-nanovms > schema.json

# Verify schema was generated
if [ ! -s schema.json ]; then
    echo "Error: schema.json is empty or was not generated"
    exit 1
fi

# Generate SDKs for all languages
echo "Generating SDKs..."
pulumi package gen-sdk . --version $VERSION --out ../sdk

# Fix nodejs package.json for proper npm publishing
echo "Updating nodejs package.json..."
cd ../sdk/nodejs
jq '. + {
  "main": "bin/index.js",
  "types": "bin/index.d.ts",
  "files": ["bin/", "README.md", "package.json"]
}' package.json > package.json.tmp && mv package.json.tmp package.json
cd ../../provider

# Initialize Go SDK module
echo "Initializing Go SDK module..."
cd ../sdk/go/pulumi-nanovms
if [ ! -f go.mod ]; then
    go mod init github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms
fi
go mod tidy

# Install nodejs dependencies
cd ../../nodejs
bun install
tsc
cp package.json bin/
bun link

echo "SDK generation complete!"
