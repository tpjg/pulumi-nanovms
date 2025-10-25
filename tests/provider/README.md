# Provider Testing

This test validates the Pulumi NanoVMs provider with various cloud providers.

## Prerequisites

1. **Cloud provider credentials**: Set up authentication for your target provider
   - Azure: `az login` or environment variables
   - AWS: AWS credentials configured
   - GCP: `gcloud auth login` or service account
   - DO: `DO_TOKEN` environment variable
   - etc.

2. **Binary**: A Linux binary named "example" (automatically copied by test script)

## Configuration

The test is configured via environment variables:

- `TEST_PROVIDER`: Cloud provider to test (default: "onprem")
- `TEST_BUCKET`: Storage bucket/container name (default: "ops-images")
- `TEST_ZONE`: Cloud region/zone (default: "westus2")

## Running the Test

Use the test script from the repository root:

```sh
# Test with Azure
./test-providers.sh azure

# Test with DO (Digital Ocean)
./test-providers.sh do

# Test with GCP
./test-providers.sh gcp

# Test with custom bucket and zone
export TEST_BUCKET=my-storage
export TEST_ZONE=eastus2
./test-providers.sh azure
```

The script will:
1. Set `TEST_PROVIDER` environment variable
2. Copy the example binary
3. Install dependencies
4. Create a provider-specific Pulumi stack
5. Deploy the image and instance
6. Verify the instance is running
7. Tear down resources
8. Remove the stack

## Manual Testing

If you prefer to run manually:

```sh
cd tests/provider
export TEST_PROVIDER=azure
export TEST_BUCKET=my-bucket
export TEST_ZONE=westus2
bun install
pulumi stack init azure-test
pulumi up
```

## Verification

You can verify with ops CLI:
```sh
ops instance list -t azure
# or
ops instance list -t ${TEST_PROVIDER}
```
