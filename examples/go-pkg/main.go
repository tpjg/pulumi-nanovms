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
			ShowDebug: true,
			//Bridged: false,
			//Memory:  "2G",
		},
		CloudConfig: types.ProviderConfig{
			BucketName: "ops-1992",
			Zone:       "ams3",
		},
		Args: []string{"hi.js"},
	}
	config, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	pulumi.Run(func(ctx *pulumi.Context) error {
		img, err := ops.NewPackageImage(ctx, "test", &ops.PackageImageArgs{
			Name:            pulumi.String("test-pkg-image"),
			PackageName:     pulumi.String("eyberg/node:20.5.0"),
			Provider:        pulumi.String("do"),
			Config:          pulumi.String(config),
			Force:           pulumi.Bool(true),
			UseLatestKernel: pulumi.Bool(false),
			Architecture:    pulumi.String("amd64"),
		}, pulumi.RetainOnDelete(false))
		if err != nil {
			return err
		}

		ctx.Export("imageName", img.ImageName)
		ctx.Export("path", img.ImagePath)

		instance, err := ops.NewInstance(ctx, "test-pkg-instance", &ops.InstanceArgs{
			Image:    pulumi.String("test-pkg-image"),
			Config:   img.Config,
			Provider: img.Provider,
		}, pulumi.DependsOn([]pulumi.Resource{img}))
		if err != nil {
			return err
		}

		ctx.Export("instanceId", instance.InstanceID)
		ctx.Export("instanceImage", instance.Image)
		ctx.Export("instanceProvider", instance.Provider)
		ctx.Export("instanceIPs", instance.Public_ips)
		ctx.Export("instanceStatus", instance.Status)

		return nil
	})
}
