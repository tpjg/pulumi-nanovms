---
title: Installation & Configuration
meta_desc: Installation and configuration guide for the Pulumi NanoVMs provider.
layout: package
---

## Installation

The Pulumi NanoVMs provider is available for multiple languages. Follow the instructions below to install the SDK for your language of choice.

### TypeScript/JavaScript (Node.js)

```bash
npm install @tpjg/pulumi-nanovms
```

Or with Yarn:

```bash
yarn add @tpjg/pulumi-nanovms
```

Or with Bun:

```bash
bun add @tpjg/pulumi-nanovms
```

### Python

```bash
pip install tpjg-pulumi-nanovms
```

### Go

```bash
go get github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms
```

### .NET (C#/F#)

```bash
dotnet add package Tpjg.PulumiNanovms
```

## Prerequisites

Before using the NanoVMs provider, ensure you have:

1. **Pulumi CLI** installed ([Installation Guide](https://www.pulumi.com/docs/install/))
2. **NanoVMs ops CLI** installed (this will also install qemu and its utilities)
   ```bash
   # macOS
   brew install nanovms/ops/ops

   # Linux
   curl https://ops.city/get.sh -sSfL | sh
   ```
3. **Application binary** compiled for Linux (x86_64 or arm64)
4. **Cloud provider credentials** (for cloud deployments)

## Configuration

The NanoVMs provider uses the same code that is used to build the [NanoVMs ops tool](https://github.com/nanovms/ops), it allows a Pulumi "Infrastructure as Code" style configuration instead of scripts and separate (JSON) configuration files.

### DigitalOcean Configuration

For DigitalOcean deployments, you need to configure your DO credentials:

1. **Create a DigitalOcean API token**:
   - Go to [DigitalOcean API Tokens](https://cloud.digitalocean.com/account/api/tokens)
   - Generate a new personal access token with read/write permissions

2. **Configure credentials**:

   Set environment variables:
   ```bash
   export DO_TOKEN="your-digitalocean-token"
   ```

3. **Create a DigitalOcean Space (for images)**:
   - Create a Spaces bucket (e.g., `ops-bucket`)
   - Note the bucket name and region for your configuration

### On-Premises Configuration

For local/on-premises deployments:

1. **Install required virtualization**:
   - Linux: QEMU/KVM
   - macOS: QEMU (installed automatically with ops)
   - Windows: not supported at this time

2. **No additional configuration required** - local deployments work out of the box

## Unikernel Configuration

When creating images, you provide a JSON configuration that controls the unikernel's behavior:

```javascript
const config = {
  // Environment variables for your application
  Env: {
    PORT: "8080",
    DATABASE_URL: "postgres://..."
  },

  // Runtime configuration
  RunConfig: {
    Memory: "2G",        // Memory allocation
    ShowDebug: true,     // Enable debug output
    Bridged: false,      // Network mode
  },

  // Cloud-specific configuration (for DigitalOcean)
  CloudConfig: {
    BucketName: "ops-bucket",  // Your DO Spaces bucket
    Zone: "ams3",              // DigitalOcean region
  }
};
```

### Common Configuration Options

See the [ops documentation](https://docs.ops.city/ops/configuration) for more details.

**Environment Variables (`Env`)**:
- Set any environment variables your application needs
- Example: `{ "PORT": "8080", "LOG_LEVEL": "info" }`

**RunConfig**:
- `Memory`: RAM allocation (e.g., "512M", "2G", "4G")

**CloudConfig** (for DigitalOcean):
- `BucketName`: Your DigitalOcean Spaces bucket for storing images
- `Zone`: DigitalOcean region (e.g., "nyc3", "sfo3", "ams3")

## Provider Configuration in Pulumi

The NanoVMs provider reads cloud credentials from the ops configuration files and environment variables. You don't need to configure credentials directly in your Pulumi program.

### Example: Basic Setup

```typescript
import * as nanovms from "@tpjg/pulumi-nanovms";

// The provider automatically uses credentials from:
// - ~/.ops/config.json
// - Environment variables (DO_TOKEN, etc.)

const image = new nanovms.Image("my-image", {
  name: "my-app",
  elf: "./my-app-binary",
  provider: "do",  // or "onprem"
  config: JSON.stringify({
    CloudConfig: {
      BucketName: "ops-bucket",
      Zone: "ams3"
    }
  })
});
```

## Troubleshooting

### Image Build Failures

If image builds fail:
- Verify your binary is compiled for Linux (x86_64 or arm64)
- Check that the binary path is correct
- Ensure you have sufficient disk space for the image

### Instance Deployment Issues

If instances fail to deploy:
- Verify your DigitalOcean Spaces bucket exists and is accessible
- Check that the zone/region is valid
- Ensure you have sufficient quota in your DigitalOcean account
- Review the instance status and error messages

## Next Steps

- Review the [Provider Reference](./) for detailed resource documentation
- Check out [example projects](https://github.com/tpjg/pulumi-nanovms/tree/main/examples)
- Learn more about [NanoVMs and unikernels](https://nanovms.com)

## Resources

- [Ops Documentation](https://docs.ops.city/)
- [NanoVMs ops CLI](https://github.com/nanovms/ops)
- [DigitalOcean Spaces](https://docs.digitalocean.com/products/spaces/)
- [Pulumi Documentation](https://www.pulumi.com/docs/)
