package main

import (
	"encoding/json"

	"github.com/nanovms/ops/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	ops "github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms"
)

func main() {

	cfg := types.Config{
		RunConfig: types.RunConfig{
			Bridged: false,
			Memory:  "1G",
		},
	}
	config, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	pulumi.Run(func(ctx *pulumi.Context) error {
		img, err := ops.NewImage(ctx, "test", &ops.ImageArgs{
			Name:     pulumi.String("test-image"),
			Elf:      pulumi.String("example"),
			Provider: pulumi.String("onprem"),
			Config:   pulumi.String(config),
		})
		if err != nil {
			return err
		}

		ctx.Export("imageId", img.ImageId)
		ctx.Export("checksum", img.Checksum)
		ctx.Export("config", img.Config)
		ctx.Export("path", img.ImagePath)
		return nil
	})
}
