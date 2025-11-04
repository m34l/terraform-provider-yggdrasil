BINARY=terraform-provider-yggdrasil
TFPLUGINDOCS=$(shell go env GOPATH)/bin/tfplugindocs

build:
	go build -o bin/$(BINARY) .

install-local: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/m34l/yggdrasil/0.1.0/darwin_amd64
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/m34l/yggdrasil/0.1.0/darwin_arm64

docs:
	@if [ ! -f $(TFPLUGINDOCS) ]; then \
		echo "Installing tfplugindocs..."; \
		go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest; \
	fi
	$(TFPLUGINDOCS) generate

docs-validate:
	$(TFPLUGINDOCS) validate

test:
	go test ./...

acc:
	TF_ACC=1 go test ./... -v -timeout=30m

clean:
	rm -rf bin/ dist/

.PHONY: build install-local docs docs-validate test acc clean
