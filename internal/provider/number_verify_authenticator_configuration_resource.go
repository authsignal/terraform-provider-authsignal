package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &numberVerifyAuthenticatorConfigurationResource{}
	_ resource.ResourceWithConfigure   = &numberVerifyAuthenticatorConfigurationResource{}
	_ resource.ResourceWithImportState = &numberVerifyAuthenticatorConfigurationResource{}
)

func NewNumberVerifyAuthenticatorConfigurationResource() resource.Resource {
	return &numberVerifyAuthenticatorConfigurationResource{}
}

type numberVerifyAuthenticatorConfigurationResource struct {
	client *authsignal.Client
}

type numberVerifyAuthenticatorConfigurationResourceModel struct {
	AuthenticatorId    types.String `tfsdk:"authenticator_id"`
	VerificationMethod types.String `tfsdk:"verification_method"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	AppId              types.String `tfsdk:"app_id"`
	Environment        types.String `tfsdk:"environment"`
	Region             types.String `tfsdk:"region"`
}

func (r *numberVerifyAuthenticatorConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_number_verify_authenticator_configuration"
}

func (r *numberVerifyAuthenticatorConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the tenant's Number Verify authenticator configuration, which verifies a phone number with the user's mobile carrier. Destroying the resource deletes the configuration.",
		Attributes: map[string]schema.Attribute{
			"authenticator_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verification_method": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"app_id": schema.StringAttribute{
				Description: "The app registration ID assigned when your app was registered with the carrier network.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"environment": schema.StringAttribute{
				Description: "Whether to verify against live carrier networks or simulated carrier responses. Defaults to `PRODUCTION`." + allowedValues(allowedNumberVerifyEnvironments),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedNumberVerifyEnvironments...),
				},
			},
			"region": schema.StringAttribute{
				Description: "The carrier network region used in production." + allowedValues(allowedNumberVerifyRegions),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedNumberVerifyRegions...),
				},
			},
		},
	}
}

func (r *numberVerifyAuthenticatorConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config numberVerifyAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.CreateNumberVerifyAuthenticatorConfiguration(numberVerifyCreateBody(plan, config))

	if statusCode == 409 {
		resp.Diagnostics.Append(createConflictDiagnostic("authsignal_number_verify_authenticator_configuration", "Number Verify", numberVerifyAuthenticatorSlug))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Number Verify authenticator configuration",
			"Could not create the Number Verify authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(numberVerifyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, numberVerifyStateAfterWrite(plan, response))...)
}

func (r *numberVerifyAuthenticatorConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	response, statusCode, err := r.client.GetNumberVerifyAuthenticatorConfiguration()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Number Verify Authenticator Configuration",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(numberVerifyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, numberVerifyStateFromResponse(response))...)
}

func (r *numberVerifyAuthenticatorConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state numberVerifyAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, _, err := r.client.UpdateNumberVerifyAuthenticatorConfiguration(numberVerifyUpdateBody(plan, config, state))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Number Verify authenticator configuration",
			"Could not update the Number Verify authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(numberVerifyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, numberVerifyStateAfterWrite(plan, response))...)
}

func (r *numberVerifyAuthenticatorConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, statusCode, err := r.client.DeleteNumberVerifyAuthenticatorConfiguration()

	if statusCode == 404 {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Number Verify authenticator configuration",
			"Could not delete the Number Verify authenticator configuration, unexpected error: "+err.Error(),
		)
	}
}

func (r *numberVerifyAuthenticatorConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(importStateSlug(req.ID, numberVerifyAuthenticatorSlug)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, numberVerifyAuthenticatorConfigurationResourceModel{})...)
}

func (r *numberVerifyAuthenticatorConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*authsignal.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *authsignal.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func numberVerifyCreateBody(plan numberVerifyAuthenticatorConfigurationResourceModel, config numberVerifyAuthenticatorConfigurationResourceModel) authsignal.CreateNumberVerifyAuthenticatorConfigurationBody {
	return authsignal.CreateNumberVerifyAuthenticatorConfigurationBody{
		IsActive:    boolPointer(config.IsActive),
		AppId:       plan.AppId.ValueString(),
		Environment: stringPointer(config.Environment),
		Region:      stringPointer(config.Region),
	}
}

func numberVerifyUpdateBody(plan numberVerifyAuthenticatorConfigurationResourceModel, config numberVerifyAuthenticatorConfigurationResourceModel, state numberVerifyAuthenticatorConfigurationResourceModel) authsignal.UpdateNumberVerifyAuthenticatorConfigurationBody {
	var body authsignal.UpdateNumberVerifyAuthenticatorConfigurationBody

	if isKnownAndSet(config.IsActive) {
		body.IsActive = authsignal.SetValue(config.IsActive.ValueBool())
	}

	if !plan.AppId.Equal(state.AppId) {
		body.AppId = authsignal.SetValue(plan.AppId.ValueString())
	}

	if isKnownAndSet(config.Environment) {
		body.Environment = authsignal.SetValue(config.Environment.ValueString())
	}

	if isKnownAndSet(config.Region) {
		body.Region = authsignal.SetValue(config.Region.ValueString())
	}

	return body
}

func numberVerifyStateAfterWrite(plan numberVerifyAuthenticatorConfigurationResourceModel, response *authsignal.NumberVerifyAuthenticatorConfiguration) numberVerifyAuthenticatorConfigurationResourceModel {
	state := plan
	state.AuthenticatorId = types.StringValue(response.AuthenticatorId)
	state.VerificationMethod = types.StringValue(response.VerificationMethod)

	state.IsActive = resolvedValue(plan.IsActive, types.BoolValue(response.IsActive))
	state.Environment = resolvedValue(plan.Environment, types.StringPointerValue(response.Environment))
	state.Region = resolvedValue(plan.Region, types.StringPointerValue(response.Region))

	return state
}

func numberVerifyStateFromResponse(response *authsignal.NumberVerifyAuthenticatorConfiguration) numberVerifyAuthenticatorConfigurationResourceModel {
	return numberVerifyAuthenticatorConfigurationResourceModel{
		AuthenticatorId:    types.StringValue(response.AuthenticatorId),
		VerificationMethod: types.StringValue(response.VerificationMethod),
		IsActive:           types.BoolValue(response.IsActive),
		AppId:              types.StringPointerValue(response.AppId),
		Environment:        types.StringPointerValue(response.Environment),
		Region:             types.StringPointerValue(response.Region),
	}
}
