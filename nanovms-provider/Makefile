VERSION ?= 0.1.0
BINARY_NAME = pulumi-resource-nanovms

.PHONY: build install clean test

build:
	go build -o ${BINARY_NAME} .

install: build
	cp ${BINARY_NAME} ${GOPATH}/bin/

clean:
	rm -f ${BINARY_NAME}

test:
	go test ./...

dev: build
	cp ${BINARY_NAME} ~/.pulumi/bin/

.PHONY: schema
schema:
	pulumi package gen-schema schema.json

.PHONY: gen
gen:
	pulumi package gen-sdk schema.json --language go --out sdk/go
	pulumi package gen-sdk schema.json --language nodejs --out sdk/nodejs
	pulumi package gen-sdk schema.json --language python --out sdk/python
	pulumi package gen-sdk schema.json --language dotnet --out sdk/dotnet