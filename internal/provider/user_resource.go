// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/uyuni-project/uyuni-tools/shared/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// userResource is the resource implementation.
type userResource struct {
	client *api.HTTPClient
}

// userResourceModel maps the resource schema data.
type userResourceModel struct {
	Login     types.String `tfsdk:"login"`
	Password  types.String `tfsdk:"password"`
	FirstName types.String `tfsdk:"firstname"`
	LastName  types.String `tfsdk:"lastname"`
	Email     types.String `tfsdk:"email"`
	Roles     types.Set    `tfsdk:"roles"`
}

// userAPIModel represents the user data structure returned by the Uyuni API.
type userAPIModel struct {
	First_name          string `json:"first_name"`
	Last_name           string `json:"last_name"`
	Email               string `json:"email"`
	Org_id              int    `json:"org_id"`
	Org_name            string `json:"org_name"`
	Prefix              string `json:"prefix"`
	Last_login_date     string `json:"last_login_date"`
	Created_date        string `json:"created_date"`
	Enabled             bool   `json:"enabled"`
	Use_pam             bool   `json:"use_pam"`
	Read_only           bool   `json:"read_only"`
	Errata_notification bool   `json:"errata_notification"`
}

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// Metadata returns the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the resource.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a user in Uyuni, including their roles and permissions.",
		Attributes: map[string]schema.Attribute{
			"login": schema.StringAttribute{
				Required:    true,
				Description: "User's login name (must be unique).",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "User's password.",
			},
			"firstname": schema.StringAttribute{
				Required:    true,
				Description: "User's first name.",
			},
			"lastname": schema.StringAttribute{
				Required:    true,
				Description: "User's last name.",
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "User's email address.",
			},
			"roles": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Set of roles assigned to the user. Common roles include: org_admin, system_group_admin, channel_admin, config_admin.",
			},
		},
	}
}

