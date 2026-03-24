package resources_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/resources"
)

func TestOrgMembershipResource_metadataAndSchema(t *testing.T) {
	t.Parallel()
	r := resources.NewOrgMembershipResource()
	var meta resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "snyk"}, &meta)
	if meta.TypeName != "snyk_org_membership" {
		t.Errorf("TypeName = %q", meta.TypeName)
	}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Attributes == nil {
		t.Fatal("nil schema")
	}
}

func TestGroupMembershipResource_metadataAndSchema(t *testing.T) {
	t.Parallel()
	r := resources.NewGroupMembershipResource()
	var meta resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "snyk"}, &meta)
	if meta.TypeName != "snyk_group_membership" {
		t.Errorf("TypeName = %q", meta.TypeName)
	}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Attributes == nil {
		t.Fatal("nil schema")
	}
}

func TestOrgMembershipResource_Configure_client(t *testing.T) {
	t.Parallel()
	r := resources.NewOrgMembershipResource().(*resources.OrgMembershipResource)
	c, err := client.New("tok", "https://api.snyk.io")
	if err != nil {
		t.Fatal(err)
	}
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func TestOrgMembershipResource_Configure_invalid(t *testing.T) {
	t.Parallel()
	r := resources.NewOrgMembershipResource().(*resources.OrgMembershipResource)
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
}

func TestGroupMembershipResource_Configure_invalid(t *testing.T) {
	t.Parallel()
	r := resources.NewGroupMembershipResource().(*resources.GroupMembershipResource)
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 42}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
}
