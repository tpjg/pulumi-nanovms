package main

import (
	"encoding/json"

	"github.com/nanovms/ops/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	ops "github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms"
)

func main() {

	cfg := types.Config{
		Env:       map[string]string{"BAR": "3600"},
		Klibs:     []string{"tls", "userdata_env"},
		RunConfig: types.RunConfig{},
		CloudConfig: types.ProviderConfig{
			Zone: "eu-central-1",
			UserData: `ENV_FROM_USER_DATA=test1
ENV_NUMBER2=test2
`,
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
			Provider:        pulumi.String("aws"),
			Config:          pulumi.String(config),
			Force:           pulumi.Bool(true),
			UseLatestKernel: pulumi.Bool(false),
		}, pulumi.RetainOnDelete(false))
		if err != nil {
			return err
		}

		ctx.Export("imageName", img.ImageName)
		ctx.Export("path", img.ImagePath)

		instance, err := ops.NewInstance(ctx, "test-instance", &ops.InstanceArgs{
			Image:    pulumi.String("test-image"),
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
