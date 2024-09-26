package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uyuni-project/uyuni-tools/shared/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &UsersDataSource{}
	_ datasource.DataSourceWithConfigure = &UsersDataSource{}
)

// UsersDataSourceModel maps the data source schema data.
type UsersDataSourceModel struct {
	Users []userModel `tfsdk:"user"`
}

// userModel maps user schema data.
type userModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Login types.String `tfsdk:"login"`
}

type user_api struct {
	Id       int
	Login    string
	Login_UC string
	Enabled  bool
}

// NewUsersDataSource is a helper function to simplify the provider implementation.
func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// UsersDataSource is the data source implementation.
type UsersDataSource struct {
	client *api.HTTPClient
}

// Metadata returns the data source type name.
func (d *UsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *UsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"login": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UsersDataSourceModel

	// read users from API
	users, err := api.Get[[]user_api](d.client, "user/listUsers")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Uyuni user",
			err.Error(),
		)
		return
	}

	// Map response body to model
	for _, this_user := range users.Result {
		userState := userModel{
			ID:    types.Int64Value(int64(this_user.Id)),
			Login: types.StringValue(this_user.Login),
		}

		state.Users = append(state.Users, userState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *UsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
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
