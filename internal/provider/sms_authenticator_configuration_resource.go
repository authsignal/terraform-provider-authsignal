package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                   = &smsAuthenticatorConfigurationResource{}
	_ resource.ResourceWithConfigure      = &smsAuthenticatorConfigurationResource{}
	_ resource.ResourceWithImportState    = &smsAuthenticatorConfigurationResource{}
	_ resource.ResourceWithValidateConfig = &smsAuthenticatorConfigurationResource{}
)

func NewSmsAuthenticatorConfigurationResource() resource.Resource {
	return &smsAuthenticatorConfigurationResource{}
}

type smsAuthenticatorConfigurationResource struct {
	client *authsignal.Client
}

type smsAuthenticatorConfigurationResourceModel struct {
	AuthenticatorId    types.String `tfsdk:"authenticator_id"`
	VerificationMethod types.String `tfsdk:"verification_method"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	SmsProvider        types.String `tfsdk:"sms_provider"`
	SmsCountryCodes    types.Set    `tfsdk:"sms_country_codes"`
	DefaultCountryCode types.String `tfsdk:"default_country_code"`

	SubmissionRateLimitConfiguration types.Object `tfsdk:"submission_rate_limit_configuration"`
	SendingRateLimitConfigurations   types.List   `tfsdk:"sending_rate_limit_configurations"`
	PrefixRateLimitConfigurations    types.List   `tfsdk:"prefix_rate_limit_configurations"`
	AllowedCustomSmsVariables        types.Set    `tfsdk:"allowed_custom_sms_variables"`

	TwilioCredentials              types.Object `tfsdk:"twilio_credentials"`
	TwilioCredentialsVersion       types.String `tfsdk:"twilio_credentials_version"`
	BirdSmsCredentials             types.Object `tfsdk:"bird_sms_credentials"`
	BirdSmsCredentialsVersion      types.String `tfsdk:"bird_sms_credentials_version"`
	MessageMediaCredentials        types.Object `tfsdk:"message_media_credentials"`
	MessageMediaCredentialsVersion types.String `tfsdk:"message_media_credentials_version"`
	ModicaGroupCredentials         types.Object `tfsdk:"modica_group_credentials"`
	ModicaGroupCredentialsVersion  types.String `tfsdk:"modica_group_credentials_version"`
	TnzCredentials                 types.Object `tfsdk:"tnz_credentials"`
	TnzCredentialsVersion          types.String `tfsdk:"tnz_credentials_version"`
}

const (
	twilioCredentialsBlock       = "twilio_credentials"
	birdSmsCredentialsBlock      = "bird_sms_credentials"
	messageMediaCredentialsBlock = "message_media_credentials"
	modicaGroupCredentialsBlock  = "modica_group_credentials"
	tnzCredentialsBlock          = "tnz_credentials"
)

var twilioBlock = credentialBlock{
	Name:        twilioCredentialsBlock,
	Providers:   []string{"TWILIO"},
	Description: "Twilio credentials. Required when `sms_provider` is `TWILIO` and no Twilio credentials are stored yet.",
	Members: []credentialMember{
		{Name: "auth_token", Kind: credentialString, Secret: true, Required: true, Description: "Twilio auth token."},
		{Name: "messaging_service_sid", Kind: credentialString, Required: true, Description: "Twilio Messaging Service SID."},
		{Name: "account_sid", Kind: credentialString, Required: true, Description: "Twilio Account SID."},
	},
}

var birdSmsBlock = credentialBlock{
	Name:        birdSmsCredentialsBlock,
	Providers:   []string{"BIRD"},
	Description: "Bird SMS credentials. Required when `sms_provider` is `BIRD` and no Bird credentials are stored yet.",
	Members: []credentialMember{
		{Name: "access_key", Kind: credentialString, Secret: true, Required: true, Description: "Bird access key."},
		{Name: "workspace_id", Kind: credentialString, Required: true, Description: "Bird workspace ID."},
		{Name: "channel_id", Kind: credentialString, Description: "Bird channel ID."},
		{Name: "navigator_id", Kind: credentialString, Description: "Bird navigator ID."},
		{Name: "project_id", Kind: credentialString, Description: "Bird project ID."},
		{Name: "locale", Kind: credentialString, Description: "The locale Bird renders message templates in."},
		{Name: "enable_message_templates", Kind: credentialBool, Description: "Whether Bird message templates are used instead of the message body Authsignal sends."},
	},
}

var messageMediaBlock = credentialBlock{
	Name:        messageMediaCredentialsBlock,
	Providers:   []string{"MESSAGE_MEDIA"},
	Description: "MessageMedia credentials. Required when `sms_provider` is `MESSAGE_MEDIA` and no MessageMedia credentials are stored yet.",
	Members: []credentialMember{
		{Name: "api_key", Kind: credentialString, Secret: true, Required: true, Description: "MessageMedia API key."},
		{Name: "api_secret", Kind: credentialString, Secret: true, Required: true, Description: "MessageMedia API secret."},
		{Name: "source_number", Kind: credentialString, Description: "Number messages are sent from."},
	},
}

var modicaGroupBlock = credentialBlock{
	Name:        modicaGroupCredentialsBlock,
	Providers:   []string{"MODICA_GROUP"},
	Description: "Modica Group credentials. Required when `sms_provider` is `MODICA_GROUP` and no Modica Group credentials are stored yet.",
	Members: []credentialMember{
		{Name: "username", Kind: credentialString, Required: true, Description: "Modica Group username."},
		{Name: "password", Kind: credentialString, Secret: true, Required: true, Description: "Modica Group password."},
	},
}

var tnzBlock = credentialBlock{
	Name:        tnzCredentialsBlock,
	Providers:   []string{"TNZ"},
	Description: "TNZ credentials. Required when `sms_provider` is `TNZ` and no TNZ credentials are stored.",
	Members: []credentialMember{
		{Name: "api_key", Kind: credentialString, Secret: true, Required: true, Description: "TNZ API key."},
	},
}

var smsCredentialBlocks = []credentialBlock{twilioBlock, birdSmsBlock, messageMediaBlock, modicaGroupBlock, tnzBlock}

func (r *smsAuthenticatorConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sms_authenticator_configuration"
}

func (r *smsAuthenticatorConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		"sms_provider": schema.StringAttribute{
			Description: "SMS delivery provider. Supply its credential block when first selecting it." + allowedValues(allowedSmsProviders),
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(allowedSmsProviders...),
			},
		},
		"sms_country_codes": schema.SetAttribute{
			Description: "Allowed two-letter destination country codes. Set to `[]` to clear the restriction and `default_country_code`.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(stringvalidator.OneOf(smsCountryCodes...)),
			},
		},
		"default_country_code": schema.StringAttribute{
			Description: "Country code used for numbers without one. It must be included in `sms_country_codes`.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(smsCountryCodes...),
			},
			PlanModifiers: []planmodifier.String{
				excludedDefaultCountryCode{},
			},
		},
		"submission_rate_limit_configuration": schema.SingleNestedAttribute{
			Description: "Code submission limit.",
			Optional:    true,
			Computed:    true,
			Attributes:  rateLimitAttributes(),
		},
		"sending_rate_limit_configurations": schema.ListNestedAttribute{
			Description: "Message delivery limits. All configured limits apply.",
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
		"prefix_rate_limit_configurations": schema.ListNestedAttribute{
			Description: "Message delivery limits by number prefix. Set to `[]` to clear them.",
			Optional:    true,
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: prefixRateLimitAttributes(),
			},
			Validators: []validator.List{
				listvalidator.SizeAtMost(5),
			},
		},
		"allowed_custom_sms_variables": schema.SetAttribute{
			Description: "Custom SMS template variables. Set to `[]` to clear them.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
		},
	}

	for _, block := range smsCredentialBlocks {
		for name, attribute := range block.schemaAttributes() {
			attributes[name] = attribute
		}
	}

	resp.Schema = schema.Schema{
		Description: "Manages the tenant's SMS OTP authenticator configuration. Destroying the resource deletes the configuration.",
		Attributes:  attributes,
	}
}

func countryCodeDiagnostics(ctx context.Context, countryCodes types.Set, defaultCode types.String) diag.Diagnostics {
	var diags diag.Diagnostics

	if !isKnownAndSet(countryCodes) {
		return diags
	}

	if len(countryCodes.Elements()) == 0 {
		if isKnownAndSet(defaultCode) {
			diags.Append(clearedCountryCodesDiagnostic())
		}

		return diags
	}

	if !isKnownAndSet(defaultCode) {
		return diags
	}

	allowed, elementDiags := sortedStringsFrom(ctx, countryCodes)
	diags.Append(elementDiags...)
	if diags.HasError() {
		return diags
	}

	for _, code := range allowed {
		if code == defaultCode.ValueString() {
			return diags
		}
	}

	diags.Append(defaultCountryCodeOutsideRestrictionDiagnostic(defaultCode.ValueString(), allowed))

	return diags
}

func prefixRateLimitAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Optional: true,
		},
		"block_size": schema.StringAttribute{
			Description: "How numbers are grouped: the string `country`, or a positive number of leading digits.",
			Optional:    true,
			Validators: []validator.String{
				blockSizeValidator{},
			},
		},
		"rate_limit": schema.Int64Attribute{
			Description: "Maximum messages in the window. Must be between 1 and 1000.",
			Optional:    true,
			Validators: []validator.Int64{
				int64validator.Between(1, 1000),
			},
		},
		"window_in_minutes": schema.Float64Attribute{
			Description: "Window length in minutes. Must be greater than 0 and at most 1440; fractional values are supported.",
			Optional:    true,
			Validators: []validator.Float64{
				float64validator.AtMost(1440),
				greaterThanZeroValidator{},
			},
		},
		"country_codes": schema.SetAttribute{
			Description: "The two-letter country codes this limit applies to.",
			Optional:    true,
			ElementType: types.StringType,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(stringvalidator.OneOf(smsCountryCodes...)),
			},
		},
	}
}

func (r *smsAuthenticatorConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config smsAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(writeOnlyBlockDiagnostics(smsCredentialBlocks, config.credentialObject, config.credentialVersion)...)

	resp.Diagnostics.Append(countryCodeDiagnostics(ctx, config.SmsCountryCodes, config.DefaultCountryCode)...)

	if !isKnownAndSet(config.SmsProvider) {
		return
	}

	provider := config.SmsProvider.ValueString()

	for _, block := range smsCredentialBlocks {
		if block.appliesTo(provider) {
			continue
		}

		if isKnownAndSet(config.credentialObject(block.Name)) {
			resp.Diagnostics.Append(mismatchedCredentialDiagnostic(block.Name, "sms_provider", provider))
		}
	}
}

func (m smsAuthenticatorConfigurationResourceModel) credentialObject(name string) types.Object {
	switch name {
	case twilioCredentialsBlock:
		return m.TwilioCredentials
	case birdSmsCredentialsBlock:
		return m.BirdSmsCredentials
	case messageMediaCredentialsBlock:
		return m.MessageMediaCredentials
	case modicaGroupCredentialsBlock:
		return m.ModicaGroupCredentials
	case tnzCredentialsBlock:
		return m.TnzCredentials
	default:
		return types.ObjectNull(map[string]attr.Type{})
	}
}

func (m smsAuthenticatorConfigurationResourceModel) credentialVersion(name string) types.String {
	switch name {
	case twilioCredentialsBlock:
		return m.TwilioCredentialsVersion
	case birdSmsCredentialsBlock:
		return m.BirdSmsCredentialsVersion
	case messageMediaCredentialsBlock:
		return m.MessageMediaCredentialsVersion
	case modicaGroupCredentialsBlock:
		return m.ModicaGroupCredentialsVersion
	case tnzCredentialsBlock:
		return m.TnzCredentialsVersion
	default:
		return types.StringNull()
	}
}

func (r *smsAuthenticatorConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config smsAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := smsCreateBody(ctx, plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.CreateSmsAuthenticatorConfiguration(body)

	if statusCode == 409 {
		resp.Diagnostics.Append(createConflictDiagnostic("authsignal_sms_authenticator_configuration", "sms", smsAuthenticatorSlug))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SMS authenticator configuration",
			"Could not create the SMS authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(smsResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, smsStateAfterWrite(plan, response))...)
}

func (r *smsAuthenticatorConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state smsAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.GetSmsAuthenticatorConfiguration()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal SMS Authenticator Configuration",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(smsResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, smsStateFromResponse(response, state))...)
}

func (r *smsAuthenticatorConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state smsAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := smsUpdateBody(ctx, plan, config, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, _, err := r.client.UpdateSmsAuthenticatorConfiguration(body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating SMS authenticator configuration",
			"Could not update the SMS authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(smsResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, smsStateAfterWrite(plan, response))...)
}

func (r *smsAuthenticatorConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, statusCode, err := r.client.DeleteSmsAuthenticatorConfiguration()

	if statusCode == 404 {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting SMS authenticator configuration",
			"Could not delete the SMS authenticator configuration, unexpected error: "+err.Error(),
		)
	}
}

func (r *smsAuthenticatorConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(importStateSlug(req.ID, smsAuthenticatorSlug)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emptySmsModel())...)
}

func emptySmsModel() smsAuthenticatorConfigurationResourceModel {
	model := smsAuthenticatorConfigurationResourceModel{
		SmsCountryCodes:                  types.SetNull(types.StringType),
		SubmissionRateLimitConfiguration: types.ObjectNull(rateLimitAttributeTypes),
		SendingRateLimitConfigurations:   types.ListNull(rateLimitObjectType()),
		PrefixRateLimitConfigurations:    types.ListNull(prefixRateLimitObjectType()),
		AllowedCustomSmsVariables:        types.SetNull(types.StringType),
	}

	model.TwilioCredentials = types.ObjectNull(twilioBlock.attributeTypes())
	model.BirdSmsCredentials = types.ObjectNull(birdSmsBlock.attributeTypes())
	model.MessageMediaCredentials = types.ObjectNull(messageMediaBlock.attributeTypes())
	model.ModicaGroupCredentials = types.ObjectNull(modicaGroupBlock.attributeTypes())
	model.TnzCredentials = types.ObjectNull(tnzBlock.attributeTypes())

	return model
}

func (r *smsAuthenticatorConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func smsCreateBody(ctx context.Context, plan smsAuthenticatorConfigurationResourceModel, config smsAuthenticatorConfigurationResourceModel) (authsignal.CreateSmsAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := authsignal.CreateSmsAuthenticatorConfigurationBody{
		IsActive:           boolPointer(config.IsActive),
		SmsProvider:        plan.SmsProvider.ValueString(),
		DefaultCountryCode: stringPointer(config.DefaultCountryCode),
	}

	if isKnownAndSet(config.SmsCountryCodes) {
		countryCodes, setDiags := sortedStringsFrom(ctx, config.SmsCountryCodes)
		diags.Append(setDiags...)
		body.SmsCountryCodes = authsignal.SetList(countryCodes)
	}

	if isKnownAndSet(config.AllowedCustomSmsVariables) {
		variables, setDiags := sortedStringsFrom(ctx, config.AllowedCustomSmsVariables)
		diags.Append(setDiags...)
		body.AllowedCustomSmsVariables = authsignal.SetList(variables)
	}

	submission, submissionDiags := rateLimitFrom(ctx, config.SubmissionRateLimitConfiguration)
	diags.Append(submissionDiags...)
	body.SubmissionRateLimitConfiguration = submission

	sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
	diags.Append(sendingDiags...)
	if sending != nil {
		body.SendingRateLimitConfigurations = authsignal.SetList(*sending)
	}

	if isKnownAndSet(config.PrefixRateLimitConfigurations) {
		prefixes, prefixDiags := prefixRateLimitsFrom(ctx, config.PrefixRateLimitConfigurations)
		diags.Append(prefixDiags...)
		body.PrefixRateLimitConfigurations = authsignal.SetList(prefixes)
	}

	provider := plan.SmsProvider.ValueString()

	for _, block := range smsCredentialBlocks {
		if !block.appliesTo(provider) {
			continue
		}

		values := readCredentialValues(config.credentialObject(block.Name), block)
		if !values.present {
			diags.Append(missingCredentialDiagnostic(block.Name, "sms_provider", provider))
			continue
		}

		if missing := block.missingRequired(values); len(missing) > 0 {
			diags.Append(incompleteCredentialDiagnostic(block.Name, missing))
			continue
		}

		applySmsCreateCredentials(&body, block.Name, values)
	}

	return body, diags
}

func applySmsCreateCredentials(body *authsignal.CreateSmsAuthenticatorConfigurationBody, name string, values credentialValues) {
	switch name {
	case twilioCredentialsBlock:
		body.TwilioCredentials = &authsignal.TwilioCredentialsCreate{
			AuthToken:           values.stringValue("auth_token"),
			MessagingServiceSid: values.stringValue("messaging_service_sid"),
			AccountSid:          values.stringValue("account_sid"),
		}
	case birdSmsCredentialsBlock:
		body.BirdSmsCredentials = &authsignal.BirdSmsCredentialsCreate{
			AccessKey:              values.stringValue("access_key"),
			WorkspaceId:            values.stringValue("workspace_id"),
			ChannelId:              values.stringPointer("channel_id"),
			NavigatorId:            values.stringPointer("navigator_id"),
			ProjectId:              values.stringPointer("project_id"),
			Locale:                 values.stringPointer("locale"),
			EnableMessageTemplates: values.boolPointer("enable_message_templates"),
		}
	case messageMediaCredentialsBlock:
		body.MessageMediaCredentials = &authsignal.MessageMediaCredentialsCreate{
			ApiKey:       values.stringValue("api_key"),
			ApiSecret:    values.stringValue("api_secret"),
			SourceNumber: values.stringPointer("source_number"),
		}
	case modicaGroupCredentialsBlock:
		body.ModicaGroupCredentials = &authsignal.ModicaGroupCredentialsCreate{
			Username: values.stringValue("username"),
			Password: values.stringValue("password"),
		}
	case tnzCredentialsBlock:
		body.TnzCredentials = &authsignal.TnzCredentialsCreate{
			ApiKey: values.stringValue("api_key"),
		}
	}
}

// Values are read from the configuration rather than the plan: Terraform proposes the prior value for an
// omitted Optional+Computed attribute, so only the configuration can tell an omission from a restatement.
// This holds for the update body of all four resources.
func smsUpdateBody(ctx context.Context, plan smsAuthenticatorConfigurationResourceModel, config smsAuthenticatorConfigurationResourceModel, state smsAuthenticatorConfigurationResourceModel) (authsignal.UpdateSmsAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	var body authsignal.UpdateSmsAuthenticatorConfigurationBody

	providerChanging := !plan.SmsProvider.Equal(state.SmsProvider)
	provider := plan.SmsProvider.ValueString()

	if providerChanging {
		body.SmsProvider = authsignal.SetValue(provider)
	}

	if isKnownAndSet(config.IsActive) {
		body.IsActive = authsignal.SetValue(config.IsActive.ValueBool())
	}

	if isKnownAndSet(config.DefaultCountryCode) {
		body.DefaultCountryCode = authsignal.SetValue(config.DefaultCountryCode.ValueString())
	}

	if isKnownAndSet(config.SmsCountryCodes) {
		countryCodes, setDiags := sortedStringsFrom(ctx, config.SmsCountryCodes)
		diags.Append(setDiags...)
		body.SmsCountryCodes = authsignal.SetList(countryCodes)
	}

	if isKnownAndSet(config.AllowedCustomSmsVariables) {
		variables, setDiags := sortedStringsFrom(ctx, config.AllowedCustomSmsVariables)
		diags.Append(setDiags...)
		body.AllowedCustomSmsVariables = authsignal.SetList(variables)
	}

	if isKnownAndSet(config.PrefixRateLimitConfigurations) {
		prefixes, prefixDiags := prefixRateLimitsFrom(ctx, config.PrefixRateLimitConfigurations)
		diags.Append(prefixDiags...)
		body.PrefixRateLimitConfigurations = authsignal.SetList(prefixes)
	}

	if isKnownAndSet(config.SubmissionRateLimitConfiguration) {
		submission, submissionDiags := rateLimitFrom(ctx, config.SubmissionRateLimitConfiguration)
		diags.Append(submissionDiags...)
		if submission != nil {
			body.SubmissionRateLimitConfiguration = authsignal.SetValue(*submission)
		}
	}

	if isKnownAndSet(config.SendingRateLimitConfigurations) {
		sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
		diags.Append(sendingDiags...)
		if sending != nil {
			body.SendingRateLimitConfigurations = authsignal.SetValue(*sending)
		}
	}

	for _, block := range smsCredentialBlocks {
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
			applySmsUpdateCredentials(&body, block, values, decision.IncludeSecrets)
		}
	}

	return body, diags
}

func applySmsUpdateCredentials(body *authsignal.UpdateSmsAuthenticatorConfigurationBody, block credentialBlock, values credentialValues, includeSecrets bool) {
	secret := func(name string) *string {
		if !includeSecrets {
			return nil
		}

		return values.stringPointer(name)
	}

	switch block.Name {
	case twilioCredentialsBlock:
		body.TwilioCredentials = authsignal.SetValue(authsignal.TwilioCredentialsUpdate{
			AuthToken:           secret("auth_token"),
			MessagingServiceSid: values.stringPointer("messaging_service_sid"),
			AccountSid:          values.stringPointer("account_sid"),
		})
	case birdSmsCredentialsBlock:
		body.BirdSmsCredentials = authsignal.SetValue(authsignal.BirdSmsCredentialsUpdate{
			AccessKey:              secret("access_key"),
			WorkspaceId:            values.stringPointer("workspace_id"),
			ChannelId:              values.stringPointer("channel_id"),
			NavigatorId:            values.stringPointer("navigator_id"),
			ProjectId:              values.stringPointer("project_id"),
			Locale:                 values.stringPointer("locale"),
			EnableMessageTemplates: values.boolPointer("enable_message_templates"),
		})
	case messageMediaCredentialsBlock:
		body.MessageMediaCredentials = authsignal.SetValue(authsignal.MessageMediaCredentialsUpdate{
			ApiKey:       secret("api_key"),
			ApiSecret:    secret("api_secret"),
			SourceNumber: values.stringPointer("source_number"),
		})
	case modicaGroupCredentialsBlock:
		body.ModicaGroupCredentials = authsignal.SetValue(authsignal.ModicaGroupCredentialsUpdate{
			Username: values.stringPointer("username"),
			Password: secret("password"),
		})
	case tnzCredentialsBlock:
		body.TnzCredentials = authsignal.SetValue(authsignal.TnzCredentialsUpdate{
			ApiKey: secret("api_key"),
		})
	}
}

func smsStateAfterWrite(plan smsAuthenticatorConfigurationResourceModel, response *authsignal.SmsAuthenticatorConfiguration) smsAuthenticatorConfigurationResourceModel {
	state := plan
	state.AuthenticatorId = types.StringValue(response.AuthenticatorId)
	state.VerificationMethod = types.StringValue(response.VerificationMethod)

	state.IsActive = resolvedValue(plan.IsActive, types.BoolValue(response.IsActive))
	state.SmsCountryCodes = resolvedValue(plan.SmsCountryCodes, stringSetPointerValue(response.SmsCountryCodes))
	state.DefaultCountryCode = resolvedValue(plan.DefaultCountryCode, types.StringPointerValue(response.DefaultCountryCode))
	state.SubmissionRateLimitConfiguration = resolvedValue(plan.SubmissionRateLimitConfiguration, rateLimitValue(response.SubmissionRateLimitConfiguration))
	state.SendingRateLimitConfigurations = resolvedValue(plan.SendingRateLimitConfigurations, rateLimitListValue(response.SendingRateLimitConfigurations))
	state.PrefixRateLimitConfigurations = resolvedValue(plan.PrefixRateLimitConfigurations, prefixRateLimitListValue(response.PrefixRateLimitConfigurations))
	state.AllowedCustomSmsVariables = resolvedValue(plan.AllowedCustomSmsVariables, stringSetPointerValue(response.AllowedCustomSmsVariables))

	state.TwilioCredentials = credentialPlannedState(twilioBlock, plan.TwilioCredentials)
	state.BirdSmsCredentials = credentialPlannedState(birdSmsBlock, plan.BirdSmsCredentials)
	state.MessageMediaCredentials = credentialPlannedState(messageMediaBlock, plan.MessageMediaCredentials)
	state.ModicaGroupCredentials = credentialPlannedState(modicaGroupBlock, plan.ModicaGroupCredentials)
	state.TnzCredentials = credentialPlannedState(tnzBlock, plan.TnzCredentials)

	return state
}

func smsStateFromResponse(response *authsignal.SmsAuthenticatorConfiguration, prior smsAuthenticatorConfigurationResourceModel) smsAuthenticatorConfigurationResourceModel {
	state := smsAuthenticatorConfigurationResourceModel{
		AuthenticatorId:                  types.StringValue(response.AuthenticatorId),
		VerificationMethod:               types.StringValue(response.VerificationMethod),
		IsActive:                         types.BoolValue(response.IsActive),
		SmsProvider:                      types.StringPointerValue(response.SmsProvider),
		SmsCountryCodes:                  stringSetPointerValue(response.SmsCountryCodes),
		DefaultCountryCode:               types.StringPointerValue(response.DefaultCountryCode),
		SubmissionRateLimitConfiguration: rateLimitValue(response.SubmissionRateLimitConfiguration),
		SendingRateLimitConfigurations:   rateLimitListValue(response.SendingRateLimitConfigurations),
		PrefixRateLimitConfigurations:    prefixRateLimitListValue(response.PrefixRateLimitConfigurations),
		AllowedCustomSmsVariables:        stringSetPointerValue(response.AllowedCustomSmsVariables),

		TwilioCredentialsVersion:       prior.TwilioCredentialsVersion,
		BirdSmsCredentialsVersion:      prior.BirdSmsCredentialsVersion,
		MessageMediaCredentialsVersion: prior.MessageMediaCredentialsVersion,
		ModicaGroupCredentialsVersion:  prior.ModicaGroupCredentialsVersion,
		TnzCredentialsVersion:          prior.TnzCredentialsVersion,
	}

	applySmsCredentialReadState(&state, response, prior)

	return state
}

func applySmsCredentialReadState(state *smsAuthenticatorConfigurationResourceModel, response *authsignal.SmsAuthenticatorConfiguration, prior smsAuthenticatorConfigurationResourceModel) {
	twilioMetadata := map[string]attr.Value{}
	if response.TwilioCredentials != nil {
		twilioMetadata = map[string]attr.Value{
			"messaging_service_sid": types.StringPointerValue(response.TwilioCredentials.MessagingServiceSid),
			"account_sid":           types.StringPointerValue(response.TwilioCredentials.AccountSid),
		}
	}
	state.TwilioCredentials = credentialReadState(twilioBlock, twilioMetadata, response.TwilioCredentials != nil, prior.TwilioCredentials)

	birdMetadata := map[string]attr.Value{}
	if response.BirdSmsCredentials != nil {
		birdMetadata = map[string]attr.Value{
			"workspace_id":             types.StringPointerValue(response.BirdSmsCredentials.WorkspaceId),
			"channel_id":               types.StringPointerValue(response.BirdSmsCredentials.ChannelId),
			"navigator_id":             types.StringPointerValue(response.BirdSmsCredentials.NavigatorId),
			"project_id":               types.StringPointerValue(response.BirdSmsCredentials.ProjectId),
			"locale":                   types.StringPointerValue(response.BirdSmsCredentials.Locale),
			"enable_message_templates": types.BoolPointerValue(response.BirdSmsCredentials.EnableMessageTemplates),
		}
	}
	state.BirdSmsCredentials = credentialReadState(birdSmsBlock, birdMetadata, response.BirdSmsCredentials != nil, prior.BirdSmsCredentials)

	messageMediaMetadata := map[string]attr.Value{}
	if response.MessageMediaCredentials != nil {
		messageMediaMetadata = map[string]attr.Value{
			"source_number": types.StringPointerValue(response.MessageMediaCredentials.SourceNumber),
		}
	}
	state.MessageMediaCredentials = credentialReadState(messageMediaBlock, messageMediaMetadata, response.MessageMediaCredentials != nil, prior.MessageMediaCredentials)

	modicaMetadata := map[string]attr.Value{}
	if response.ModicaGroupCredentials != nil {
		modicaMetadata = map[string]attr.Value{
			"username": types.StringPointerValue(response.ModicaGroupCredentials.Username),
		}
	}
	state.ModicaGroupCredentials = credentialReadState(modicaGroupBlock, modicaMetadata, response.ModicaGroupCredentials != nil, prior.ModicaGroupCredentials)

	state.TnzCredentials = credentialReadState(tnzBlock, nil, response.TnzCredentials != nil, prior.TnzCredentials)
}
