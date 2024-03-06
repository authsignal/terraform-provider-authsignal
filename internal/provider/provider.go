package provider

import (
	"context"
	"os"

	"authsignal.com/authsignal-management-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &authsignalProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &authsignalProvider{
			version: version,
		}
	}
}

// authsignalProvider is the provider implementation.
type authsignalProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type authsignalProviderModel struct {
	Host      types.String `tfsdk:"host"`
	TenantID  types.String `tfsdk:"tenant_id"`
	ApiSecret types.String `tfsdk:"api_secret"`
}

// Metadata returns the provider type name.
func (p *authsignalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "authsignal"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *authsignalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"tenant_id": schema.StringAttribute{
				Optional: true,
			},
			"api_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a Authsignal API client for data sources and resources.
func (p *authsignalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config authsignalProviderModel
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
			"Unknown Authsignal API Host",
			"The provider cannot create the Authsignal API client as there is an unknown configuration value for the Authsignal API Host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the AUTHSIGNAL_HOST environment variable.",
		)
	}

	if config.TenantID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant_id"),
			"Unknown Authsignal API Tenant ID",
			"The provider cannot create the Authsignal API client as there is an unknown configuration value for the Authsignal API Tenant ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the AUTHSIGNAL_TENANT_ID environment variable.",
		)
	}

	if config.ApiSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_secret"),
			"Unknown Authsignal API ApiSecret",
			"The provider cannot create the Authsignal API client as there is an unknown configuration value for the Authsignal API Secret. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the AUTHSIGNAL_API_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("AUTHSIGNAL_HOST")
	tenant_id := os.Getenv("AUTHSIGNAL_TENANT_ID")
	api_secret := os.Getenv("AUTHSIGNAL_API_SECRET")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.TenantID.IsNull() {
		tenant_id = config.TenantID.ValueString()
	}

	if !config.ApiSecret.IsNull() {
		api_secret = config.ApiSecret.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Authsignal API Host",
			"The provider cannot create the Authsignal API client as there is a missing or empty value for the Authsignal API host. "+
				"Set the host value in the configuration or use the AUTHSIGNAL_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if tenant_id == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("tenant_id"),
			"Missing Authsignal API Tenant ID",
			"The provider cannot create the Authsignal API client as there is a missing or empty value for the Authsignal API Tenant ID. "+
				"Set the tenant_id value in the configuration or use the AUTHSIGNAL_TENANT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if api_secret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_secret"),
			"Missing Authsignal API Secret",
			"The provider cannot create the Authsignal API client as there is a missing or empty value for the Authsignal API Secret. "+
				"Set the api_secret value in the configuration or use the AUTHSIGNAL_API_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "authsignal_host", host)
	ctx = tflog.SetField(ctx, "authsignal_tenant_id", tenant_id)

	// Create a new Authsignal client using the configuration values
	client := authsignal.NewClient(host, tenant_id, api_secret)

	// Make the Authsignal client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = &client
	resp.ResourceData = &client

	tflog.Info(ctx, "Configured Authsignal client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *authsignalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewActionConfigurationDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *authsignalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewActionConfigurationResource,
	}
}
