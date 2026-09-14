package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &passkeyAuthenticatorConfigurationResource{}
	_ resource.ResourceWithConfigure   = &passkeyAuthenticatorConfigurationResource{}
	_ resource.ResourceWithImportState = &passkeyAuthenticatorConfigurationResource{}
)

func NewPasskeyAuthenticatorConfigurationResource() resource.Resource {
	return &passkeyAuthenticatorConfigurationResource{}
}

type passkeyAuthenticatorConfigurationResource struct {
	client *authsignal.Client
}

type passkeyAuthenticatorConfigurationResourceModel struct {
	AuthenticatorId             types.String `tfsdk:"authenticator_id"`
	VerificationMethod          types.String `tfsdk:"verification_method"`
	IsActive                    types.Bool   `tfsdk:"is_active"`
	RelyingParty                types.String `tfsdk:"relying_party"`
	ExpectedOrigins             types.Set    `tfsdk:"expected_origins"`
	PasskeyRegistrationHints    types.List   `tfsdk:"passkey_registration_hints"`
	UserVerificationRequirement types.String `tfsdk:"user_verification_requirement"`
	AuthenticatorAttachment     types.String `tfsdk:"authenticator_attachment"`
}

func (r *passkeyAuthenticatorConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_passkey_authenticator_configuration"
}

func (r *passkeyAuthenticatorConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the tenant's passkey authenticator configuration. Destroying the resource deletes the configuration.",
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
			"relying_party": schema.StringAttribute{
				Description: "The relying party domain passkeys are scoped to, without a URL scheme.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"expected_origins": schema.SetAttribute{
				Description: "Allowed web or Android origins. Each must begin with `http://`, `https://`, or `android:`.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1), expectedOriginValidator{}),
				},
			},
			"passkey_registration_hints": schema.ListAttribute{
				Description: "Ordered browser hints for preferred authenticator types. Set to `[]` to clear them. Each element is one of `client-device`, `hybrid`, or `security-key`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(allowedPasskeyRegistrationHints...)),
				},
			},
			"user_verification_requirement": schema.StringAttribute{
				Description: "Whether the authenticator must verify the user, rather than only proving possession." + allowedValues(allowedUserVerificationRequirements),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedUserVerificationRequirements...),
				},
			},
			"authenticator_attachment": schema.StringAttribute{
				Description: "Which kinds of authenticator the browser may offer." + allowedValues(allowedAuthenticatorAttachments),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedAuthenticatorAttachments...),
				},
			},
		},
	}
}

