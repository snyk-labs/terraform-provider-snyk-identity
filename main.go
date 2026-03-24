package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/provider"
)

func main() {
	if err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/snyk-labs/snyk-identity",
	}); err != nil {
		log.Fatal(err)
	}
}
