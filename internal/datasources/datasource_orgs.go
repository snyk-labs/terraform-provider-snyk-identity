package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &OrgsDataSource{}

type OrgsDataSource struct {
	api *client.Client
}

func NewOrgsDataSource() datasource.DataSource {
	return &OrgsDataSource{}
}

func (d *OrgsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_orgs"
}

func (d *OrgsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all organizations in a Snyk group. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/orgs#get-groups-group_id-orgs) GET /groups/{group_id}/orgs.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use group_id to reference.",
			},
			"orgs": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of organizations in the group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique identifier of the organization.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Resource type (e.g. org).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Name of the organization (from attributes.name).",
						},
						"slug": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "URL slug of the organization (from attributes.slug).",
						},
						"group_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Group UUID the organization belongs to (from attributes.group_id).",
						},
					},
				},
			},
		},
	}
}

type OrgsDataSourceModel struct {
	GroupID types.String  `tfsdk:"group_id"`
	ID      types.String  `tfsdk:"id"`
	Orgs    []orgRefModel `tfsdk:"orgs"`
}

// orgRefModel matches the nested object schema (id, type, name, slug, group_id).
type orgRefModel struct {
	ID      types.String `tfsdk:"id"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	Slug    types.String `tfsdk:"slug"`
	GroupID types.String `tfsdk:"group_id"`
}

func (d *OrgsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrgsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config OrgsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	orgs, err := d.api.ListGroupOrgs(groupID)
	if err != nil {
		resp.Diagnostics.AddError("list group orgs failed", err.Error())
		return
	}

	orgRefs := make([]orgRefModel, 0, len(orgs))
	for _, o := range orgs {
		orgRefs = append(orgRefs, orgRefModel{
			ID:      types.StringValue(o.ID),
			Type:    types.StringValue(o.Type),
			Name:    types.StringValue(orgAttrString(o.Attributes, "name")),
			Slug:    types.StringValue(orgAttrString(o.Attributes, "slug")),
			GroupID: types.StringValue(orgAttrString(o.Attributes, "group_id")),
		})
	}

	state := OrgsDataSourceModel{
		GroupID: config.GroupID,
		ID:      types.StringValue(groupID),
		Orgs:    orgRefs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// orgAttrString returns the string value for key from a JSON:API attributes map, or "" if missing.
func orgAttrString(attrs map[string]interface{}, key string) string {
	if attrs == nil {
		return ""
	}
	v, ok := attrs[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
