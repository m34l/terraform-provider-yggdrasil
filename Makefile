BINARY=terraform-provider-yggdrasil

build:
	go build -o bin/$(BINARY) .

install-local: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/m34l/yggdrasil/0.1.0/darwin_amd64
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/m34l/yggdrasil/0.1.0/darwin_arm64
	# ganti GOOS/GOARCH sesuai mesinmu, atau gunakan goreleaser

docs:
	tfplugindocs

test:
	go test ./...

acc:
	TF_ACC=1 go test ./... -v -timeout=30m
