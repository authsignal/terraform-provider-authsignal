package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &pushAuthenticatorConfigurationResource{}
	_ resource.ResourceWithConfigure      = &pushAuthenticatorConfigurationResource{}
	_ resource.ResourceWithImportState    = &pushAuthenticatorConfigurationResource{}
	_ resource.ResourceWithValidateConfig = &pushAuthenticatorConfigurationResource{}
)

func NewPushAuthenticatorConfigurationResource() resource.Resource {
	return &pushAuthenticatorConfigurationResource{}
}

type pushAuthenticatorConfigurationResource struct {
	client *authsignal.Client
}

type pushAuthenticatorConfigurationResourceModel struct {
	AuthenticatorId             types.String `tfsdk:"authenticator_id"`
	VerificationMethod          types.String `tfsdk:"verification_method"`
	IsActive                    types.Bool   `tfsdk:"is_active"`
	PushProvider                types.String `tfsdk:"push_provider"`
	WebhookUrl                  types.String `tfsdk:"webhook_url"`
	CredentialLifetimeInMinutes types.Int64  `tfsdk:"credential_lifetime_in_minutes"`
	RequireAppAttestation       types.Bool   `tfsdk:"require_app_attestation"`
	AppAttestationFailureMode   types.String `tfsdk:"app_attestation_failure_mode"`

	SendingRateLimitConfigurations types.List `tfsdk:"sending_rate_limit_configurations"`

	ApnsCredentials        types.Object `tfsdk:"apns_credentials"`
	ApnsCredentialsVersion types.String `tfsdk:"apns_credentials_version"`
	FcmCredentials         types.Object `tfsdk:"fcm_credentials"`
	FcmCredentialsVersion  types.String `tfsdk:"fcm_credentials_version"`

	FcmProjectId   types.String `tfsdk:"fcm_project_id"`
	FcmClientEmail types.String `tfsdk:"fcm_client_email"`
}

const (
	apnsCredentialsBlock = "apns_credentials"
	fcmCredentialsBlock  = "fcm_credentials"
)

var apnsBlock = credentialBlock{
	Name:      apnsCredentialsBlock,
	Providers: []string{"DEFAULT"},
	Description: "Apple Push Notification service credentials. Usable when `push_provider` is `DEFAULT`, which delivers to " +
		"Apple devices through APNs and to Android devices through Firebase.",
	Members: []credentialMember{
		{Name: "private_key", Kind: credentialString, Secret: true, Required: true, Description: "APNs signing key (.p8 contents)."},
		{Name: "team_id", Kind: credentialString, Required: true, Description: "Apple developer team ID."},
		{Name: "key_id", Kind: credentialString, Required: true, Description: "ID of the APNs signing key."},
		{Name: "bundle_id", Kind: credentialString, Required: true, Description: "Bundle ID of the app receiving the notification."},
		{Name: "sandbox", Kind: credentialBool, Description: "Whether notifications go to the APNs sandbox rather than production."},
	},
}

var fcmBlock = credentialBlock{
	Name:      fcmCredentialsBlock,
	Providers: []string{"DEFAULT", "FIREBASE"},
	Description: "Firebase Cloud Messaging credentials. Required for `FIREBASE` and supported for `DEFAULT`. " +
		"Derived project and client email values are available as `fcm_project_id` and `fcm_client_email`.",
	Members: []credentialMember{
		{Name: "service_account_key", Kind: credentialString, Secret: true, Required: true, Description: "Service account key as JSON."},
	},
}

var pushCredentialBlocks = []credentialBlock{apnsBlock, fcmBlock}

func (r *pushAuthenticatorConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_push_authenticator_configuration"
}

