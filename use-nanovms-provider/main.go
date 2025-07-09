package main

import (
	"encoding/json"

	"github.com/nanovms/ops/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	ops "github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms"
)

func main() {

	cfg := types.Config{
		Env: map[string]string{"BAR": "3600"},
		RunConfig: types.RunConfig{
			Bridged: false,
			Memory:  "2G",
		},
	}
	config, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	pulumi.Run(func(ctx *pulumi.Context) error {
		img, err := ops.NewImage(ctx, "test", &ops.ImageArgs{
			Name:            pulumi.String("test-image"),
			Elf:             pulumi.String("example"),
			Provider:        pulumi.String("onprem"),
			Config:          pulumi.String(config),
			Force:           pulumi.Bool(true),
			UseLatestKernel: pulumi.Bool(false),
		}, pulumi.RetainOnDelete(false))
		if err != nil {
			return err
		}

		ctx.Export("imageId", img.ImageId)
		ctx.Export("checksum", img.Checksum)
		//ctx.Export("config", img.Config)
		ctx.Export("path", img.ImagePath)

		instance, err := ops.NewInstance(ctx, "test-instance", &ops.InstanceArgs{
			Image:    pulumi.String("test-image"),
			Config:   img.Config,
			Provider: img.Provider,
		}, pulumi.DependsOn([]pulumi.Resource{img}))
		if err != nil {
			return err
		}

		ctx.Export("instanceId", instance.Instance)
		ctx.Export("instanceImage", instance.Image)
		//ctx.Export("instanceConfig", instance.Config)
		ctx.Export("instanceProvider", instance.Provider)

		return nil
	})
}
