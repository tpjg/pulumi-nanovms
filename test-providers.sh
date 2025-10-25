#!/bin/bash
set -e

# Simple script to test provider support
# Usage: ./test-providers.sh [provider-name]
# Default: azure

PROVIDER=${1:-azure}
TEST_DIR="tests/provider"
STACK_NAME="${PROVIDER}-test"
EXAMPLE_BINARY="examples/application/example/example"

echo "Testing provider: ${PROVIDER}"
echo "Test directory: ${TEST_DIR}"

# Set environment variable for provider
export TEST_PROVIDER="${PROVIDER}"

# Warn about defaults if environment variables are not set
if [ -z "${TEST_BUCKET}" ]; then
    echo "Warning: TEST_BUCKET not set, using default value"
    echo "  You can set it with: export TEST_BUCKET=your-bucket-name"
fi

if [ -z "${TEST_ZONE}" ]; then
    echo "Warning: TEST_ZONE not set, using default value"
    echo "  You can set it with: export TEST_ZONE=your-zone"
fi

# Check if test directory exists
if [ ! -d "${TEST_DIR}" ]; then
    echo "Error: Test directory ${TEST_DIR} does not exist"
    exit 1
fi

# Copy example binary
echo "==> Copying example binary to test directory"
if [ -f "${EXAMPLE_BINARY}" ]; then
    cp "${EXAMPLE_BINARY}" "${TEST_DIR}/example"
    echo "✓ Binary copied"
else
    echo "Warning: Example binary not found at ${EXAMPLE_BINARY}"
    echo "You may need to build it first with:"
    echo "  cd examples/application/example && GOOS=linux GOARCH=amd64 go build"
fi

cd "${TEST_DIR}"

# Install dependencies
echo "==> Installing dependencies with bun"
bun install

# Initialize stack
echo "==> Initializing stack: ${STACK_NAME}"
pulumi stack init ${STACK_NAME} 2>/dev/null || pulumi stack select ${STACK_NAME}

# Deploy
echo "==> Running pulumi up"
pulumi up --yes --skip-preview

# Wait and verify instance is running
echo "==> Verifying instance is running"
MAX_ATTEMPTS=10
ATTEMPT=0
INSTANCE_FOUND=false

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    echo "Attempt $((ATTEMPT + 1))/${MAX_ATTEMPTS}: Checking for running instance..."

    if ops instance list -t ${PROVIDER} 2>/dev/null | grep -q "test-image"; then
        echo "✓ Instance found and running!"
        INSTANCE_FOUND=true
        break
    fi

    ATTEMPT=$((ATTEMPT + 1))
    if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
        sleep 5
    fi
done

if [ "$INSTANCE_FOUND" = false ]; then
    echo "Warning: Instance not found after ${MAX_ATTEMPTS} attempts"
fi

# Tear down
echo "==> Running pulumi down"
pulumi down --yes --skip-preview

# Remove stack
echo "==> Removing stack"
pulumi stack rm ${STACK_NAME} --yes --force

echo "==> Test completed for provider: ${PROVIDER}"
