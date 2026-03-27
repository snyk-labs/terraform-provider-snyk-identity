package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &SSOConnectionsDataSource{}

type SSOConnectionsDataSource struct {
	api *client.Client
}

func NewSSOConnectionsDataSource() datasource.DataSource {
	return &SSOConnectionsDataSource{}
}

func (d *SSOConnectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_connections"
}

func (d *SSOConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists SSO connections for a Snyk group. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-sso_connections) GET /groups/{group_id}/sso_connections.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use group_id to reference.",
			},
			"connections": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of SSO connections for the group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique identifier of the SSO connection.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Resource type (e.g. sso_connection).",
						},
					},
				},
			},
		},
	}
}

type SSOConnectionsDataSourceModel struct {
	GroupID     types.String            `tfsdk:"group_id"`
	ID          types.String            `tfsdk:"id"`
	Connections []ssoConnectionRefModel `tfsdk:"connections"`
}

// ssoConnectionRefModel matches the nested object schema (id, type).
type ssoConnectionRefModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func (d *SSOConnectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("invalid provider data", "expected *client.Client")
		return
	}
	d.api = api
}

func (d *SSOConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config SSOConnectionsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	tflog.Debug(ctx, "Listing SSO connections for group", map[string]any{
		"group_id": groupID,
	})
	connections, err := d.api.ListGroupSSOConnections(ctx, groupID)
	if err != nil {
		tflog.Error(ctx, "List SSO connections failed", map[string]any{
			"group_id": groupID,
			"error":    err.Error(),
		})
		resp.Diagnostics.AddError("list SSO connections failed", err.Error())
		return
	}

	connectionRefs := make([]ssoConnectionRefModel, 0, len(connections))
	for _, c := range connections {
		connectionRefs = append(connectionRefs, ssoConnectionRefModel{
			ID:   types.StringValue(c.ID),
			Type: types.StringValue(c.Type),
		})
	}

	state := SSOConnectionsDataSourceModel{
		GroupID:     config.GroupID,
		ID:          types.StringValue(groupID),
		Connections: connectionRefs,
	}
	tflog.Debug(ctx, "Listed SSO connections for group", map[string]any{
		"group_id": groupID,
		"count":    len(connectionRefs),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
