Put a Linux image with the name "example" in this folder.
E.g. do something like this:

```sh
cd ../application/example
GOOS=linux GOOS=amd64 go build
cp example ../../go
cd ../../go
```

Set your Digital Ocean API token (DO_TOKEN) and SPACES secret and token.
Then run `pulumi up` to create the image and the instance, see main.go in this folder.
Or change main.go to use a different provider.

Run `pulumi refresh && pulumi up` to update the outputs with information about the running instance.
