package datasources_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/datasources"
)

func TestSSOConnectionsDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewSSOConnectionsDataSource()
	assertDataSource(t, d, "snyk", "snyk_sso_connections")
}

func TestSSOConnectionUsersDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewSSOConnectionUsersDataSource()
	assertDataSource(t, d, "snyk", "snyk_sso_connection_users")
}

func TestOrgsDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewOrgsDataSource()
	assertDataSource(t, d, "snyk", "snyk_orgs")
}

func TestOrgMembershipsDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewOrgMembershipsDataSource()
	assertDataSource(t, d, "snyk", "snyk_org_memberships")
}

func TestRolesDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewRolesDataSource()
	assertDataSource(t, d, "snyk", "snyk_roles")
}

func TestGroupDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewGroupDataSource()
	assertDataSource(t, d, "snyk", "snyk_group")
}

func TestGroupMembershipsDataSource(t *testing.T) {
	t.Parallel()
	d := datasources.NewGroupMembershipsDataSource()
	assertDataSource(t, d, "snyk", "snyk_group_memberships")
}

func assertDataSource(t *testing.T, ds datasource.DataSource, providerTypeName, wantType string) {
	t.Helper()
	var meta datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: providerTypeName}, &meta)
	if meta.TypeName != wantType {
		t.Errorf("TypeName = %q, want %q", meta.TypeName, wantType)
	}
	var schemaResp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Attributes == nil {
		t.Error("nil schema attributes")
	}
}
