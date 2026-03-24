package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &GroupDataSource{}

type GroupDataSource struct {
	api *client.Client
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves details of a Snyk group. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/group#get-groups-group_id) GET /groups/{group_id}.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the group (same as group_id).",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource type (e.g. group).",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the group (from attributes.name).",
			},
			"slug": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL slug of the group (from attributes.slug).",
			},
		},
	}
}

type GroupDataSourceModel struct {
	GroupID types.String `tfsdk:"group_id"`
	ID      types.String `tfsdk:"id"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	Slug    types.String `tfsdk:"slug"`
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config GroupDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	group, err := d.api.GetGroup(groupID)
	if err != nil {
		resp.Diagnostics.AddError("get group failed", err.Error())
		return
	}

	state := GroupDataSourceModel{
		GroupID: config.GroupID,
		ID:      types.StringValue(group.ID),
		Type:    types.StringValue(group.Type),
		Name:    types.StringValue(groupAttrString(group.Attributes, "name")),
		Slug:    types.StringValue(groupAttrString(group.Attributes, "slug")),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// groupAttrString returns the string value for key from a JSON:API attributes map, or "" if missing.
func groupAttrString(attrs map[string]interface{}, key string) string {
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
