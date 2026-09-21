package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &tenantSettingsResource{}
	_ resource.ResourceWithConfigure   = &tenantSettingsResource{}
	_ resource.ResourceWithImportState = &tenantSettingsResource{}
)

func NewTenantSettingsResource() resource.Resource {
	return &tenantSettingsResource{}
}

type tenantSettingsResource struct {
	client *authsignal.Client
}

type tenantSettingsResourceModel struct {
	TokenDurationInMinutes           types.Int64  `tfsdk:"token_duration_in_minutes"`
	AuthenticatorEventsWebhookConfig types.Object `tfsdk:"authenticator_events_webhook_config"`
	LogEventsWebhookConfig           types.Object `tfsdk:"log_events_webhook_config"`
	IpWhitelist                      types.Set    `tfsdk:"ip_whitelist"`
}

type authenticatorEventsWebhookConfigModel struct {
	Url                        types.String `tfsdk:"url"`
	IncludeCredentialPublicKey types.Bool   `tfsdk:"include_credential_public_key"`
}

type logEventsWebhookConfigModel struct {
	EndpointUrl types.String `tfsdk:"endpoint_url"`
}

var authenticatorEventsWebhookConfigAttributeTypes = map[string]attr.Type{
	"url":                           types.StringType,
	"include_credential_public_key": types.BoolType,
}

var logEventsWebhookConfigAttributeTypes = map[string]attr.Type{
	"endpoint_url": types.StringType,
}

var ipv4CidrPattern = regexp.MustCompile(
	`^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])/([0-9]|[12][0-9]|3[0-2])$`,
)

var _ validator.String = ipv4CidrValidator{}

type ipv4CidrValidator struct{}

func (v ipv4CidrValidator) Description(_ context.Context) string {
	return "must be an IPv4 range in CIDR notation"
}

func (v ipv4CidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipv4CidrValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnownAndSet(req.ConfigValue) {
		return
	}

	if ipv4CidrPattern.MatchString(req.ConfigValue.ValueString()) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid IP Range",
		fmt.Sprintf("Expected an IPv4 range in CIDR notation such as \"203.0.113.0/24\", got %q.", req.ConfigValue.ValueString()),
	)
}

func (r *tenantSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_settings"
}

func (r *tenantSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages webhook, session, and network settings for the existing tenant.",
		Attributes: map[string]schema.Attribute{
			"token_duration_in_minutes": schema.Int64Attribute{
				Description: "Challenge token lifetime in minutes. Also sets the pre-built UI session length.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"authenticator_events_webhook_config": schema.SingleNestedAttribute{
				Description: "Authenticator events webhook.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: "HTTPS endpoint for authenticator events.",
						Required:    true,
						Validators: []validator.String{
							httpsUrlValidator{},
						},
					},
					"include_credential_public_key": schema.BoolAttribute{
						Description: "Whether passkey `authenticator.created` events include the credential public key.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"log_events_webhook_config": schema.SingleNestedAttribute{
				Description: "Log events webhook.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"endpoint_url": schema.StringAttribute{
						Description: "HTTPS endpoint for batched log events.",
						Required:    true,
						Validators: []validator.String{
							httpsUrlValidator{},
						},
					},
				},
			},
			"ip_whitelist": schema.SetAttribute{
				Description: "IPv4 ranges in CIDR notation allowed to call the Server API. Set to `[]` to allow every address.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(ipv4CidrValidator{}),
				},
			},
		},
	}
}

