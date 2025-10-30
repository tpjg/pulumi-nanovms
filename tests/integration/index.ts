import * as pulumi from "@pulumi/pulumi";
import * as nanovms from "@tpjg/nanovms";

// Get the example application binary path from environment or use default
const exampleBinary =
  process.env.EXAMPLE_BINARY || "../../examples/application/example/example";

// Configuration for the test unikernel
const config = {
  Env: {
    TEST_VAR: "integration-test",
  },
  RunConfig: {
    Memory: "256M",
    Ports: ["8888"],
  },
};

const configJson = JSON.stringify(config);

// Create a NanoVMs image for the test application
const image = new nanovms.Image(
  "integration-test-image",
  {
    name: "integration-test",
    elf: exampleBinary,
    provider: "onprem", // Use onprem (QEMU) for testing
    config: configJson,
    force: true,
    useLatestKernel: false,
  },
  { retainOnDelete: false },
);

// Deploy the image as an instance
const instance = new nanovms.Instance(
  "integration-test-instance",
  {
    image: image.imageName,
    config: image.config,
    provider: "onprem",
  },
  { dependsOn: [image] },
);

// Export outputs for verification
export const imageName = image.imageName;
export const imagePath = image.imagePath;
export const instanceId = instance.instanceID;
export const instanceStatus = instance.status;
export const publicIPs = instance.public_ips;
export const privateIPs = instance.private_ips;
