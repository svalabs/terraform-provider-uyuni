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
	_ resource.Resource              = &userResource{}
	_ resource.ResourceWithConfigure = &userResource{}
)

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the resource implementation.
type userResource struct {
	client *api.HTTPClient
}

// userResourceModel maps the resource schema data.
type userResourceModel struct {
	// ID        types.String `tfsdk:"id"`
	Login     types.String `tfsdk:"login"`
	Password  types.String `tfsdk:"password"`
	FirstName types.String `tfsdk:"firstname"`
	LastName  types.String `tfsdk:"lastname"`
	Email     types.String `tfsdk:"email"`
}

// Metadata returns the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the resource.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// "id": schema.StringAttribute{
			// 	Computed: true,
			// },
			"login": schema.StringAttribute{
				Required: true,
			},
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"firstname": schema.StringAttribute{
				Required: true,
			},
			"lastname": schema.StringAttribute{
				Required: true,
			},
			"email": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Create a new resource.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new user
	data := map[string]interface{}{
		"login":     plan.Login.ValueString(),
		"password":  plan.Password.ValueString(),
		"firstName": plan.FirstName.ValueString(),
		"lastName":  plan.LastName.ValueString(),
		"email":     plan.Email.ValueString(),
	}

	tflog.Info(ctx, "About to create user")
	tflog.Info(ctx, ""+plan.Login.String()+" - "+plan.Password.String()+" - "+plan.FirstName.String()+" - "+plan.LastName.String()+" - "+plan.Email.String())

	_, err := api.Post[int](r.client, "user/create", data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			"Could not create user, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "User created")

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	tflog.Info(ctx, fmt.Sprintf("Updated state object be like: %v", resp.State))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed user value from Uyuni
	type user_api struct {
		First_names         string
		First_name          string
		Last_name           string
		Email               string
		Org_id              int
		Org_name            string
		Prefix              string
		Last_login_date     string
		Created_date        string
		Enabled             bool
		Use_pam             bool
		Read_only           bool
		Errata_notification bool
	}
	tflog.Info(ctx, fmt.Sprintf("About to look for user %s", state.Login.ValueString()))
	this_user, err := api.Get[user_api](r.client, "user/getDetails?login="+state.Login.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Uyuuni user",
			"Could not read User "+state.Login.ValueString()+": "+err.Error(),
		)
		return
	}

	state.FirstName = types.StringValue(this_user.Result.First_name)
	state.LastName = types.StringValue(this_user.Result.Last_name)
	state.Email = types.StringValue(this_user.Result.Email)
	tflog.Info(ctx, fmt.Sprintf("Information returned from API: %v", this_user.Result))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing user
	//err := r.client.DeleteOrder(state.ID.ValueString())
	// this_user, err := api.Get[user_api](r.client, "user/getDetails?login="+state.Login.ValueString())
	_, err := api.Post[int](r.client, "user/delete?login="+state.Login.ValueString(), map[string]interface{}{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Uyuni user",
			"Could not delete order, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.HTTPClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *uyuni.client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