func (r *pushAuthenticatorConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
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
		"push_provider": schema.StringAttribute{
			Description: "Push delivery provider. `DEFAULT` supports APNs and Firebase; `WEBHOOK` requires `webhook_url`." + allowedValues(allowedPushProviders),
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(allowedPushProviders...),
			},
		},
		"webhook_url": schema.StringAttribute{
			Description: "HTTPS delivery endpoint. Required for `WEBHOOK` and rejected for other providers. Switching providers clears it.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				supersededWebhookUrl{providerAttribute: "push_provider"},
			},
			Validators: []validator.String{
				httpsUrlValidator{},
			},
		},
		"credential_lifetime_in_minutes": schema.Int64Attribute{
			Description: "Push credential lifetime in minutes. Must be between 0 and 525600.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.Int64{
				int64validator.Between(0, 525600),
			},
		},
		"require_app_attestation": schema.BoolAttribute{
			Description: "Whether the app must attest its integrity before a push challenge is accepted. `app_attestation_failure_mode` decides what happens when attestation fails.",
			Optional:    true,
			Computed:    true,
		},
		"app_attestation_failure_mode": schema.StringAttribute{
			Description: "What happens when app attestation fails. Applies only while `require_app_attestation` is true. `BLOCK` refuses the challenge; `ALLOW_WITH_WARNING` allows it and flags it.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(allowedAppAttestationFailureModes...),
			},
		},
		"sending_rate_limit_configurations": schema.ListNestedAttribute{
			Description: "Notification delivery limits. All configured limits apply.",
			Optional:    true,
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: rateLimitAttributes(),
			},
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.SizeAtMost(6),
			},
		},
		"fcm_project_id": schema.StringAttribute{
			Description: "The Firebase project id, read back from the service account key.",
			Computed:    true,
		},
		"fcm_client_email": schema.StringAttribute{
			Description: "The Firebase client email, read back from the service account key.",
			Computed:    true,
		},
	}

	for _, block := range pushCredentialBlocks {
		for name, attribute := range block.schemaAttributes() {
			attributes[name] = attribute
		}
	}

	resp.Schema = schema.Schema{
		Description: "Manages the tenant's push authenticator configuration. Destroying the resource deletes the configuration.",
		Attributes:  attributes,
	}
}

func (r *pushAuthenticatorConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config pushAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(writeOnlyBlockDiagnostics(pushCredentialBlocks, config.credentialObject, config.credentialVersion)...)

	if !isKnownAndSet(config.PushProvider) {
		return
	}

	provider := config.PushProvider.ValueString()

	for _, block := range pushCredentialBlocks {
		if block.appliesTo(provider) {
			continue
		}

		if isKnownAndSet(config.credentialObject(block.Name)) {
			resp.Diagnostics.Append(mismatchedCredentialDiagnostic(block.Name, "push_provider", provider))
		}
	}

	resp.Diagnostics.Append(webhookUrlDiagnostics("push_provider", provider, config.WebhookUrl)...)
}

func (m pushAuthenticatorConfigurationResourceModel) credentialObject(name string) types.Object {
	switch name {
	case apnsCredentialsBlock:
		return m.ApnsCredentials
	case fcmCredentialsBlock:
		return m.FcmCredentials
	default:
		return types.ObjectNull(map[string]attr.Type{})
	}
}

func (m pushAuthenticatorConfigurationResourceModel) credentialVersion(name string) types.String {
	switch name {
	case apnsCredentialsBlock:
		return m.ApnsCredentialsVersion
	case fcmCredentialsBlock:
		return m.FcmCredentialsVersion
	default:
		return types.StringNull()
	}
}

