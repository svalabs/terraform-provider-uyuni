// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/uyuni-project/uyuni-tools/shared/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

// usersDataSource is the data source implementation.
type usersDataSource struct {
	client *api.HTTPClient
}

// usersDataSourceModel maps the data source schema data.
type usersDataSourceModel struct {
	ID    types.String    `tfsdk:"id"`
	Users []userDataModel `tfsdk:"users"`
}

// userDataModel represents a user in the data source.
type userDataModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Login   types.String `tfsdk:"login"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

// userListAPIModel represents the user data structure returned by the Uyuni API for user lists.
type userListAPIModel struct {
	Id       int    `json:"id"`
	Login    string `json:"login"`
	Login_UC string `json:"login_uc"`
	Enabled  bool   `json:"enabled"`
}

// NewUsersDataSource is a helper function to simplify the provider implementation.
func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

// Metadata returns the data source type name.
func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of all users in the Uyuni organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Data source identifier.",
			},
			"users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of users in the organization.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:    true,
							Description: "User ID.",
						},
						"login": schema.StringAttribute{
							Computed:    true,
							Description: "User login name.",
						},
						"enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the user is enabled.",
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state usersDataSourceModel

	tflog.Debug(ctx, "Reading users from Uyuni API")

	// Get users from Uyuni API
	usersResp, err := api.Get[[]userListAPIModel](d.client, "user/listUsers")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Uyuni Users",
			fmt.Sprintf("Could not read users from Uyuni API: %s", err.Error()),
		)
		return
	}

	// Map response to model
	state.ID = types.StringValue("users")
	state.Users = make([]userDataModel, len(usersResp.Result))

	for i, user := range usersResp.Result {
		state.Users[i] = userDataModel{
			ID:      types.Int64Value(int64(user.Id)),
			Login:   types.StringValue(user.Login),
			Enabled: types.BoolValue(user.Enabled),
		}
	}

	tflog.Debug(ctx, "Successfully read users", map[string]any{
		"user_count": len(state.Users),
	})

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Configure adds the provider configured client to the data source.
func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.HTTPClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.HTTPClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}
