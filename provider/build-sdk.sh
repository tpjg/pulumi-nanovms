#!/bin/sh
go build
pulumi package get-schema ./pulumi-nanovms >schema.json
pulumi package gen-sdk . --local --language=go --out ../sdk
cd ../sdk/go/pulumi-nanovms && go mod init github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms && go mod tidy