func (r *pushAuthenticatorConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config pushAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := pushCreateBody(ctx, plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.CreatePushAuthenticatorConfiguration(body)

	if statusCode == 409 {
		resp.Diagnostics.Append(createConflictDiagnostic("authsignal_push_authenticator_configuration", "push", pushAuthenticatorSlug))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating push authenticator configuration",
			"Could not create the push authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(pushResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, pushStateAfterWrite(plan, response))...)
}

func (r *pushAuthenticatorConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pushAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.GetPushAuthenticatorConfiguration()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Push Authenticator Configuration",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(pushResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, pushStateFromResponse(response, state))...)
}

func (r *pushAuthenticatorConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state pushAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := pushUpdateBody(ctx, plan, config, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, _, err := r.client.UpdatePushAuthenticatorConfiguration(body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating push authenticator configuration",
			"Could not update the push authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(pushResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, pushStateAfterWrite(plan, response))...)
}

func (r *pushAuthenticatorConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, statusCode, err := r.client.DeletePushAuthenticatorConfiguration()

	if statusCode == 404 {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting push authenticator configuration",
			"Could not delete the push authenticator configuration, unexpected error: "+err.Error(),
		)
	}
}

func (r *pushAuthenticatorConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(importStateSlug(req.ID, pushAuthenticatorSlug)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emptyPushModel())...)
}

func emptyPushModel() pushAuthenticatorConfigurationResourceModel {
	model := pushAuthenticatorConfigurationResourceModel{
		SendingRateLimitConfigurations: types.ListNull(rateLimitObjectType()),
	}

	model.ApnsCredentials = types.ObjectNull(apnsBlock.attributeTypes())
	model.FcmCredentials = types.ObjectNull(fcmBlock.attributeTypes())

	return model
}

func (r *pushAuthenticatorConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func pushCreateBody(ctx context.Context, plan pushAuthenticatorConfigurationResourceModel, config pushAuthenticatorConfigurationResourceModel) (authsignal.CreatePushAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := authsignal.CreatePushAuthenticatorConfigurationBody{
		IsActive:                    boolPointer(config.IsActive),
		PushProvider:                plan.PushProvider.ValueString(),
		WebhookUrl:                  stringPointer(config.WebhookUrl),
		CredentialLifetimeInMinutes: int64Pointer(config.CredentialLifetimeInMinutes),
		RequireAppAttestation:       boolPointer(config.RequireAppAttestation),
		AppAttestationFailureMode:   stringPointer(config.AppAttestationFailureMode),
	}

	sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
	diags.Append(sendingDiags...)
	if sending != nil {
		body.SendingRateLimitConfigurations = authsignal.SetList(*sending)
	}

	provider := plan.PushProvider.ValueString()

	for _, block := range pushCredentialBlocks {
		if !block.appliesTo(provider) {
			continue
		}

		values := readCredentialValues(config.credentialObject(block.Name), block)
		if !values.present {
			continue
		}

		if missing := block.missingRequired(values); len(missing) > 0 {
			diags.Append(incompleteCredentialDiagnostic(block.Name, missing))
			continue
		}

		applyPushCreateCredentials(&body, block.Name, values)
	}

	diags.Append(pushCredentialsSufficient(provider, body.ApnsCredentials != nil, body.FcmCredentials != nil)...)

	return body, diags
}

func pushCredentialsSufficient(provider string, hasApns bool, hasFcm bool) diag.Diagnostics {
	var diags diag.Diagnostics

	switch provider {
	case "DEFAULT":
		if !hasApns && !hasFcm {
			diags.AddError(
				"Missing Credentials for the Configured Provider",
				"`push_provider` is \"DEFAULT\", so at least one of `apns_credentials` and `fcm_credentials` has to be set when the configuration is created.",
			)
		}
	case "FIREBASE":
		if !hasFcm {
			diags.Append(missingCredentialDiagnostic(fcmCredentialsBlock, "push_provider", provider))
		}
	}

	return diags
}

func applyPushCreateCredentials(body *authsignal.CreatePushAuthenticatorConfigurationBody, name string, values credentialValues) {
	switch name {
	case apnsCredentialsBlock:
		body.ApnsCredentials = &authsignal.ApnsCredentialsCreate{
			TeamId:     values.stringValue("team_id"),
			KeyId:      values.stringValue("key_id"),
			BundleId:   values.stringValue("bundle_id"),
			PrivateKey: values.stringValue("private_key"),
			Sandbox:    values.boolPointer("sandbox"),
		}
	case fcmCredentialsBlock:
		body.FcmCredentials = &authsignal.FcmCredentialsCreate{
			ServiceAccountKey: values.stringValue("service_account_key"),
		}
	}
}

func pushUpdateBody(ctx context.Context, plan pushAuthenticatorConfigurationResourceModel, config pushAuthenticatorConfigurationResourceModel, state pushAuthenticatorConfigurationResourceModel) (authsignal.UpdatePushAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	var body authsignal.UpdatePushAuthenticatorConfigurationBody

	providerChanging := !plan.PushProvider.Equal(state.PushProvider)
	provider := plan.PushProvider.ValueString()

	if providerChanging {
		body.PushProvider = authsignal.SetValue(provider)
	}

	if isKnownAndSet(config.IsActive) {
		body.IsActive = authsignal.SetValue(config.IsActive.ValueBool())
	}

	if isKnownAndSet(config.RequireAppAttestation) {
		body.RequireAppAttestation = authsignal.SetValue(config.RequireAppAttestation.ValueBool())
	}

	if isKnownAndSet(config.AppAttestationFailureMode) {
		body.AppAttestationFailureMode = authsignal.SetValue(config.AppAttestationFailureMode.ValueString())
	}

	if isKnownAndSet(config.WebhookUrl) {
		body.WebhookUrl = authsignal.SetValue(config.WebhookUrl.ValueString())
	}

	if isKnownAndSet(config.CredentialLifetimeInMinutes) {
		body.CredentialLifetimeInMinutes = authsignal.SetValue(config.CredentialLifetimeInMinutes.ValueInt64())
	}

	if isKnownAndSet(config.SendingRateLimitConfigurations) {
		sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
		diags.Append(sendingDiags...)
		if sending != nil {
			body.SendingRateLimitConfigurations = authsignal.SetValue(*sending)
		}
	}

	for _, block := range pushCredentialBlocks {
		values := readCredentialValues(config.credentialObject(block.Name), block)

		decision := credentialPlanFor(block, credentialContext{
			ProviderChanging: providerChanging,
			IsEffective:      block.appliesTo(provider),
			ConfigValues:     values,
			StateObject:      state.credentialObject(block.Name),
			PriorVersion:     state.credentialVersion(block.Name),
			VersionChanged:   !plan.credentialVersion(block.Name).Equal(state.credentialVersion(block.Name)),
		})

		if decision.Send == credentialWrite {
			applyPushUpdateCredentials(&body, block, values, decision.IncludeSecrets)
		}
	}

	return body, diags
}

func applyPushUpdateCredentials(body *authsignal.UpdatePushAuthenticatorConfigurationBody, block credentialBlock, values credentialValues, includeSecrets bool) {
	secret := func(name string) *string {
		if !includeSecrets {
			return nil
		}

		return values.stringPointer(name)
	}

	switch block.Name {
	case apnsCredentialsBlock:
		body.ApnsCredentials = authsignal.SetValue(authsignal.ApnsCredentialsUpdate{
			TeamId:     values.stringPointer("team_id"),
			KeyId:      values.stringPointer("key_id"),
			BundleId:   values.stringPointer("bundle_id"),
			PrivateKey: secret("private_key"),
			Sandbox:    values.boolPointer("sandbox"),
		})
	case fcmCredentialsBlock:
		body.FcmCredentials = authsignal.SetValue(authsignal.FcmCredentialsUpdate{
			ServiceAccountKey: secret("service_account_key"),
		})
	}
}

func pushStateAfterWrite(plan pushAuthenticatorConfigurationResourceModel, response *authsignal.PushAuthenticatorConfiguration) pushAuthenticatorConfigurationResourceModel {
	state := plan
	state.AuthenticatorId = types.StringValue(response.AuthenticatorId)
	state.VerificationMethod = types.StringValue(response.VerificationMethod)

	state.IsActive = resolvedValue(plan.IsActive, types.BoolValue(response.IsActive))
	state.RequireAppAttestation = resolvedValue(plan.RequireAppAttestation, types.BoolPointerValue(response.RequireAppAttestation))
	state.AppAttestationFailureMode = resolvedValue(plan.AppAttestationFailureMode, types.StringPointerValue(response.AppAttestationFailureMode))
	state.WebhookUrl = resolvedValue(plan.WebhookUrl, types.StringPointerValue(response.WebhookUrl))
	state.CredentialLifetimeInMinutes = resolvedValue(plan.CredentialLifetimeInMinutes, types.Int64PointerValue(response.CredentialLifetimeInMinutes))
	state.SendingRateLimitConfigurations = resolvedValue(plan.SendingRateLimitConfigurations, rateLimitListValue(response.SendingRateLimitConfigurations))

	state.ApnsCredentials = credentialPlannedState(apnsBlock, plan.ApnsCredentials)
	state.FcmCredentials = credentialPlannedState(fcmBlock, plan.FcmCredentials)

	applyFirebaseDerivedState(&state, response)

	return state
}

func pushStateFromResponse(response *authsignal.PushAuthenticatorConfiguration, prior pushAuthenticatorConfigurationResourceModel) pushAuthenticatorConfigurationResourceModel {
	state := pushAuthenticatorConfigurationResourceModel{
		AuthenticatorId:                types.StringValue(response.AuthenticatorId),
		VerificationMethod:             types.StringValue(response.VerificationMethod),
		IsActive:                       types.BoolValue(response.IsActive),
		PushProvider:                   types.StringPointerValue(response.PushProvider),
		WebhookUrl:                     types.StringPointerValue(response.WebhookUrl),
		CredentialLifetimeInMinutes:    types.Int64PointerValue(response.CredentialLifetimeInMinutes),
		RequireAppAttestation:          types.BoolPointerValue(response.RequireAppAttestation),
		AppAttestationFailureMode:      types.StringPointerValue(response.AppAttestationFailureMode),
		SendingRateLimitConfigurations: rateLimitListValue(response.SendingRateLimitConfigurations),

		ApnsCredentialsVersion: prior.ApnsCredentialsVersion,
		FcmCredentialsVersion:  prior.FcmCredentialsVersion,
	}

	applyPushCredentialReadState(&state, response, prior)

	return state
}

func applyPushCredentialReadState(state *pushAuthenticatorConfigurationResourceModel, response *authsignal.PushAuthenticatorConfiguration, prior pushAuthenticatorConfigurationResourceModel) {
	apnsMetadata := map[string]attr.Value{}
	if response.ApnsCredentials != nil {
		apnsMetadata = map[string]attr.Value{
			"team_id":   types.StringPointerValue(response.ApnsCredentials.TeamId),
			"key_id":    types.StringPointerValue(response.ApnsCredentials.KeyId),
			"bundle_id": types.StringPointerValue(response.ApnsCredentials.BundleId),
			"sandbox":   types.BoolPointerValue(response.ApnsCredentials.Sandbox),
		}
	}
	state.ApnsCredentials = credentialReadState(apnsBlock, apnsMetadata, response.ApnsCredentials != nil, prior.ApnsCredentials)

	state.FcmCredentials = credentialReadState(fcmBlock, nil, response.FcmCredentials != nil, prior.FcmCredentials)

	applyFirebaseDerivedState(state, response)
}

func applyFirebaseDerivedState(state *pushAuthenticatorConfigurationResourceModel, response *authsignal.PushAuthenticatorConfiguration) {
	state.FcmProjectId = types.StringNull()
	state.FcmClientEmail = types.StringNull()

	if response.FcmCredentials != nil {
		state.FcmProjectId = types.StringPointerValue(response.FcmCredentials.ProjectId)
		state.FcmClientEmail = types.StringPointerValue(response.FcmCredentials.ClientEmail)
	}
}
