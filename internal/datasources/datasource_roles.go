package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &RolesDataSource{}

type RolesDataSource struct {
	api *client.Client
}

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

func (d *RolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all roles in a Snyk group. Uses the [Snyk v1 API](https://docs.snyk.io/snyk-api/reference/groups-v1#get-group-groupid-roles) GET /v1/group/{group_id}/roles (single response, no paging).",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use group_id to reference.",
			},
			"roles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of roles in the group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Display name of the role.",
						},
						"custom_role": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the role is a custom role.",
						},
						"public_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique identifier (UUID) of the role.",
						},
					},
				},
			},
		},
	}
}

type RolesDataSourceModel struct {
	GroupID types.String   `tfsdk:"group_id"`
	ID      types.String   `tfsdk:"id"`
	Roles   []roleRefModel `tfsdk:"roles"`
}

// roleRefModel matches the nested object schema (name, custom_role, public_id).
type roleRefModel struct {
	Name       types.String `tfsdk:"name"`
	CustomRole types.Bool   `tfsdk:"custom_role"`
	PublicID   types.String `tfsdk:"public_id"`
}

func (d *RolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config RolesDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	roles, err := d.api.ListGroupRolesV1(groupID)
	if err != nil {
		resp.Diagnostics.AddError("list group roles failed", err.Error())
		return
	}

	roleRefs := make([]roleRefModel, 0, len(roles))
	for _, r := range roles {
		roleRefs = append(roleRefs, roleRefModel{
			Name:       types.StringValue(r.Name),
			CustomRole: types.BoolValue(r.CustomRole),
			PublicID:   types.StringValue(r.PublicID),
		})
	}

	state := RolesDataSourceModel{
		GroupID: config.GroupID,
		ID:      types.StringValue(groupID),
		Roles:   roleRefs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
