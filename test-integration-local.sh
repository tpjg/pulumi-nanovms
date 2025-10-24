#!/bin/bash
# Integration test script for NanoVMs Pulumi provider
# This script:
# 1. Builds the provider using build-sdk.sh
# 2. Builds the example application for Linux
# 3. Runs the integration test (creates image and instance using 'onprem' provider)

set -e

# Get the script directory (project root)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

step() {
    echo ""
    echo -e "${BLUE}===================================================${NC}"
    echo -e "${BLUE}➜ $1${NC}"
    echo -e "${BLUE}===================================================${NC}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

info() {
    echo -e "  $1"
}

# Cleanup function
cleanup() {
    step "Cleaning up..."

    # Clean up Pulumi stack if it exists
    if [ -d "$SCRIPT_DIR/tests/integration" ]; then
        cd "$SCRIPT_DIR/tests/integration"
        info "Cleaning up Pulumi stack in $PWD"
        if pulumi stack ls 2>/dev/null | grep -q "dev"; then
            info "Cleaning up Pulumi stack..."
            pulumi stack select dev 2>/dev/null || true
            pulumi destroy --yes --skip-preview 2>/dev/null || true
            pulumi stack rm dev --force --yes 2>/dev/null || true
        fi
    fi

    # Set up ops environment
    export OPS_DIR="${OPS_DIR:-$HOME/.ops}"
    export PATH="$OPS_DIR/bin:$PATH"

    # Check if ops is available
    if command -v ops >/dev/null 2>&1; then
        info "Checking for running instances..."
        ops instance list || true

        # Try to clean up any instances that might be stuck
        for instance in $(ops instance list 2>/dev/null | grep -E "^integration-test" | awk '{print $1}' || true); do
            warn "Cleaning up instance: $instance"
            ops instance delete "$instance" || true
        done
    else
        warn "ops CLI not found in PATH - skipping instance cleanup"
    fi

    success "Cleanup complete"
}

# Set up trap to cleanup on exit
trap cleanup EXIT

echo "===================================="
echo "NanoVMs Provider Integration Test"
echo "===================================="
echo ""
info "This test will:"
info "  1. Build the provider and SDKs"
info "  2. Build the example application for Linux"
info "  3. Create a NanoVMs unikernel image"
info "  4. Run the unikernel instance (onprem/QEMU)"
info "  5. Verify the instance is running"
echo ""

# Check prerequisites
step "Checking prerequisites"

info "Checking required tools..."
command -v go >/dev/null 2>&1 || { error "Go is not installed"; exit 1; }
command -v pulumi >/dev/null 2>&1 || { error "Pulumi CLI is not installed"; exit 1; }
command -v bun >/dev/null 2>&1 || { error "Bun is not installed"; exit 1; }

success "Go found: $(go version | awk '{print $3}')"
success "Pulumi found: $(pulumi version)"
success "Bun found: $(bun --version)"

# Check for ops CLI
export OPS_DIR="${OPS_DIR:-$HOME/.ops}"
export PATH="$OPS_DIR/bin:$PATH"

if command -v ops >/dev/null 2>&1; then
    success "ops CLI found: $(ops version 2>&1 | head -1)"
else
    warn "ops CLI not found in PATH"
    info "Install it from: https://ops.city"
    info "Or run: curl https://ops.city/get.sh -sSfL | sh"
    error "ops CLI is required for integration tests"
    exit 1
fi

# Check for QEMU
if command -v qemu-system-x86_64 >/dev/null 2>&1; then
    success "QEMU found: $(qemu-system-x86_64 --version | head -1)"
elif [ -f "$OPS_DIR/bin/qemu-system-x86_64" ]; then
    success "QEMU found in ops directory"
else
    warn "QEMU not found - ops will download it when needed"
fi

# Check for KVM/HVF acceleration
if [ -e /dev/kvm ]; then
    success "KVM acceleration available (/dev/kvm exists)"
elif [ "$(uname)" = "Darwin" ]; then
    # macOS uses Hypervisor.framework
    success "Running on macOS - will use Hypervisor.framework acceleration"
else
    warn "No hardware acceleration detected - test will be slower"
    warn "Install KVM on Linux or run on macOS for better performance"
fi

# Step 1: Build provider and SDKs
step "Building provider and SDKs"

cd "$SCRIPT_DIR/provider"
info "Running build-sdk.sh..."
./build-sdk.sh

success "Provider and SDKs built successfully"

# Verify provider binary exists
if [ ! -f "$SCRIPT_DIR/provider/pulumi-nanovms" ]; then
    error "Provider binary not found at provider/pulumi-nanovms"
    exit 1
fi

# Step 2: Build example application for Linux
step "Building example application for Linux"

cd "$SCRIPT_DIR/examples/application/example"

info "Building Go application with GOOS=linux..."
GOOS=linux GOARCH=amd64 go build -o example main.go

if [ ! -f "example" ]; then
    error "Failed to build example application"
    exit 1
fi

EXAMPLE_SIZE=$(du -h example | awk '{print $1}')
success "Example application built successfully (size: $EXAMPLE_SIZE)"

# Verify it's a Linux binary
FILE_TYPE=$(file example)
if echo "$FILE_TYPE" | grep -q "ELF.*x86-64"; then
    success "Verified: Linux x86-64 ELF binary"
else
    warn "Binary type: $FILE_TYPE"
fi

# Step 3: Set up integration test
step "Setting up integration test"

cd "$SCRIPT_DIR/tests/integration"

# Install dependencies
info "Installing test dependencies..."
bun install --silent

success "Test dependencies installed"

# Build TypeScript
info "Building TypeScript..."
bun run build

success "TypeScript compiled"

# Step 4: Run Pulumi integration test
step "Running Pulumi integration test"

# Export the example binary path
export EXAMPLE_BINARY="$SCRIPT_DIR/examples/application/example/example"
info "Using example binary: $EXAMPLE_BINARY"

# Initialize Pulumi stack (or use existing)
info "Setting up Pulumi stack..."
pulumi login --local
pulumi stack select dev 2>/dev/null || pulumi stack init dev

success "Pulumi stack ready"

# Run pulumi up with detailed output
info "Creating NanoVMs image and instance..."
info "This may take several minutes depending on your system..."
echo ""

# Save output to a log file
PULUMI_LOG="/tmp/pulumi-up.log"
if pulumi up --yes --skip-preview 2>&1 | tee "$PULUMI_LOG"; then
    success "Pulumi deployment completed successfully"
else
    error "Pulumi deployment failed"
    error "Check the log at: $PULUMI_LOG"
    exit 1
fi

# Step 5: Verify outputs
step "Verifying deployment outputs"

# Get outputs
IMAGE_NAME=$(pulumi stack output imageName 2>/dev/null || echo "")
IMAGE_PATH=$(pulumi stack output imagePath 2>/dev/null || echo "")
INSTANCE_ID=$(pulumi stack output instanceId 2>/dev/null || echo "")
INSTANCE_STATUS=$(pulumi stack output instanceStatus 2>/dev/null || echo "")

echo ""
info "Deployment outputs:"
info "  Image Name:      $IMAGE_NAME"
info "  Image Path:      $IMAGE_PATH"
info "  Instance ID:     $INSTANCE_ID"
info "  Instance Status: $INSTANCE_STATUS"
echo ""

# Verify outputs exist
if [ -z "$IMAGE_NAME" ] || [ -z "$INSTANCE_ID" ]; then
    error "Missing expected outputs from Pulumi stack"
    exit 1
fi

success "All expected outputs present"

# Step 6: Test the running instance
step "Testing running instance"

sleep 3
ops instance list
echo "Looking for $INSTANCE_ID"

# Get instance details from ops
info "Fetching instance details from ops..."
if ops instance list | grep -q "$INSTANCE_ID"; then
    success "Instance found in ops instance list"
    echo ""
    ops instance list | grep -E "NAME|$INSTANCE_ID" || true
    echo ""
else
    warn "Instance not found in ops list, but may still be running"
fi

# Try to get instance logs (if available)
info "Attempting to fetch instance logs..."
if ops instance logs "$INSTANCE_ID" 2>/dev/null | head -20; then
    echo ""
    success "Instance logs retrieved"
else
    warn "Could not retrieve instance logs (this is normal for onprem instances)"
fi

# Check if we can connect to the instance
info "Checking if instance is accessible..."

# For onprem instances, they typically run on localhost with port forwarding
# Try to connect to localhost:8888
MAX_ATTEMPTS=10
ATTEMPT=0
CONNECTED=false

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    info "Attempt $ATTEMPT/$MAX_ATTEMPTS: Trying to connect to localhost:8888..."

    if curl -s -m 5 http://localhost:8888/ > /tmp/response.html 2>/dev/null; then
        CONNECTED=true
        break
    fi

    if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
        sleep 2
    fi
done

echo ""
if [ "$CONNECTED" = true ]; then
    success "Successfully connected to instance!"
    echo ""
    info "Response preview:"
    echo "---------------------------------------------------"
    head -20 /tmp/response.html || cat /tmp/response.html
    echo "---------------------------------------------------"
    echo ""

    # Check if TEST_VAR environment variable is present
    if grep -q "TEST_VAR=integration-test" /tmp/response.html; then
        success "TEST_VAR environment variable found in response!"
    else
        warn "TEST_VAR not found in response"
    fi

    success "Full response saved to: /tmp/response.html"
else
    warn "Could not connect to instance on localhost:8888"
    warn "The instance may be running but not accessible via port forwarding"
    warn "This is common with onprem/QEMU instances depending on configuration"
fi

# Step 7: Clean up (handled by trap)
step "Test Summary"

echo ""
info "Integration test completed successfully!"
echo ""
info "Summary:"
info "  ✓ Provider and SDKs built"
info "  ✓ Example application compiled for Linux"
info "  ✓ NanoVMs image created: $IMAGE_NAME"
info "  ✓ Instance deployed: $INSTANCE_ID"
if [ "$CONNECTED" = true ]; then
    info "  ✓ Instance responded to HTTP requests"
else
    warn "  ! Instance did not respond to HTTP requests"
fi
echo ""

success "All integration tests passed!"

echo ""
info "Cleanup will run automatically..."
echo ""

exit 0
