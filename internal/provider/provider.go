package provider

import (
	"context"
	"os"

	"github.com/authsignal/authsignal-management-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &authsignalProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &authsignalProvider{
			version: version,
		}
	}
}

type authsignalProvider struct {
	version string
}

type authsignalProviderModel struct {
	Host      types.String `tfsdk:"host"`
	TenantID  types.String `tfsdk:"tenant_id"`
	ApiSecret types.String `tfsdk:"api_secret"`
}

func (p *authsignalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "authsignal"
	resp.Version = p.version
}

func (p *authsignalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The host URL of the Authsignal Management API for your tenant.",
				Optional:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of your tenant.",
				Optional:    true,
			},
			"api_secret": schema.StringAttribute{
				Description: "The Management API Secret obtained from Authsignal's admin portal.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *authsignalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config authsignalProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	client := authsignal.NewClient(host, tenant_id, api_secret)

	resp.DataSourceData = &client
	resp.ResourceData = &client

	tflog.Info(ctx, "Configured Authsignal client", map[string]any{"success": true})
}

func (p *authsignalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewActionConfigurationDataSource,
		NewRuleDataSource,
	}
}

func (p *authsignalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewActionConfigurationResource,
		NewRuleResource,
	}
}