func tenantSettingsRequestFrom(ctx context.Context, model tenantSettingsResourceModel) (authsignal.TenantSettings, diag.Diagnostics) {
	var settings authsignal.TenantSettings
	var diags diag.Diagnostics

	if isKnownAndSet(model.TokenDurationInMinutes) {
		settings.TokenDurationInMinutes = authsignal.SetValue(model.TokenDurationInMinutes.ValueInt64())
	}

	if isKnownAndSet(model.AuthenticatorEventsWebhookConfig) {
		var config authenticatorEventsWebhookConfigModel
		diags.Append(model.AuthenticatorEventsWebhookConfig.As(ctx, &config, basetypes.ObjectAsOptions{})...)

		webhook := authsignal.AuthenticatorEventsWebhookConfig{}

		if isKnownAndSet(config.Url) {
			webhook.Url = config.Url.ValueStringPointer()
		}

		if isKnownAndSet(config.IncludeCredentialPublicKey) {
			webhook.IncludeCredentialPublicKey = config.IncludeCredentialPublicKey.ValueBoolPointer()
		}

		settings.AuthenticatorEventsWebhookConfig = authsignal.SetValue(webhook)
	}

	if isKnownAndSet(model.LogEventsWebhookConfig) {
		var config logEventsWebhookConfigModel
		diags.Append(model.LogEventsWebhookConfig.As(ctx, &config, basetypes.ObjectAsOptions{})...)

		webhook := authsignal.LogEventsWebhookConfig{}

		if isKnownAndSet(config.EndpointUrl) {
			webhook.EndpointUrl = config.EndpointUrl.ValueStringPointer()
		}

		settings.LogEventsWebhookConfig = authsignal.SetValue(webhook)
	}

	if isKnownAndSet(model.IpWhitelist) {
		addresses, setDiags := sortedStringsFrom(ctx, model.IpWhitelist)
		diags.Append(setDiags...)
		settings.IpWhitelist = authsignal.SetList(addresses)
	}

	return settings, diags
}

func authenticatorEventsWebhookConfigValue(response *authsignal.AuthenticatorEventsWebhookConfigResponse) types.Object {
	if response == nil {
		return types.ObjectNull(authenticatorEventsWebhookConfigAttributeTypes)
	}

	return types.ObjectValueMust(authenticatorEventsWebhookConfigAttributeTypes, map[string]attr.Value{
		"url":                           types.StringPointerValue(response.Url),
		"include_credential_public_key": types.BoolPointerValue(response.IncludeCredentialPublicKey),
	})
}

func logEventsWebhookConfigValue(response *authsignal.LogEventsWebhookConfigResponse) types.Object {
	if response == nil {
		return types.ObjectNull(logEventsWebhookConfigAttributeTypes)
	}

	return types.ObjectValueMust(logEventsWebhookConfigAttributeTypes, map[string]attr.Value{
		"endpoint_url": types.StringPointerValue(response.EndpointUrl),
	})
}

func tenantSettingsModelFromResponse(tenant *authsignal.TenantResponse) tenantSettingsResourceModel {
	return tenantSettingsResourceModel{
		TokenDurationInMinutes:           types.Int64PointerValue(tenant.TokenDurationInMinutes),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhookConfigValue(tenant.AuthenticatorEventsWebhookConfig),
		LogEventsWebhookConfig:           logEventsWebhookConfigValue(tenant.LogEventsWebhookConfig),
		IpWhitelist:                      stringSetPointerValue(tenant.IpWhitelist),
	}
}

func (r *tenantSettingsResource) write(ctx context.Context, config tenantSettingsResourceModel) (*authsignal.TenantResponse, diag.Diagnostics) {
	settings, diags := tenantSettingsRequestFrom(ctx, config)
	if diags.HasError() {
		return nil, diags
	}

	tenant, _, err := r.client.UpdateTenant(settings)
	if err != nil {
		diags.AddError(
			"Error updating tenant settings",
			"Could not update tenant settings, unexpected error: "+err.Error(),
		)
		return nil, diags
	}

	return tenant, diags
}

func (r *tenantSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config tenantSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, diags := r.write(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenantSettingsModelFromResponse(tenant))...)
}

func (r *tenantSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tenant, statusCode, err := r.client.GetTenant()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Tenant Settings",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenantSettingsModelFromResponse(tenant))...)
}

func (r *tenantSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config tenantSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, diags := r.write(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenantSettingsModelFromResponse(tenant))...)
}

func (r *tenantSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *tenantSettingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The provider is configured against a single tenant, so the import ID is not needed.
	// Seed typed empty state; the subsequent Read populates it from the API.
	resp.Diagnostics.Append(resp.State.Set(ctx, emptyTenantSettingsModel())...)
}

// emptyTenantSettingsModel returns a model whose nested attributes carry their attribute types.
// A zero-value model stores untyped null objects, which Read cannot convert back into the schema.
func emptyTenantSettingsModel() tenantSettingsResourceModel {
	return tenantSettingsResourceModel{
		TokenDurationInMinutes:           types.Int64Null(),
		AuthenticatorEventsWebhookConfig: types.ObjectNull(authenticatorEventsWebhookConfigAttributeTypes),
		LogEventsWebhookConfig:           types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:                      types.SetNull(types.StringType),
	}
}

func (r *tenantSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*authsignal.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *authsignal.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
