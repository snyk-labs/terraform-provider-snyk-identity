package resources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ resource.Resource = &OrgMembershipResource{}
var _ resource.ResourceWithImportState = &OrgMembershipResource{}

type OrgMembershipResource struct {
	api *client.Client
}

func NewOrgMembershipResource() resource.Resource {
	return &OrgMembershipResource{}
}

func (r *OrgMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_membership"
}

func (r *OrgMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a Snyk organization membership for a user with a role. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/orgs#post-orgs-org_id-memberships) POST /orgs/{org_id}/memberships.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the org membership (membership_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk organization UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user UUID to add as a member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The role UUID to assign (from tenant roles or org default roles). Can be updated in place via PATCH.",
			},
		},
	}
}

type OrgMembershipResourceModel struct {
	ID     types.String `tfsdk:"id"`
	OrgID  types.String `tfsdk:"org_id"`
	UserID types.String `tfsdk:"user_id"`
	RoleID types.String `tfsdk:"role_id"`
}

func (r *OrgMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("invalid provider data", "expected *client.Client")
		return
	}
	r.api = api
}

func (r *OrgMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	membershipID, err := r.api.CreateOrgMembership(
		plan.OrgID.ValueString(),
		plan.UserID.ValueString(),
		plan.RoleID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("create org membership failed", err.Error())
		return
	}

	plan.ID = types.StringValue(membershipID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *OrgMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Snyk REST API does not expose GET /orgs/{org_id}/memberships/{membership_id};
	// we only have list. We keep the state as-is and assume membership still exists.
	// If the user deleted it out-of-band, the next apply would fail on update or
	// we could add a list+filter read. For simplicity we do not re-read.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *OrgMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only role_id can be updated in place via PATCH; org_id and user_id still require replace.
	if err := r.api.UpdateOrgMembership(plan.OrgID.ValueString(), plan.ID.ValueString(), plan.RoleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("update org membership failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *OrgMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.api.DeleteOrgMembership(state.OrgID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete org membership failed", err.Error())
		return
	}
}

func (r *OrgMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "org_id/membership_id" (e.g. "b667f176-df52-4b0a-9954-117af6b05ab7/550e8400-e29b-41d4-a716-446655440000")
	orgID, membershipID, ok := parseOrgMembershipImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("invalid import id", "use format: org_id/membership_id")
		return
	}

	m, err := r.api.GetOrgMembershipByID(orgID, membershipID)
	if err != nil {
		resp.Diagnostics.AddError("read membership for import", err.Error())
		return
	}
	if m == nil {
		resp.Diagnostics.AddError("membership not found", "no membership found with the given org_id and membership id")
		return
	}

	userID, roleID := extractUserAndRoleFromMembership(m)
	if userID == "" || roleID == "" {
		resp.Diagnostics.AddError("import incomplete", "could not determine user_id or role_id from API response; create the resource manually instead")
		return
	}

	state := OrgMembershipResourceModel{
		ID:     types.StringValue(membershipID),
		OrgID:  types.StringValue(orgID),
		UserID: types.StringValue(userID),
		RoleID: types.StringValue(roleID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// parseOrgMembershipImportID returns orgID, membershipID, true if id is "org_id/membership_id".
func parseOrgMembershipImportID(id string) (orgID, membershipID string, ok bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// extractUserAndRoleFromMembership returns user_id and role_id from a list membership item.
func extractUserAndRoleFromMembership(m *client.ListOrgMembershipItem) (userID, roleID string) {
	if m == nil {
		return "", ""
	}
	return m.Relationships.User.Data.ID, m.Relationships.Role.Data.ID
}
