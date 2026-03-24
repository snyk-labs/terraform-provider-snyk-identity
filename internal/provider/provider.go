package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/datasources"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/resources"
)

var _ provider.Provider = &SnykIdentityProvider{}

type SnykIdentityProvider struct{}

type SnykIdentityProviderModel struct {
	APIToken    types.String `tfsdk:"api_token"`
	APIEndpoint types.String `tfsdk:"api_endpoint"`
}

// deriveAPIBaseURL builds the HTTP client base URL from api_endpoint.
// Empty input defaults to https://api.snyk.io. Values without a scheme get https:// prepended.
// Values that already start with http:// or https:// are used as-is (trailing slash trimmed).
func deriveAPIBaseURL(apiEndpoint string) string {
	s := strings.TrimSpace(apiEndpoint)
	if s == "" {
		return "https://api.snyk.io"
	}
	s = strings.TrimRight(s, "/")
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return s
	}
	return "https://" + strings.TrimLeft(s, "/")
}

func New() provider.Provider {
	return &SnykIdentityProvider{}
}

func (p *SnykIdentityProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "snyk"
}

func (p *SnykIdentityProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Snyk API token for authentication. Must have org.membership.add and/or group.membership.add as needed.",
			},
			"api_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "Snyk API endpoint hostname or URL (e.g. api.snyk.io). If no scheme is set, https:// is prepended. " +
					"Unset uses https://api.snyk.io.",
			},
		},
	}
}

func (p *SnykIdentityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config SnykIdentityProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIToken.IsNull() || config.APIToken.ValueString() == "" {
		resp.Diagnostics.AddError("api_token required", "Snyk API token must be set in provider configuration")
		return
	}

	endpoint := ""
	if !config.APIEndpoint.IsNull() {
		endpoint = config.APIEndpoint.ValueString()
	}
	baseURL := deriveAPIBaseURL(endpoint)

	c, err := client.New(config.APIToken.ValueString(), baseURL)
	if err != nil {
		resp.Diagnostics.AddError("client creation failed", err.Error())
		return
	}

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *SnykIdentityProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewOrgMembershipResource,
		resources.NewGroupMembershipResource,
	}
}

func (p *SnykIdentityProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewSSOConnectionsDataSource,
		datasources.NewSSOConnectionUsersDataSource,
		datasources.NewOrgsDataSource,
		datasources.NewOrgMembershipsDataSource,
		datasources.NewRolesDataSource,
		datasources.NewGroupDataSource,
		datasources.NewGroupMembershipsDataSource,
	}
}
