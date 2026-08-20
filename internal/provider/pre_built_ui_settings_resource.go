package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &preBuiltUiSettingsResource{}
	_ resource.ResourceWithConfigure   = &preBuiltUiSettingsResource{}
	_ resource.ResourceWithImportState = &preBuiltUiSettingsResource{}
)

func NewPreBuiltUiSettingsResource() resource.Resource {
	return &preBuiltUiSettingsResource{}
}

type preBuiltUiSettingsResource struct {
	client *authsignal.Client
}

type preBuiltUiSettingsResourceModel struct {
	HideSuccessScreenOnEnrollment types.Bool `tfsdk:"hide_success_screen_on_enrollment"`
}

func (r *preBuiltUiSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pre_built_ui_settings"
}

func (r *preBuiltUiSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a tenant's pre-built UI settings. The tenant itself already exists and cannot be created or deleted through this API, so this resource only ever updates settings. Applying it changes the settings configured here and leaves the rest of the tenant untouched. Destroying it stops Terraform managing those settings rather than resetting them. A setting left out of the configuration stays unmanaged, so it can still be set in the admin portal.",
		Attributes: map[string]schema.Attribute{
			"hide_success_screen_on_enrollment": schema.BoolAttribute{
				Description: "Whether the pre-built UI skips the success screen shown after a user enrolls an authenticator.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

// tenantSettingsFromModel builds a partial update from the values the configuration actually sets.
// A null attribute is left out of the request so the tenant keeps whatever it already has.
func tenantSettingsFromModel(model preBuiltUiSettingsResourceModel) authsignal.TenantSettings {
	var settings authsignal.TenantSettings

	if !model.HideSuccessScreenOnEnrollment.IsNull() && !model.HideSuccessScreenOnEnrollment.IsUnknown() {
		settings.HideSuccessScreenOnEnrollment = authsignal.SetValue(model.HideSuccessScreenOnEnrollment.ValueBool())
	}

	return settings
}

func preBuiltUiSettingsModelFromResponse(tenant *authsignal.TenantResponse) preBuiltUiSettingsResourceModel {
	return preBuiltUiSettingsResourceModel{
		HideSuccessScreenOnEnrollment: types.BoolPointerValue(tenant.HideSuccessScreenOnEnrollment),
	}
}

func (r *preBuiltUiSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config preBuiltUiSettingsResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, _, err := r.client.UpdateTenant(tenantSettingsFromModel(config))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating pre-built UI settings",
			"Could not create pre-built UI settings, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, preBuiltUiSettingsModelFromResponse(tenant))
	resp.Diagnostics.Append(diags...)
}

func (r *preBuiltUiSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tenant, statusCode, err := r.client.GetTenant()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Pre-Built UI Settings",
			err.Error(),
		)
		return
	}

	diags := resp.State.Set(ctx, preBuiltUiSettingsModelFromResponse(tenant))
	resp.Diagnostics.Append(diags...)
}

func (r *preBuiltUiSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config preBuiltUiSettingsResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, _, err := r.client.UpdateTenant(tenantSettingsFromModel(config))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating pre-built UI settings",
			"Could not update pre-built UI settings, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, preBuiltUiSettingsModelFromResponse(tenant))
	resp.Diagnostics.Append(diags...)
}

// Delete drops the settings from state without calling the API. These settings live on the tenant,
// which cannot be deleted through the Management API, and resetting them would be a surprising
// thing for a destroy to do.
func (r *preBuiltUiSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *preBuiltUiSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The provider is configured against a single tenant, so there is nothing to key the import on.
	// Seed empty state; the subsequent Read populates it from the API.
	resp.Diagnostics.Append(resp.State.Set(ctx, preBuiltUiSettingsResourceModel{})...)
}

func (r *preBuiltUiSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
