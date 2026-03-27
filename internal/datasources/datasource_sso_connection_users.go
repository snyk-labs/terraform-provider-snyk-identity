package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ datasource.DataSource = &SSOConnectionUsersDataSource{}

type SSOConnectionUsersDataSource struct {
	api *client.Client
}

func NewSSOConnectionUsersDataSource() datasource.DataSource {
	return &SSOConnectionUsersDataSource{}
}

func (d *SSOConnectionUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_connection_users"
}

func (d *SSOConnectionUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists users of a Snyk group SSO connection. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-sso_connections-sso_id-users) GET /groups/{group_id}/sso_connections/{sso_id}/users.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
			},
			"sso_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SSO connection UUID (from the group's sso_connections).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for Terraform; use group_id and sso_id to reference.",
			},
			"users": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of users linked to the SSO connection.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique identifier of the user.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Resource type (e.g. user).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Display name of the user (from attributes.name).",
						},
						"username": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Username of the user (from attributes.username).",
						},
						"email": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Email address of the user (from attributes.email).",
						},
						"active": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the user is active (from attributes.active).",
						},
					},
				},
			},
		},
	}
}

type SSOConnectionUsersDataSourceModel struct {
	GroupID types.String                `tfsdk:"group_id"`
	SSOID   types.String                `tfsdk:"sso_id"`
	ID      types.String                `tfsdk:"id"`
	Users   []ssoConnectionUserRefModel `tfsdk:"users"`
}

// ssoConnectionUserRefModel matches the nested object schema (id, type, name, username, email, active).
type ssoConnectionUserRefModel struct {
	ID       types.String `tfsdk:"id"`
	Type     types.String `tfsdk:"type"`
	Name     types.String `tfsdk:"name"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
	Active   types.Bool   `tfsdk:"active"`
}

func (d *SSOConnectionUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SSOConnectionUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.api == nil {
		resp.Diagnostics.AddError("provider not configured", "the snyk-identity provider must be configured with api_token (and optional api_endpoint) so the data source can call the Snyk API")
		return
	}
	var config SSOConnectionUsersDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := config.GroupID.ValueString()
	ssoID := config.SSOID.ValueString()
	tflog.Debug(ctx, "Listing SSO connection users", map[string]any{
		"group_id": groupID,
		"sso_id":   ssoID,
	})
	users, err := d.api.ListGroupSSOConnectionUsers(ctx, groupID, ssoID)
	if err != nil {
		tflog.Error(ctx, "List SSO connection users failed", map[string]any{
			"group_id": groupID,
			"sso_id":   ssoID,
			"error":    err.Error(),
		})
		resp.Diagnostics.AddError("list SSO connection users failed", err.Error())
		return
	}

	userRefs := make([]ssoConnectionUserRefModel, 0, len(users))
	for _, u := range users {
		userRefs = append(userRefs, ssoConnectionUserRefModel{
			ID:       types.StringValue(u.ID),
			Type:     types.StringValue(u.Type),
			Name:     types.StringValue(attrString(u.Attributes, "name")),
			Username: types.StringValue(attrString(u.Attributes, "username")),
			Email:    types.StringValue(attrString(u.Attributes, "email")),
			Active:   types.BoolValue(attrBool(u.Attributes, "active")),
		})
	}

	id := groupID + "/" + ssoID
	state := SSOConnectionUsersDataSourceModel{
		GroupID: config.GroupID,
		SSOID:   config.SSOID,
		ID:      types.StringValue(id),
		Users:   userRefs,
	}
	tflog.Debug(ctx, "Listed SSO connection users", map[string]any{
		"group_id": groupID,
		"sso_id":   ssoID,
		"count":    len(userRefs),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// attrString returns the string value for key from a JSON:API attributes map, or "" if missing.
func attrString(attrs map[string]any, key string) string {
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

// attrBool returns the bool value for key from a JSON:API attributes map, or false if missing.
func attrBool(attrs map[string]any, key string) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs[key]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
