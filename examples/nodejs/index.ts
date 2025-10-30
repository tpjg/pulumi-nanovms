import * as pulumi from "@pulumi/pulumi";
import * as nanovms from "@tpjg/nanovms";

// Create configuration object matching the Go example
const config = {
  Env: { BAR: "3600" },
  RunConfig: {
    ShowDebug: true,
    // Bridged: false,
    // Memory: "2G",
  },
  CloudConfig: {
    BucketName: "ops-1992",
    Zone: "ams3",
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
    provider: "onprem",
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
