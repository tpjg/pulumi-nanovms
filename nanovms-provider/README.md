# NanoVMs Pulumi Provider

A Pulumi provider for building and deploying NanoVMs unikernel images.

## Overview

This provider allows you to build NanoVMs images using Pulumi's infrastructure-as-code approach. It wraps the `ops` command-line tool to provide declarative image building capabilities.

## Prerequisites

- [OPS](https://github.com/nanovms/ops) - The NanoVMs command-line tool must be installed including a valid kernel somewhere in `$HOME/.ops`
- [Pulumi](https://www.pulumi.com/) - Infrastructure as Code platform

## License

This project is licensed under the MIT License.
