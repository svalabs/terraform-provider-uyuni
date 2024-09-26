package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/uyuni-project/uyuni-tools/shared/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &uyuniProvider{}
)

// uyuniProvider is the provider implementation.
type uyuniProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// uyuniProviderModel maps provider schema data to a Go type.
type uyuniProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &uyuniProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *uyuniProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "uyuni"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *uyuniProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a uyuni API client for data sources and resources.
func (p *uyuniProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Uyuni client")

	// Retrieve provider data from configuration
	var config uyuniProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Uyuni API Host",
			"The provider cannot create the Uyuni API client as there is an unknown configuration value for the Uyuni API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the Uyuni_HOST environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Uyuni API Username",
			"The provider cannot create the Uyuni API client as there is an unknown configuration value for the Uyuni API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the Uyuni_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Uyuni API Password",
			"The provider cannot create the Uyuni API client as there is an unknown configuration value for the Uyuni API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the Uyuni_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("UYUNI_HOST")
	username := os.Getenv("UYUNI_USERNAME")
	password := os.Getenv("UYUNI_PASSWORD")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Uyuni API Host",
			"The provider cannot create the Uyuni API client as there is a missing or empty value for the Uyuni API host. "+
				"Set the host value in the configuration or use the Uyuni_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Uyuni API Username",
			"The provider cannot create the Uyuni API client as there is a missing or empty value for the Uyuni API username. "+
				"Set the username value in the configuration or use the Uyuni_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Uyuni API Password",
			"The provider cannot create the Uyuni API client as there is a missing or empty value for the Uyuni API password. "+
				"Set the password value in the configuration or use the Uyuni_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "uyuni_host", host)
	ctx = tflog.SetField(ctx, "uyuni_username", username)
	ctx = tflog.SetField(ctx, "uyuni_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "uyuni_password")

	tflog.Debug(ctx, "Creating HashiCups client")

	// Create a new Uyuni client using the configuration values
	var _conn = api.ConnectionDetails{
		Server:   host,
		User:     username,
		Password: password,
		CAcert:   "",
		Insecure: true,
	}
	client, err := api.Init(&_conn)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Uyuni API Client",
			"An unexpected error occurred when creating the Uyuni API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Uyuni Client Error: "+err.Error(),
		)
		return
	}

	// Make the Uyuni client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Uyuni client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *uyuniProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUsersDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *uyuniProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
	}
}
