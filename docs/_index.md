---
title: NanoVMs Provider
meta_desc: The Pulumi NanoVMs provider enables you to build and deploy unikernel images to various cloud platforms.
layout: package
---

The Pulumi NanoVMs provider enables you to build and deploy applications as lightweight, secure unikernel images. Unikernels are minimal, single-purpose virtual machines that run a single application with only the necessary OS components, resulting in improved security, faster boot times, and reduced resource usage.

## Example

{{< chooser language "typescript,python,go,csharp" >}}

{{% choosable language typescript %}}

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as nanovms from "@tpjg/nanovms";

// Create configuration for the unikernel
const config = {
  Env: { BAR: "3600" },
  RunConfig: {
    ShowDebug: true,
  },
  CloudConfig: {
    BucketName: "ops-bucket",
    Zone: "ams3",
  },
};

// Build a NanoVMs unikernel image
const image = new nanovms.Image("my-app-image", {
  name: "my-app",
  elf: "./my-app-binary",  // Path to your application binary
  provider: "do",           // DigitalOcean
  config: JSON.stringify(config),
  force: true,
  useLatestKernel: false,
});

// Deploy the image as an instance
const instance = new nanovms.Instance("my-app-instance", {
  image: image.imageName,
  config: image.config,
  provider: image.provider,
}, { dependsOn: [image] });

// Export the instance details
export const instanceId = instance.instanceID;
export const publicIPs = instance.public_ips;
export const status = instance.status;
```

{{% /choosable %}}

{{% choosable language python %}}

```python
import pulumi
import json
import tpjg_pulumi_nanovms as nanovms

# Create configuration for the unikernel
config = {
    "Env": {"BAR": "3600"},
    "RunConfig": {
        "ShowDebug": True,
    },
    "CloudConfig": {
        "BucketName": "ops-bucket",
        "Zone": "ams3",
    },
}

# Build a NanoVMs unikernel image
image = nanovms.Image("my-app-image",
    name="my-app",
    elf="./my-app-binary",  # Path to your application binary
    provider="do",          # DigitalOcean
    config=json.dumps(config),
    force=True,
    use_latest_kernel=False
)

# Deploy the image as an instance
instance = nanovms.Instance("my-app-instance",
    image=image.image_name,
    config=image.config,
    provider=image.provider,
    opts=pulumi.ResourceOptions(depends_on=[image])
)

# Export the instance details
pulumi.export("instanceId", instance.instance_id)
pulumi.export("publicIPs", instance.public_ips)
pulumi.export("status", instance.status)
```

{{% /choosable %}}

{{% choosable language go %}}

```go
package main

import (
	"encoding/json"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	nanovms "github.com/tpjg/pulumi-nanovms/sdk/go/pulumi-nanovms"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create configuration for the unikernel
		config := map[string]interface{}{
			"Env": map[string]string{
				"BAR": "3600",
			},
			"RunConfig": map[string]interface{}{
				"ShowDebug": true,
			},
			"CloudConfig": map[string]interface{}{
				"BucketName": "ops-bucket",
				"Zone":       "ams3",
			},
		}

		configJSON, _ := json.Marshal(config)

		// Build a NanoVMs unikernel image
		image, err := nanovms.NewImage(ctx, "my-app-image", &nanovms.ImageArgs{
			Name:            pulumi.String("my-app"),
			Elf:             pulumi.String("./my-app-binary"),
			Provider:        pulumi.String("do"),
			Config:          pulumi.String(string(configJSON)),
			Force:           pulumi.Bool(true),
			UseLatestKernel: pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Deploy the image as an instance
		instance, err := nanovms.NewInstance(ctx, "my-app-instance", &nanovms.InstanceArgs{
			Image:    image.ImageName,
			Config:   image.Config,
			Provider: image.Provider,
		}, pulumi.DependsOn([]pulumi.Resource{image}))
		if err != nil {
			return err
		}

		// Export the instance details
		ctx.Export("instanceId", instance.InstanceID)
		ctx.Export("publicIPs", instance.Public_ips)
		ctx.Export("status", instance.Status)

		return nil
	})
}
```

{{% /choosable %}}

{{% choosable language csharp %}}

```csharp
using System.Collections.Generic;
using System.Text.Json;
using Pulumi;
using Tpjg.PulumiNanovms;

return await Deployment.RunAsync(() =>
{
    // Create configuration for the unikernel
    var config = new Dictionary<string, object>
    {
        ["Env"] = new Dictionary<string, string>
        {
            ["BAR"] = "3600"
        },
        ["RunConfig"] = new Dictionary<string, object>
        {
            ["ShowDebug"] = true
        },
        ["CloudConfig"] = new Dictionary<string, object>
        {
            ["BucketName"] = "ops-bucket",
            ["Zone"] = "ams3"
        }
    };

    var configJson = JsonSerializer.Serialize(config);

    // Build a NanoVMs unikernel image
    var image = new Image("my-app-image", new ImageArgs
    {
        Name = "my-app",
        Elf = "./my-app-binary",
        Provider = "do",
        Config = configJson,
        Force = true,
        UseLatestKernel = false
    });

    // Deploy the image as an instance
    var instance = new Instance("my-app-instance", new InstanceArgs
    {
        Image = image.ImageName,
        Config = image.Config,
        Provider = image.Provider
    }, new CustomResourceOptions
    {
        DependsOn = { image }
    });

    // Export the instance details
    return new Dictionary<string, object?>
    {
        ["instanceId"] = instance.InstanceID,
        ["publicIPs"] = instance.Public_ips,
        ["status"] = instance.Status
    };
});
```

{{% /choosable %}}

{{< /chooser >}}

## Resources

The NanoVMs provider includes the following resources:

### Image

Builds a unikernel image from your application binary. The image can be targeted for various cloud providers or on-premises deployment.

**Key Properties:**
- `name` - The name of the image
- `elf` - Path to your application executable
- `provider` - Target platform (`do` for DigitalOcean, `onprem` for local/on-premises)
- `config` - JSON configuration for the unikernel (environment variables, cloud settings, etc.)
- `force` - Whether to overwrite an existing image
- `useLatestKernel` - Whether to use the latest NanoVMs kernel

### Instance

Deploys a built unikernel image as a running instance on the target cloud provider.

**Key Properties:**
- `image` - The name of the image to deploy
- `config` - Configuration for the instance
- `provider` - Target platform for deployment

**Outputs:**
- `instanceID` - The unique identifier for the instance
- `public_ips` - Public IP addresses assigned to the instance
- `private_ips` - Private IP addresses assigned to the instance
- `status` - Current status of the instance
- `pid` - Provider-specific instance ID

## Supported Cloud Providers

- **DigitalOcean** (`do`) - Fully supported for cloud deployments
- **On-Premises** (`onprem`) - For local testing and private cloud deployments

Additional providers (GCP, AWS, Azure, etc.) are on the roadmap to be supported.

## Getting Started

To get started with the NanoVMs provider:

1. [Install and configure the provider](./installation-configuration)
2. Prepare your application binary (compiled for Linux x86_64)
3. Create a Pulumi program using one of the examples above
4. Run `pulumi up` to build and deploy your unikernel

## Learn More

- [NanoVMs Documentation](https://nanovms.com/documentation)
- [NanoVMs Ops CLI](https://github.com/nanovms/ops)
- [Installation & Configuration](./installation-configuration)
