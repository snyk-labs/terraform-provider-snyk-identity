package provider_test

import (
	"context"
	"testing"

	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	snykprovider "github.com/snyk-labs/terraform-provider-snyk-identity/internal/provider"
)

func TestNew(t *testing.T) {
	t.Parallel()
	p := snykprovider.New()
	if p == nil {
		t.Fatal("nil provider")
	}
	var _ tfprovider.Provider = p
}

func TestSnykIdentityProvider_Metadata(t *testing.T) {
	t.Parallel()
	p := snykprovider.New()
	var resp tfprovider.MetadataResponse
	p.Metadata(context.Background(), tfprovider.MetadataRequest{}, &resp)
	if resp.TypeName != "snyk" {
		t.Errorf("TypeName = %q", resp.TypeName)
	}
}

func TestSnykIdentityProvider_Schema(t *testing.T) {
	t.Parallel()
	p := snykprovider.New()
	var resp tfprovider.SchemaResponse
	p.Schema(context.Background(), tfprovider.SchemaRequest{}, &resp)
	if resp.Schema.Attributes == nil {
		t.Fatal("nil schema attributes")
	}
	if _, ok := resp.Schema.Attributes["api_token"]; !ok {
		t.Error("missing api_token")
	}
	if _, ok := resp.Schema.Attributes["api_endpoint"]; !ok {
		t.Error("missing api_endpoint")
	}
}

func TestSnykIdentityProvider_Resources(t *testing.T) {
	t.Parallel()
	p := snykprovider.New()
	fns := p.Resources(context.Background())
	if len(fns) != 2 {
		t.Fatalf("len(Resources) = %d", len(fns))
	}
	for i, fn := range fns {
		if fn == nil {
			t.Fatalf("nil factory at %d", i)
		}
		if fn() == nil {
			t.Fatalf("nil resource at %d", i)
		}
	}
}

func TestSnykIdentityProvider_DataSources(t *testing.T) {
	t.Parallel()
	p := snykprovider.New()
	fns := p.DataSources(context.Background())
	if len(fns) != 7 {
		t.Fatalf("len(DataSources) = %d", len(fns))
	}
	for i, fn := range fns {
		if fn == nil {
			t.Fatalf("nil factory at %d", i)
		}
		if fn() == nil {
			t.Fatalf("nil data source at %d", i)
		}
	}
}
