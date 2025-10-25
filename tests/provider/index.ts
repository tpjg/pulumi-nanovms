import * as pulumi from "@pulumi/pulumi";
import * as nanovms from "@tpjg/pulumi-nanovms";

// Read configuration from environment variables
const provider = process.env.TEST_PROVIDER || "onprem";
const bucketname = process.env.TEST_BUCKET || "ops-images";
const zone = process.env.TEST_ZONE || "westus2";

// Create configuration object
const config = {
  Env: { TESTING: "yes" },
  RunConfig: {},
  CloudConfig: {
    BucketName: bucketname,
    Zone: zone,
  },
};

// Convert config to JSON string
const configJson = JSON.stringify(config);

// Create a NanoVMs image
const img = new nanovms.Image(
  "test",
  {
    name: "test-image",
    elf: "example",
    provider: provider,
    config: configJson,
    force: true,
    useLatestKernel: false,
  },
  { retainOnDelete: false },
);

// Export image outputs
export const imageName = img.imageName;
export const path = img.imagePath;

// Create a NanoVMs instance
const instance = new nanovms.Instance(
  "test-instance",
  {
    image: "test-image",
    config: img.config,
    provider: img.provider,
  },
  { dependsOn: [img] },
);

// Export instance outputs
export const instanceId = instance.instanceID;
export const instanceImage = instance.image;
export const instanceProvider = instance.provider;
export const instanceIPs = instance.public_ips;
export const instanceStatus = instance.status;
