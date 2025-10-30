package main

import (
	"context"
	"flag"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/m34l/terraform-provider-yggdrasil/internal/provider"
)

var (
	version = "0.1.0"
)

func main() {
	flag.Parse()
	providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/m34l/yggdrasil",
	})
}