func (r *passkeyAuthenticatorConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config passkeyAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := passkeyCreateBody(ctx, plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.CreatePasskeyAuthenticatorConfiguration(body)

	if statusCode == 409 {
		resp.Diagnostics.Append(createConflictDiagnostic("authsignal_passkey_authenticator_configuration", "passkey", passkeyAuthenticatorSlug))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating passkey authenticator configuration",
			"Could not create the passkey authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(passkeyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, passkeyStateAfterWrite(plan, response))...)
}

func (r *passkeyAuthenticatorConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	response, statusCode, err := r.client.GetPasskeyAuthenticatorConfiguration()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Passkey Authenticator Configuration",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(passkeyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, passkeyStateFromResponse(response))...)
}

func (r *passkeyAuthenticatorConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state passkeyAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := passkeyUpdateBody(ctx, plan, config, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, _, err := r.client.UpdatePasskeyAuthenticatorConfiguration(body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating passkey authenticator configuration",
			"Could not update the passkey authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(passkeyResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, passkeyStateAfterWrite(plan, response))...)
}

func (r *passkeyAuthenticatorConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, statusCode, err := r.client.DeletePasskeyAuthenticatorConfiguration()

	if statusCode == 404 {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting passkey authenticator configuration",
			"Could not delete the passkey authenticator configuration, unexpected error: "+err.Error(),
		)
	}
}

func (r *passkeyAuthenticatorConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(importStateSlug(req.ID, passkeyAuthenticatorSlug)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emptyPasskeyModel())...)
}

func emptyPasskeyModel() passkeyAuthenticatorConfigurationResourceModel {
	return passkeyAuthenticatorConfigurationResourceModel{
		ExpectedOrigins:          types.SetNull(types.StringType),
		PasskeyRegistrationHints: types.ListNull(types.StringType),
	}
}

func (r *passkeyAuthenticatorConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func passkeyCreateBody(ctx context.Context, plan passkeyAuthenticatorConfigurationResourceModel, config passkeyAuthenticatorConfigurationResourceModel) (authsignal.CreatePasskeyAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	origins, originDiags := sortedStringsFrom(ctx, plan.ExpectedOrigins)
	diags.Append(originDiags...)

	body := authsignal.CreatePasskeyAuthenticatorConfigurationBody{
		IsActive:                    boolPointer(config.IsActive),
		RelyingParty:                plan.RelyingParty.ValueString(),
		ExpectedOrigins:             origins,
		UserVerificationRequirement: stringPointer(config.UserVerificationRequirement),
		AuthenticatorAttachment:     stringPointer(config.AuthenticatorAttachment),
	}

	if isKnownAndSet(config.PasskeyRegistrationHints) {
		hints, hintDiags := stringsFrom(ctx, config.PasskeyRegistrationHints)
		diags.Append(hintDiags...)
		body.PasskeyRegistrationHints = authsignal.SetList(hints)
	}

	return body, diags
}

func passkeyUpdateBody(ctx context.Context, plan passkeyAuthenticatorConfigurationResourceModel, config passkeyAuthenticatorConfigurationResourceModel, state passkeyAuthenticatorConfigurationResourceModel) (authsignal.UpdatePasskeyAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	var body authsignal.UpdatePasskeyAuthenticatorConfigurationBody

	if isKnownAndSet(config.IsActive) {
		body.IsActive = authsignal.SetValue(config.IsActive.ValueBool())
	}

	if !plan.RelyingParty.Equal(state.RelyingParty) {
		body.RelyingParty = authsignal.SetValue(plan.RelyingParty.ValueString())
	}

	if !plan.ExpectedOrigins.Equal(state.ExpectedOrigins) {
		origins, originDiags := sortedStringsFrom(ctx, plan.ExpectedOrigins)
		diags.Append(originDiags...)
		body.ExpectedOrigins = authsignal.SetList(origins)
	}

	if isKnownAndSet(config.PasskeyRegistrationHints) {
		hints, hintDiags := stringsFrom(ctx, config.PasskeyRegistrationHints)
		diags.Append(hintDiags...)
		body.PasskeyRegistrationHints = authsignal.SetList(hints)
	}

	if isKnownAndSet(config.UserVerificationRequirement) {
		body.UserVerificationRequirement = authsignal.SetValue(config.UserVerificationRequirement.ValueString())
	}

	if isKnownAndSet(config.AuthenticatorAttachment) {
		body.AuthenticatorAttachment = authsignal.SetValue(config.AuthenticatorAttachment.ValueString())
	}

	return body, diags
}

func passkeyStateAfterWrite(plan passkeyAuthenticatorConfigurationResourceModel, response *authsignal.PasskeyAuthenticatorConfiguration) passkeyAuthenticatorConfigurationResourceModel {
	state := plan
	state.AuthenticatorId = types.StringValue(response.AuthenticatorId)
	state.VerificationMethod = types.StringValue(response.VerificationMethod)

	state.IsActive = resolvedValue(plan.IsActive, types.BoolValue(response.IsActive))
	state.PasskeyRegistrationHints = resolvedValue(plan.PasskeyRegistrationHints, stringListPointerValue(response.PasskeyRegistrationHints))
	state.UserVerificationRequirement = resolvedValue(plan.UserVerificationRequirement, types.StringPointerValue(response.UserVerificationRequirement))
	state.AuthenticatorAttachment = resolvedValue(plan.AuthenticatorAttachment, types.StringPointerValue(response.AuthenticatorAttachment))

	return state
}

func passkeyStateFromResponse(response *authsignal.PasskeyAuthenticatorConfiguration) passkeyAuthenticatorConfigurationResourceModel {
	return passkeyAuthenticatorConfigurationResourceModel{
		AuthenticatorId:             types.StringValue(response.AuthenticatorId),
		VerificationMethod:          types.StringValue(response.VerificationMethod),
		IsActive:                    types.BoolValue(response.IsActive),
		RelyingParty:                types.StringPointerValue(response.RelyingParty),
		ExpectedOrigins:             stringSetPointerValue(response.ExpectedOrigins),
		PasskeyRegistrationHints:    stringListPointerValue(response.PasskeyRegistrationHints),
		UserVerificationRequirement: types.StringPointerValue(response.UserVerificationRequirement),
		AuthenticatorAttachment:     types.StringPointerValue(response.AuthenticatorAttachment),
	}
}
