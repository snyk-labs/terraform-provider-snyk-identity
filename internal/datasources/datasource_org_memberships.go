package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &OrgMembershipsDataSource{}

type OrgMembershipsDataSource struct {
	api *client.Client
}

func NewOrgMembershipsDataSource() datasource.DataSource {
	return &OrgMembershipsDataSource{}
}

func (d *OrgMembershipsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_memberships"
}

func (d *OrgMembershipsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all memberships of a Snyk organization. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/orgs#get-orgs-org_id-memberships) GET /orgs/{org_id}/memberships.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk organization UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use org_id to reference.",
			},
			"memberships": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All org memberships returned by the API (paginated internally).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Org membership UUID.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Resource type (e.g. org_membership).",
						},
						"user_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User UUID from the membership relationship.",
						},
						"role_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Org role UUID from the membership relationship.",
						},
					},
				},
			},
		},
	}
}

type OrgMembershipsDataSourceModel struct {
	OrgID       types.String            `tfsdk:"org_id"`
	ID          types.String            `tfsdk:"id"`
	Memberships []orgMembershipRefModel `tfsdk:"memberships"`
}

type orgMembershipRefModel struct {
	ID     types.String `tfsdk:"id"`
	Type   types.String `tfsdk:"type"`
	UserID types.String `tfsdk:"user_id"`
	RoleID types.String `tfsdk:"role_id"`
}

func (d *OrgMembershipsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrgMembershipsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config OrgMembershipsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := config.OrgID.ValueString()
	items, err := d.api.ListOrgMemberships(orgID)
	if err != nil {
		resp.Diagnostics.AddError("list org memberships failed", err.Error())
		return
	}

	refs := make([]orgMembershipRefModel, 0, len(items))
	for _, m := range items {
		refs = append(refs, orgMembershipRefModel{
			ID:     types.StringValue(m.ID),
			Type:   types.StringValue(m.Type),
			UserID: types.StringValue(m.Relationships.User.Data.ID),
			RoleID: types.StringValue(m.Relationships.Role.Data.ID),
		})
	}

	state := OrgMembershipsDataSourceModel{
		OrgID:       config.OrgID,
		ID:          types.StringValue(orgID),
		Memberships: refs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