// Create creates a new user resource.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new user
	userData := map[string]interface{}{
		"login":     plan.Login.ValueString(),
		"password":  plan.Password.ValueString(),
		"firstName": plan.FirstName.ValueString(),
		"lastName":  plan.LastName.ValueString(),
		"email":     plan.Email.ValueString(),
	}

	tflog.Info(ctx, "Creating user", map[string]any{
		"login": plan.Login.ValueString(),
		"email": plan.Email.ValueString(),
	})

	_, err := api.Post[int](r.client, "user/create", userData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			fmt.Sprintf("Could not create user %s: %s", plan.Login.ValueString(), err.Error()),
		)
		return
	}

	tflog.Info(ctx, "User created successfully", map[string]any{
		"login": plan.Login.ValueString(),
	})

	// Add roles if specified
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string
		diags = plan.Roles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, role := range roles {
			if err := r.addRoleToUser(ctx, plan.Login.ValueString(), role); err != nil {
				resp.Diagnostics.AddError(
					"Error adding role to user",
					fmt.Sprintf("Could not add role %s to user %s: %s", role, plan.Login.ValueString(), err.Error()),
				)
				return
			}
			tflog.Debug(ctx, "Added role to user", map[string]any{
				"login": plan.Login.ValueString(),
				"role":  role,
			})
		}
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	login := state.Login.ValueString()
	tflog.Debug(ctx, "Reading user", map[string]any{"login": login})

	// Get user details from Uyuni API
	user, err := api.Get[userAPIModel](r.client, "user/getDetails?login="+login)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			fmt.Sprintf("Could not read user %s: %s", login, err.Error()),
		)
		return
	}

	// Update state with user details
	state.FirstName = types.StringValue(user.Result.First_name)
	state.LastName = types.StringValue(user.Result.Last_name)
	state.Email = types.StringValue(user.Result.Email)

	// Get user roles
	userRoles, err := api.Get[[]string](r.client, "user/listRoles?login="+login)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user roles",
			fmt.Sprintf("Could not read roles for user %s: %s", login, err.Error()),
		)
		return
	}

	// Convert roles to types.Set
	if len(userRoles.Result) > 0 {
		rolesSet, diagsSet := types.SetValueFrom(ctx, types.StringType, userRoles.Result)
		diags = append(diags, diagsSet...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		state.Roles = rolesSet
	} else {
		emptySet, diagsSet := types.SetValueFrom(ctx, types.StringType, []string{})
		diags = append(diags, diagsSet...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		state.Roles = emptySet
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	login := plan.Login.ValueString()
	tflog.Debug(ctx, "Updating user", map[string]any{"login": login})

	// Update user details if they changed
	if !plan.FirstName.Equal(state.FirstName) ||
		!plan.LastName.Equal(state.LastName) ||
		!plan.Email.Equal(state.Email) ||
		!plan.Password.Equal(state.Password) {

		updateData := map[string]interface{}{
			"login": login,
			"details": map[string]interface{}{
				"first_name": plan.FirstName.ValueString(),
				"last_name":  plan.LastName.ValueString(),
				"email":      plan.Email.ValueString(),
				"password":   plan.Password.ValueString(),
			},
		}

		_, err := api.Post[int](r.client, "user/setDetails", updateData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating user details",
				fmt.Sprintf("Could not update user %s: %s", login, err.Error()),
			)
			return
		}
		tflog.Info(ctx, "User details updated", map[string]any{"login": login})
	}

	// Handle role changes
	if !plan.Roles.Equal(state.Roles) {
		if err := r.updateUserRoles(ctx, login, state.Roles, plan.Roles); err != nil {
			resp.Diagnostics.AddError(
				"Error updating user roles",
				fmt.Sprintf("Could not update roles for user %s: %s", login, err.Error()),
			)
			return
		}
	}

	// Set updated state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	login := state.Login.ValueString()
	tflog.Info(ctx, "Deleting user", map[string]any{"login": login})

	_, err := api.Post[int](r.client, "user/delete?login="+login, map[string]interface{}{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			fmt.Sprintf("Could not delete user %s: %s", login, err.Error()),
		)
		return
	}

	tflog.Info(ctx, "User deleted successfully", map[string]any{"login": login})
}

// ImportState imports an existing user by login name.
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	login := req.ID

	if login == "" {
		resp.Diagnostics.AddError(
			"Import ID Required",
			"Please provide a login name to import the user. Example: terraform import uyuni_user.example username",
		)
		return
	}

	tflog.Info(ctx, "Importing user", map[string]any{"login": login})

	// Get user details
	user, err := api.Get[userAPIModel](r.client, "user/getDetails?login="+login)
	if err != nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("Could not find user %s for import: %s", login, err.Error()),
		)
		return
	}

	// Get user roles
	userRoles, err := api.Get[[]string](r.client, "user/listRoles?login="+login)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User Roles",
			fmt.Sprintf("Could not read roles for user %s during import: %s", login, err.Error()),
		)
		return
	}

	// Create state model
	state := userResourceModel{
		Login:     types.StringValue(login),
		FirstName: types.StringValue(user.Result.First_name),
		LastName:  types.StringValue(user.Result.Last_name),
		Email:     types.StringValue(user.Result.Email),
		Password:  types.StringValue("IMPORTED_PASSWORD_CHANGE_ME"),
	}

	// Convert roles to types.Set
	if len(userRoles.Result) > 0 {
		rolesSet, diags := types.SetValueFrom(ctx, types.StringType, userRoles.Result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Roles = rolesSet
	} else {
		emptySet, diags := types.SetValueFrom(ctx, types.StringType, []string{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Roles = emptySet
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Successfully imported user", map[string]any{
		"login":      login,
		"role_count": len(userRoles.Result),
	})
}

// Configure adds the provider configured client to the resource.
func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.HTTPClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.HTTPClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Helper functions

// addRoleToUser adds a role to a user.
func (r *userResource) addRoleToUser(ctx context.Context, login, role string) error {
	roleData := map[string]interface{}{
		"login": login,
		"role":  role,
	}

	_, err := api.Post[int](r.client, "user/addRole", roleData)
	return err
}

// removeRoleFromUser removes a role from a user.
func (r *userResource) removeRoleFromUser(ctx context.Context, login, role string) error {
	roleData := map[string]interface{}{
		"login": login,
		"role":  role,
	}

	_, err := api.Post[int](r.client, "user/removeRole", roleData)
	return err
}

// updateUserRoles handles adding and removing roles based on the difference between current and planned roles.
func (r *userResource) updateUserRoles(ctx context.Context, login string, currentRoles, plannedRoles types.Set) error {
	var currentRolesList []string
	var plannedRolesList []string

	if !currentRoles.IsNull() && !currentRoles.IsUnknown() {
		diags := currentRoles.ElementsAs(ctx, &currentRolesList, false)
		if diags.HasError() {
			return fmt.Errorf("failed to convert current roles")
		}
	}

	if !plannedRoles.IsNull() && !plannedRoles.IsUnknown() {
		diags := plannedRoles.ElementsAs(ctx, &plannedRolesList, false)
		if diags.HasError() {
			return fmt.Errorf("failed to convert planned roles")
		}
	}

	// Remove roles that are no longer wanted
	for _, currentRole := range currentRolesList {
		found := false
		for _, plannedRole := range plannedRolesList {
			if currentRole == plannedRole {
				found = true
				break
			}
		}
		if !found {
			if err := r.removeRoleFromUser(ctx, login, currentRole); err != nil {
				return fmt.Errorf("failed to remove role %s: %w", currentRole, err)
			}
			tflog.Debug(ctx, "Removed role from user", map[string]any{
				"login": login,
				"role":  currentRole,
			})
		}
	}

	// Add new roles
	for _, plannedRole := range plannedRolesList {
		found := false
		for _, currentRole := range currentRolesList {
			if plannedRole == currentRole {
				found = true
				break
			}
		}
		if !found {
			if err := r.addRoleToUser(ctx, login, plannedRole); err != nil {
				return fmt.Errorf("failed to add role %s: %w", plannedRole, err)
			}
			tflog.Debug(ctx, "Added role to user", map[string]any{
				"login": login,
				"role":  plannedRole,
			})
		}
	}

	return nil
}
