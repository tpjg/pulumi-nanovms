package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version can be set via ldflags during build:
// go build -ldflags "-X main.Version=1.0.0"
var Version = "0.1.0"

func main() {
	provider, err := newProvider()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}

	err = provider.Run(context.Background(), "nanovms", Version)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}

func newProvider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithResources(
			infer.Resource(&Image{}),
			infer.Resource(&PackageImage{}),
			infer.Resource(&Instance{}),
		).
		WithNamespace("tpjg").
		WithDisplayName("pulumi-nanovms").
		WithDescription("A provider for NanoVMs with pulumi-go-provider.").
		WithHomepage("https://www.pulumi.com").
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"pulumi-nanovms": "index",
		}).
		Build()
}
