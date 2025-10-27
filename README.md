# Pulumi NanoVMs Provider

A Pulumi provider for building and deploying unikernel images using [NanoVMs](https://nanovms.com).

[![Release](https://github.com/tpjg/pulumi-nanovms/actions/workflows/release.yml/badge.svg)](https://github.com/tpjg/pulumi-nanovms/actions/workflows/release.yml)
[![Test](https://github.com/tpjg/pulumi-nanovms/actions/workflows/test.yml/badge.svg)](https://github.com/tpjg/pulumi-nanovms/actions/workflows/test.yml)

## For Users

**Looking to use this provider?** See the [complete documentation](./docs/_index.md) for:
- Installation instructions for all languages (TypeScript, Python, Go, C#)
- Usage examples and code samples
- Configuration guide
- API reference

Quick install:
```bash
# Node.js
bun install @tpjg/pulumi-nanovms

# Python
pip install tpjg-pulumi-nanovms

# Go
go get github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms

# .NET
dotnet add package Tpjg.PulumiNanovms
```

---

## For Developers

This section is for those who want to contribute to or modify the provider itself.

### Repository Structure

```
pulumi-nanovms/
├── provider/             # Provider source code (Go)
│   ├── main.go           # Entry point, provider setup
│   ├── image.go          # Image resource implementation
│   ├── instance.go       # Instance resource implementation
│   ├── utils.go          # Helper utilities
│   ├── schema.json       # Generated Pulumi schema
│   ├── build-sdk.sh      # SDK generation script
│   └── go.mod            # Go dependencies
├── sdk/                  # Generated SDKs (auto-generated, do not edit)
│   ├── nodejs/           # TypeScript/JavaScript SDK
│   ├── python/           # Python SDK
│   ├── dotnet/           # .NET SDK
│   └── go/               # Go SDK
├── examples/             # Example Pulumi programs
│   ├── nodejs/           # TypeScript examples
│   ├── go/               # Go examples
│   └── application/      # Sample applications to deploy
├── tests/
│   └── integration/      # Integration test Pulumi program
├── docs/                 # Provider documentation
│   ├── _index.md         # Main documentation page
│   └── installation-configuration.md
├── .github/
│   └── workflows/        # CI/CD pipelines
│       ├── test.yml      # CI workflow (PRs and commits)
│       ├── release.yml   # Release workflow (tags)
│       └── integration-test.yml  # Integration tests (manual trigger)
├── test-locally.sh               # Local CI testing script
└── test-integration-local.sh     # Local integration testing script
```

### Prerequisites for Development

1. **Go 1.25+** - Provider is written in Go
   ```bash
   go version  # Should be 1.25 or later
   ```

2. **Pulumi CLI** - For schema generation and SDK building
   ```bash
   pulumi version
   ```

3. **NanoVMs ops CLI** - The underlying tool for building unikernels, includes qemu (and its tools such as `qemu-img`)
   ```bash
   brew install nanovms/ops/ops  # macOS
   # or
   curl https://ops.city/get.sh -sSfL | sh  # Linux
   ```

4. **Language SDKs** (for building and testing SDKs):
   - Bun (or Node.js 18+)
   - Python 3.11+
   - .NET 6.0+ (optional)

### Building the Provider

#### Quick Build

```bash
cd provider
go build -o pulumi-nanovms
```

#### Build with Version Injection

```bash
cd provider
VERSION=0.2.0 go build -ldflags "-X main.Version=0.2.0" -o pulumi-nanovms
```

The version can also be automatically extracted from git tags:

```bash
cd provider
./build-sdk.sh  # Uses git tags or defaults to 0.1.0
```

### Generating SDKs

The provider generates SDKs for 4 languages using Pulumi's code generation:

```bash
cd provider
./build-sdk.sh
```

This script:
1. Builds the provider binary
2. Generates the Pulumi schema (`schema.json`)
3. Generates SDKs for all languages
4. Initializes the Go SDK module

**Important**: The `sdk/` directory is auto-generated. Don't edit SDK files directly - modify the provider source instead.

### Local Development Workflow

1. **Make changes** to provider code in `provider/`

2. **Rebuild and regenerate SDKs**:
   ```bash
   cd provider
   ./build-sdk.sh
   ```

3. **Test with an example**:
   ```bash
   cd examples/nodejs
   bun install
   bun run build
   # Review the TypeScript compilation for errors
   ```

### Testing

#### Manual Testing

Test the provider with a real Pulumi program:

```bash
cd examples/nodejs
bun install
export DO_TOKEN="your-token"  # the example is using DigitalOcean
pulumi stack init dev
pulumi up
```

#### Automated Testing

Run the test suite locally:

```bash
./test-locally.sh
```

This validates:
- Provider builds successfully
- Schema generation works
- All 4 SDKs generate and compile
- Example projects build

#### Integration Testing

Integration tests verify the full end-to-end workflow by actually building and deploying a unikernel.

**Run integration tests locally:**

```bash
./test-integration-local.sh
```

This comprehensive test:
1. Builds a sample application (`examples/application/example`)
2. Builds the provider and generates SDKs
3. Runs the integration test Pulumi program (`tests/integration`)
4. Creates a unikernel image and deploys it with QEMU
5. Tests that the HTTP server responds correctly
6. Cleans up all resources

**The test program:** `tests/integration/` contains a Pulumi program similar to the examples but configured for local testing with the onprem provider.

**Prerequisites for integration tests:**
- ops CLI installed (`brew install nanovms/ops/ops`)
- QEMU installed (included with ops)
- Pulumi CLI
- disk space for kernel and images

**Why not in CI?**

Integration tests require QEMU with KVM acceleration for reasonable performance. GitHub's free `ubuntu-latest` runners don't support hardware virtualization, which would make tests extremely slow.

Integration tests run locally where KVM/HVF acceleration is available.

**Running integration tests in CI:**

A manual GitHub Actions workflow is available for integration testing:

1. Go to the repository on GitHub
2. Click **Actions** → **Integration Test**
3. Click **Run workflow**
4. Wait ... (slow due to software-emulated QEMU)

This workflow runs the same test as `./test-integration-local.sh` but in GitHub's CI environment without hardware acceleration. Use it for:
- Pre-release validation
- Verifying changes to the provider core
- Testing before major releases

### Making Changes

#### Adding a New Resource

1. Create a new file in `provider/` (e.g., `volume.go`)
2. Implement the resource following the pattern in `image.go` or `instance.go`
3. Register it in `main.go`:
   ```go
   return infer.NewProviderBuilder().
       WithResources(
           infer.Resource(&Image{}),
           infer.Resource(&Instance{}),
           infer.Resource(&Volume{}),  // New resource
       ).
       Build()
   ```
4. Rebuild and regenerate SDKs: `./build-sdk.sh`

#### Modifying an Existing Resource

1. Edit the resource file (e.g., `provider/image.go`)
2. Update input/output types and methods
3. Rebuild: `cd provider && go build`
4. Regenerate SDKs: `./build-sdk.sh`
5. Test with examples

### Version Management

Versions are managed through git tags:

- Development: Uses `0.1.0` or git tag if available
- CI/CD: Automatically extracts version from git tags

The version is injected at build time into:
- Provider binary (`main.Version`)
- Schema (`schema.json`)
- All SDK packages

### Creating a Release

1. **Ensure everything is committed and pushed**:
   ```bash
   git status  # Should be clean
   ```

2. **Tag the release**:
   ```bash
   git tag v0.2.0
   git push --tags
   ```

3. **GitHub Actions automatically**:
   - Builds provider binaries for all supported platforms (Linux, macOS × amd64/arm64)
   - Note that Windows binaries are not supported yet, due to limitations in the 'ops' source code, especially related to onprem and qemu.
   - Generates all SDKs with the tagged version
   - Creates a GitHub release with binaries attached

4. **Monitor the release**:
   - Go to GitHub → Actions → Release workflow
   - Check for any failures
   - Verify GitHub Release is created

### Publishing SDKs

Publishing to package registries requires secrets to be configured:

1. **Add secrets** to GitHub repository (Settings → Secrets → Actions):
   - `NPM_TOKEN` - npm access token
   - `PYPI_TOKEN` - PyPI API token
   - `NUGET_API_KEY` - NuGet API key

2. **Enable publishing** in `.github/workflows/release.yml`:
   - Uncomment the `bun publish`, `twine upload`, and `dotnet nuget push` lines

3. **Tag and release** as normal - SDKs will be published automatically

### CI/CD Workflows

#### Test Workflow (`.github/workflows/test.yml`)
- **Triggers**: Push to main, Pull Requests
- **Purpose**: Validates builds and SDKs
- **Runs**: Provider build, SDK generation, example compilation

#### Release Workflow (`.github/workflows/release.yml`)
- **Triggers**: Git tags matching `v*.*.*`
- **Purpose**: Creates releases and publishes SDKs
- **Jobs**:
  1. `build-provider` - Multi-platform binary builds
  2. `generate-sdks` - SDK generation and publishing
  3. `create-release` - GitHub release creation

### Project Dependencies

The provider uses a custom fork of the NanoVMs ops library:

```go
replace github.com/nanovms/ops => github.com/tpjg/ops v0.1.43-tg3
```

This fork includes modifications needed for the Pulumi provider integration, such as:
- creating a session when using qemu (so the instance is not killed when the pulumi process is killed)
- a bugfix in the digital ocean provider when creating the image URL to allow images in private spaces

### Code Style

- Follow standard Go conventions (`gofmt`)
- Use the `pulumi-go-provider` framework patterns, see [Pulumi Provider SDK](https://www.pulumi.com/docs/iac/guides/building-extending/providers/pulumi-provider-sdk)
- Keep resources focused and single-purpose

### Troubleshooting Development Issues

**SDK generation fails**:
- Ensure Pulumi CLI is installed and up to date
- Check that the provider binary builds successfully
- Verify `schema.json` is generated correctly

**Provider build errors**:
- Check Go version (must be 1.25+)
- Run `go mod tidy` in `provider/`
- Verify the ops fork is accessible

**Example projects fail**:
- Rebuild SDKs: `cd provider && ./build-sdk.sh`
- Reinstall dependencies in example: `cd examples/nodejs && bun install`
- Check that binary paths in examples are correct

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`./test-locally.sh`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Resources

- [Pulumi Provider Development Guide](https://www.pulumi.com/docs/iac/guides/building-extending/providers/build-a-provider/)
- [pulumi-go-provider Framework](https://github.com/pulumi/pulumi-go-provider)
- [NanoVMs ops Source](https://github.com/nanovms/ops)

### License

See [LICENSE](./LICENSE) file for details.

### Questions?

- Open an issue on GitHub
- Check existing documentation in `docs/`
