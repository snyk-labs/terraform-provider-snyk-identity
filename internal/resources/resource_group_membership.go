package resources

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

type GroupMembershipResource struct {
	api *client.Client
}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

func (r *GroupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a Snyk group membership for a user with a role. Uses the [Snyk REST API](https://docs.snyk.io/snyk-api/reference/groups#post-groups-group_id-memberships) POST /groups/{group_id}/memberships.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the group membership (membership_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Snyk group UUID.",
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
				MarkdownDescription: "The role UUID to assign (from tenant roles). Can be updated in place via PATCH.",
			},
			"cascade_delete": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When true, deleting this membership also deletes the user's org memberships within the group.",
				PlanModifiers:       []planmodifier.Bool{
					// not RequiresReplace; can change without recreating membership
				},
			},
		},
	}
}

type GroupMembershipResourceModel struct {
	ID            types.String `tfsdk:"id"`
	GroupID       types.String `tfsdk:"group_id"`
	UserID        types.String `tfsdk:"user_id"`
	RoleID        types.String `tfsdk:"role_id"`
	CascadeDelete types.Bool   `tfsdk:"cascade_delete"`
}

func (r *GroupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating group membership", map[string]any{
		"group_id": plan.GroupID.ValueString(),
		"user_id":  plan.UserID.ValueString(),
	})
	membershipID, err := r.api.CreateGroupMembership(
		ctx,
		plan.GroupID.ValueString(),
		plan.UserID.ValueString(),
		plan.RoleID.ValueString(),
	)
	if err != nil {
		tflog.Error(ctx, "Create group membership failed", map[string]any{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("create group membership failed", err.Error())
		return
	}

	tflog.Debug(ctx, "Created group membership", map[string]any{
		"group_id":      plan.GroupID.ValueString(),
		"membership_id": membershipID,
	})
	plan.ID = types.StringValue(membershipID)
	cascade := false
	if !plan.CascadeDelete.IsNull() {
		cascade = plan.CascadeDelete.ValueBool()
	}
	plan.CascadeDelete = types.BoolValue(cascade)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading group membership (no remote refresh)", map[string]any{
		"group_id":      state.GroupID.ValueString(),
		"membership_id": state.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Updating group membership role", map[string]any{
		"group_id":      plan.GroupID.ValueString(),
		"membership_id": plan.ID.ValueString(),
	})
	// role_id can be updated in place via PATCH; cascade_delete is state-only; group_id and user_id still require replace.
	if err := r.api.UpdateGroupMembership(ctx, plan.GroupID.ValueString(), plan.ID.ValueString(), plan.RoleID.ValueString()); err != nil {
		tflog.Error(ctx, "Update group membership failed", map[string]any{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("update group membership failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cascade := state.CascadeDelete.ValueBool()
	tflog.Debug(ctx, "Deleting group membership", map[string]any{
		"group_id":       state.GroupID.ValueString(),
		"membership_id":  state.ID.ValueString(),
		"cascade_delete": cascade,
	})
	if err := r.api.DeleteGroupMembership(ctx, state.GroupID.ValueString(), state.ID.ValueString(), cascade); err != nil {
		tflog.Error(ctx, "Delete group membership failed", map[string]any{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("delete group membership failed", err.Error())
		return
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupID, membershipID, ok := parseGroupMembershipImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("invalid import id", "use format: group_id/membership_id")
		return
	}

	tflog.Debug(ctx, "Importing group membership", map[string]any{
		"group_id":      groupID,
		"membership_id": membershipID,
	})
	m, err := r.api.GetGroupMembershipByID(ctx, groupID, membershipID)
	if err != nil {
		tflog.Error(ctx, "Import group membership read failed", map[string]any{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("read membership for import", err.Error())
		return
	}
	if m == nil {
		resp.Diagnostics.AddError("membership not found", "no membership found with the given group_id and membership id")
		return
	}

	userID, roleID := extractUserAndRoleFromGroupMembership(m)
	if userID == "" || roleID == "" {
		resp.Diagnostics.AddError("import incomplete", "could not determine user_id or role_id from API response; create the resource manually instead")
		return
	}

	state := GroupMembershipResourceModel{
		ID:            types.StringValue(membershipID),
		GroupID:       types.StringValue(groupID),
		UserID:        types.StringValue(userID),
		RoleID:        types.StringValue(roleID),
		CascadeDelete: types.BoolValue(false),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func parseGroupMembershipImportID(id string) (groupID, membershipID string, ok bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func extractUserAndRoleFromGroupMembership(m *client.ListGroupMembershipItem) (userID, roleID string) {
	if m == nil {
		return "", ""
	}
	return m.Relationships.User.Data.ID, m.Relationships.Role.Data.ID
}
