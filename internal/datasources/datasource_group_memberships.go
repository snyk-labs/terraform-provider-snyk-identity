package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &GroupMembershipsDataSource{}

type GroupMembershipsDataSource struct {
	api *client.Client
}

func NewGroupMembershipsDataSource() datasource.DataSource {
	return &GroupMembershipsDataSource{}
}

func (d *GroupMembershipsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_memberships"
}

func (d *GroupMembershipsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all memberships of a Snyk group. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-memberships) GET /groups/{group_id}/memberships.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use group_id to reference.",
			},
			"memberships": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All group memberships returned by the API (paginated internally).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Group membership UUID.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Resource type (e.g. group_membership).",
						},
						"user_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User UUID from the membership relationship.",
						},
						"role_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Group role UUID from the membership relationship.",
						},
					},
				},
			},
		},
	}
}

type GroupMembershipsDataSourceModel struct {
	GroupID     types.String              `tfsdk:"group_id"`
	ID          types.String              `tfsdk:"id"`
	Memberships []groupMembershipRefModel `tfsdk:"memberships"`
}

type groupMembershipRefModel struct {
	ID     types.String `tfsdk:"id"`
	Type   types.String `tfsdk:"type"`
	UserID types.String `tfsdk:"user_id"`
	RoleID types.String `tfsdk:"role_id"`
}

func (d *GroupMembershipsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupMembershipsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config GroupMembershipsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	items, err := d.api.ListGroupMemberships(groupID)
	if err != nil {
		resp.Diagnostics.AddError("list group memberships failed", err.Error())
		return
	}

	refs := make([]groupMembershipRefModel, 0, len(items))
	for _, m := range items {
		refs = append(refs, groupMembershipRefModel{
			ID:     types.StringValue(m.ID),
			Type:   types.StringValue(m.Type),
			UserID: types.StringValue(m.Relationships.User.Data.ID),
			RoleID: types.StringValue(m.Relationships.Role.Data.ID),
		})
	}

	state := GroupMembershipsDataSourceModel{
		GroupID:     config.GroupID,
		ID:          types.StringValue(groupID),
		Memberships: refs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
