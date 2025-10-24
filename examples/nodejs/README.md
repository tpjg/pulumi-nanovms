Put a Linux image with the name "example" in this folder.
E.g. do something like this:

```sh
cd ../application/example
GOOS=linux GOARCH=amd64 go build
cp example ../../nodejs
cd ../../nodejs
```

Set your Digital Ocean API token (DO_TOKEN) and SPACES secret and token.
Then install dependencies and run `pulumi up` to create the image and the instance, see index.ts in this folder.
Or change index.ts to use a different provider.

Install the dependencies in the nodejs sdk folder:
```sh
cd ../../sdk/nodejs
bun install
```

Then install the dependencies for the example and run pulumi:
```sh
bun install
pulumi up
```

Run `pulumi refresh && pulumi up` to update the outputs with information about the running instance.
